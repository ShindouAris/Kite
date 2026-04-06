package billing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/kitecloud/kite/kite-service/internal/api/handler"
	"github.com/kitecloud/kite/kite-service/internal/api/handler/billing/payment"
	"github.com/kitecloud/kite/kite-service/internal/api/wire"
	"github.com/kitecloud/kite/kite-service/internal/util"
)

func (h *BillingHandler) HandleAppCheckout(c *handler.Context, req wire.BillingCheckoutRequest) (*wire.BillingCheckoutResponse, error) {
	planID := strings.TrimSpace(req.PlanID)

	plan := h.planManager.PlanByID(planID)
	if plan == nil {
		return nil, handler.ErrBadRequest("unknown_plan", "Unknown plan")
	}

	amount := plan.PaymentAmount
	if amount <= 0 {
		amount = int(plan.Price)
	}

	if h.config.SePayMerchantID == "" {
		return nil, fmt.Errorf("sepay merchant id is not configured")
	}

	if h.config.SePaySecretKey == "" {
		return nil, fmt.Errorf("sepay secret key is not configured")
	}

	invoiceNumber := payment.EncodeInvoiceNumber(c.App.ID, plan.ID, util.UniqueID())
	successURL := h.sepayReturnURL(c.App.ID, plan.ID, invoiceNumber, "success")
	errorURL := h.sepayReturnURL(c.App.ID, plan.ID, invoiceNumber, "error")
	cancelURL := h.sepayReturnURL(c.App.ID, plan.ID, invoiceNumber, "cancel")
	fields := []wire.BillingCheckoutField{
		{Name: "merchant", Value: h.config.SePayMerchantID},
		{Name: "currency", Value: "VND"},
		{Name: "order_amount", Value: strconv.Itoa(amount)},
		{Name: "operation", Value: "PURCHASE"},
		{Name: "order_description", Value: strings.ReplaceAll(fmt.Sprintf("Thanh toan goi %s cho %s", plan.Title, c.App.Name), ",", " ")},
		{Name: "order_invoice_number", Value: invoiceNumber},
		{Name: "success_url", Value: successURL},
		{Name: "error_url", Value: errorURL},
		{Name: "cancel_url", Value: cancelURL},
	}
	fields = append(fields, wire.BillingCheckoutField{Name: "signature", Value: signSePayCheckout(fields, h.config.SePaySecretKey)})

	return &wire.BillingCheckoutResponse{
		ActionURL:          strings.TrimRight(h.config.SePayCheckoutBaseURL, "/") + "/v1/checkout/init",
		Method:             "POST",
		PaymentID:          invoiceNumber,
		OrderInvoiceNumber: invoiceNumber,
		SuccessURL:         successURL,
		ErrorURL:           errorURL,
		CancelURL:          cancelURL,
		Fields:             fields,
	}, nil
}

func (h *BillingHandler) sepayReturnURL(appID, planID, invoiceNumber, status string) string {
	baseURL := strings.TrimRight(h.config.AppPublicBaseURL, "/")
	query := url.Values{}
	query.Set("payment", status)
	query.Set("plan_id", planID)
	query.Set("invoice", invoiceNumber)
	return fmt.Sprintf("%s/apps/%s/premium?%s", baseURL, appID, query.Encode())
}

func signSePayCheckout(fields []wire.BillingCheckoutField, secretKey string) string {
	orderedNames := []string{
		"order_amount",
		"merchant",
		"currency",
		"operation",
		"order_description",
		"order_invoice_number",
		"customer_id",
		"payment_method",
		"success_url",
		"error_url",
		"cancel_url",
	}
	valueByName := make(map[string]string, len(fields))
	for _, field := range fields {
		valueByName[field.Name] = field.Value
	}

	signedParts := make([]string, 0, len(orderedNames))
	for _, name := range orderedNames {
		if value, ok := valueByName[name]; ok && value != "" {
			signedParts = append(signedParts, name+"="+value)
		}
	}

	mac := hmac.New(sha256.New, []byte(secretKey))
	_, _ = mac.Write([]byte(strings.Join(signedParts, ",")))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
