package billing

import (
	"github.com/kitecloud/kite/kite-service/internal/api/handler"
	"github.com/kitecloud/kite/kite-service/internal/api/wire"
)

func (h *BillingHandler) HandleAppSubscriptionList(c *handler.Context) (*wire.SubscriptionListResponse, error) {
	subscriptions, err := h.subscriptionStore.SubscriptionsByAppID(c.Context(), c.App.ID)
	if err != nil {
		return nil, err
	}

	res := make(wire.SubscriptionListResponse, len(subscriptions))
	for i, subscription := range subscriptions {
		res[i] = wire.SubscriptionToWire(subscription, c.Session.UserID)
	}

	return &res, nil
}

func (h *BillingHandler) HandleSubscriptionManage(c *handler.Context) (*wire.SubscriptionManageResponse, error) {
	return nil, handler.ErrNotFound("unmanageable_subscription", "Subscription can not be managed")
}
