package billing

import (
	"github.com/kitecloud/kite/kite-service/internal/core/plan"
	"github.com/kitecloud/kite/kite-service/internal/store"
)

type BillingHandlerConfig struct {
	WebhookHMACSecret  string
	TransferCodePrefix string
	MerchantBankName   string
	MerchantAccountNo  string
	CheckoutTTLMinutes int
	AppPublicBaseURL   string
}

type BillingHandler struct {
	config            BillingHandlerConfig
	appStore          store.AppStore
	userStore         store.UserStore
	subscriptionStore store.SubscriptionStore
	entitlementStore  store.EntitlementStore
	planManager       *plan.PlanManager
}

func NewBillingHandler(
	config BillingHandlerConfig,
	appStore store.AppStore,
	userStore store.UserStore,
	subscriptionStore store.SubscriptionStore,
	entitlementStore store.EntitlementStore,
	planManager *plan.PlanManager,
) *BillingHandler {
	if config.TransferCodePrefix == "" {
		config.TransferCodePrefix = "KITE"
	}

	if config.CheckoutTTLMinutes <= 0 {
		config.CheckoutTTLMinutes = 30
	}

	return &BillingHandler{
		config:            config,
		appStore:          appStore,
		userStore:         userStore,
		subscriptionStore: subscriptionStore,
		entitlementStore:  entitlementStore,
		planManager:       planManager,
	}
}
