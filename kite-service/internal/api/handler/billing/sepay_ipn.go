package billing

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kitecloud/kite/kite-service/internal/api/handler"
	"github.com/kitecloud/kite/kite-service/internal/api/handler/billing/payment"
	"github.com/kitecloud/kite/kite-service/internal/api/wire"
	"github.com/kitecloud/kite/kite-service/internal/model"
	"github.com/kitecloud/kite/kite-service/internal/util"
	"gopkg.in/guregu/null.v4"
)

func (h *BillingHandler) HandleSePayIPN(c *handler.Context, body json.RawMessage) (*wire.BillingWebhookResponse, error) {
	if strings.TrimSpace(c.Header("X-Secret-Key")) != strings.TrimSpace(h.config.SePaySecretKey) {
		return nil, handler.ErrUnauthorized("unauthorized", "invalid sepay secret key")
	}

	var req wire.BillingSePayIPNRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, handler.ErrBadRequest("invalid_request", fmt.Sprintf("failed to decode sepay ipn: %v", err))
	}

	if !strings.EqualFold(req.NotificationType, "ORDER_PAID") {
		return &wire.BillingWebhookResponse{}, nil
	}

	code, ok := payment.DecodeInvoiceNumber(req.Order.OrderInvoiceNumber)
	if !ok {
		return nil, handler.ErrBadRequest("invalid_invoice_number", "failed to parse order invoice number")
	}

	plan := h.planManager.PlanByID(code.PlanID)
	if plan == nil {
		return nil, handler.ErrBadRequest("unknown_plan", "Unknown plan")
	}

	amount, err := payment.ParseAmountVND(req.Order.OrderAmount)
	if err != nil {
		return nil, handler.ErrBadRequest("invalid_amount", fmt.Sprintf("failed to parse order amount: %v", err))
	}

	expectedAmount := plan.PaymentAmount
	if expectedAmount <= 0 {
		expectedAmount = int(plan.Price)
	}
	if amount != expectedAmount {
		return nil, handler.ErrBadRequest("amount_mismatch", fmt.Sprintf("expected %d got %d", expectedAmount, amount))
	}

	app, err := h.appStore.App(c.Context(), code.AppID)
	if err != nil {
		return nil, fmt.Errorf("failed to load app: %w", err)
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
		UserID:                     app.OwnerUserID,
		LemonsqueezySubscriptionID: null.StringFrom(req.Order.OrderInvoiceNumber),
		LemonsqueezyCustomerID:     null.String{},
		LemonsqueezyOrderID:        null.StringFrom(req.Order.OrderID),
		LemonsqueezyProductID:      null.StringFrom(plan.ID),
		LemonsqueezyVariantID:      null.String{},
	})
	if err != nil {
		slog.Error(
			"Failed to upsert sepay subscription",
			slog.String("order_invoice_number", req.Order.OrderInvoiceNumber),
			slog.String("error", err.Error()),
		)
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
		AppID:          app.ID,
		PlanID:         plan.ID,
		CreatedAt:      now,
		UpdatedAt:      now,
		EndsAt:         entitlementEndsAt,
	}

	_, err = h.entitlementStore.UpsertSubscriptionEntitlement(c.Context(), entitlement)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert subscription entitlement: %w", err)
	}

	return &wire.BillingWebhookResponse{}, nil
}
