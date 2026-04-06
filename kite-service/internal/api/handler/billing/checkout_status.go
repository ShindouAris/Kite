package billing

import (
	"fmt"
	"strings"
	"time"

	"github.com/kitecloud/kite/kite-service/internal/api/handler"
	"github.com/kitecloud/kite/kite-service/internal/api/handler/billing/payment"
	"github.com/kitecloud/kite/kite-service/internal/api/wire"
	"github.com/kitecloud/kite/kite-service/internal/model"
	"github.com/kitecloud/kite/kite-service/internal/util"
	"gopkg.in/guregu/null.v4"
)

func (h *BillingHandler) HandleAppCheckoutStatus(c *handler.Context) (*wire.BillingCheckoutStatusResponse, error) {
	paymentID := strings.TrimSpace(c.Param("paymentID"))
	planID := strings.TrimSpace(c.Query("plan_id"))
	if paymentID == "" || planID == "" {
		return nil, handler.ErrBadRequest("invalid_request", "payment_id and plan_id are required")
	}

	invoiceParts, ok := payment.DecodeInvoiceNumber(paymentID)
	if !ok {
		return nil, handler.ErrBadRequest("invalid_invoice", "invalid invoice number")
	}

	if invoiceParts.PlanID != planID {
		return nil, handler.ErrBadRequest("plan_mismatch", "plan_id does not match invoice")
	}

	if h.config.SePayBankAccountXID == "" {
		return nil, fmt.Errorf("sepay bank account xid is not configured")
	}

	plan := h.planManager.PlanByID(planID)
	if plan == nil {
		return nil, handler.ErrBadRequest("unknown_plan", "Unknown plan")
	}

	order, err := h.sepay.GetOrder(c.Context(), h.config.SePayBankAccountXID, paymentID)
	if err != nil {
		return nil, err
	}

	status := strings.TrimSpace(order.Status)
	paid := strings.EqualFold(status, "Paid") || strings.EqualFold(status, "Captured")
	if !paid {
		return &wire.BillingCheckoutStatusResponse{
			PaymentID: paymentID,
			Status:    status,
			Paid:      false,
			Amount:    order.Amount,
		}, nil
	}

	expectedAmount := plan.PaymentAmount
	if expectedAmount <= 0 {
		expectedAmount = int(plan.Price)
	}
	if order.Amount != expectedAmount {
		return nil, fmt.Errorf("amount mismatch: expected %d got %d", expectedAmount, order.Amount)
	}

	now := time.Now().UTC()
	renewsAt := now.AddDate(50, 0, 0)
	subscription, err := h.subscriptionStore.UpsertLemonSqueezySubscription(c.Context(), model.Subscription{
		ID:                         util.UniqueID(),
		DisplayName:                plan.Title,
		Source:                     model.SubscriptionSourceSePay,
		Status:                     "active",
		StatusFormatted:            "Active",
		RenewsAt:                   renewsAt,
		TrialEndsAt:                null.Time{},
		EndsAt:                     null.Time{},
		CreatedAt:                  now,
		UpdatedAt:                  now,
		UserID:                     c.App.OwnerUserID,
		LemonsqueezySubscriptionID: null.StringFrom(paymentID),
		LemonsqueezyCustomerID:     null.String{},
		LemonsqueezyOrderID:        null.StringFrom(order.OrderCode),
		LemonsqueezyProductID:      null.StringFrom(plan.ID),
		LemonsqueezyVariantID:      null.String{},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to upsert subscription: %w", err)
	}

	entitlementEndsAt := null.Time{}
	if plan.PremiumDurationDays > 0 {
		entitlementEndsAt = null.TimeFrom(now.AddDate(0, 0, plan.PremiumDurationDays))
	}

	entitlement := model.Entitlement{
		ID:             util.UniqueID(),
		Type:           "subscription",
		SubscriptionID: null.StringFrom(subscription.ID),
		AppID:          c.App.ID,
		PlanID:         plan.ID,
		CreatedAt:      now,
		UpdatedAt:      now,
		EndsAt:         entitlementEndsAt,
	}

	_, err = h.entitlementStore.UpsertSubscriptionEntitlement(c.Context(), entitlement)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert subscription entitlement: %w", err)
	}

	return &wire.BillingCheckoutStatusResponse{
		PaymentID:           paymentID,
		Status:              status,
		Paid:                true,
		Amount:              order.Amount,
		SubscriptionCreated: true,
	}, nil
}
