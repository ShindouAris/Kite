package wire

import (
	"time"

	"github.com/kitecloud/kite/kite-service/internal/model"
	"gopkg.in/guregu/null.v4"
)

type BillingWebhookRequest struct {
	Meta struct {
		EventName  string                 `json:"event_name"`
		CustomData map[string]interface{} `json:"custom_data"`
	} `json:"meta"`
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			StoreID         int       `json:"store_id"`
			CustomerID      int       `json:"customer_id"`
			OrderID         int       `json:"order_id"`
			OrderItemID     int       `json:"order_item_id"`
			ProductID       int       `json:"product_id"`
			VariantID       int       `json:"variant_id"`
			ProductName     string    `json:"product_name"`
			VariantName     string    `json:"variant_name"`
			UserName        string    `json:"user_name"`
			UserEmail       string    `json:"user_email"`
			Status          string    `json:"status"`
			StatusFormatted string    `json:"status_formatted"`
			CardBrand       string    `json:"card_brand"`
			CardLastFour    string    `json:"card_last_four"`
			Cancelled       bool      `json:"cancelled"`
			TrialEndsAt     null.Time `json:"trial_ends_at"`
			BillingAnchor   int       `json:"billing_anchor"`
			RenewsAt        time.Time `json:"renews_at"`
			EndsAt          null.Time `json:"ends_at"`
			CreatedAt       time.Time `json:"created_at"`
			UpdatedAt       time.Time `json:"updated_at"`
			TestMode        bool      `json:"test_mode"`
		} `json:"attributes"`
	} `json:"data"`
}

type BillingWebhookResponse struct{}

type BillingCheckoutRequest struct {
	PlanID                string `json:"plan_id"`
	LemonSqueezyVariantID string `json:"lemonsqueezy_variant_id"`
}

type BillingCheckoutField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type BillingCheckoutResponse struct {
	ActionURL          string                 `json:"action_url"`
	Method             string                 `json:"method"`
	PaymentID          string                 `json:"payment_id"`
	OrderInvoiceNumber string                 `json:"order_invoice_number"`
	SuccessURL         string                 `json:"success_url"`
	ErrorURL           string                 `json:"error_url"`
	CancelURL          string                 `json:"cancel_url"`
	Fields             []BillingCheckoutField `json:"fields"`
}

type BillingCheckoutStatusResponse struct {
	PaymentID           string `json:"payment_id"`
	Status              string `json:"status"`
	Paid                bool   `json:"paid"`
	Amount              int    `json:"amount"`
	SubscriptionCreated bool   `json:"subscription_created"`
}

type BillingPaymentWebhookRequest struct {
	RefNo           string `json:"refNo"`
	Amount          string `json:"amount"`
	TransactionDate string `json:"transactionDate"`
	PostingDate     string `json:"postingDate"`
	Description     string `json:"description"`
	Sender          string `json:"sender"`
	SenderAccountNo string `json:"senderAccoundNo"`
}

type BillingSePayIPNRequest struct {
	Timestamp        int64  `json:"timestamp"`
	NotificationType string `json:"notification_type"`
	Order            struct {
		ID                 string `json:"id"`
		OrderID            string `json:"order_id"`
		OrderStatus        string `json:"order_status"`
		OrderCurrency      string `json:"order_currency"`
		OrderAmount        string `json:"order_amount"`
		OrderInvoiceNumber string `json:"order_invoice_number"`
		OrderDescription   string `json:"order_description"`
	} `json:"order"`
	Transaction struct {
		ID                  string `json:"id"`
		PaymentMethod       string `json:"payment_method"`
		TransactionID       string `json:"transaction_id"`
		TransactionType     string `json:"transaction_type"`
		TransactionDate     string `json:"transaction_date"`
		TransactionStatus   string `json:"transaction_status"`
		TransactionAmount   string `json:"transaction_amount"`
		TransactionCurrency string `json:"transaction_currency"`
	} `json:"transaction"`
	Customer struct {
		ID         string `json:"id"`
		CustomerID string `json:"customer_id"`
	} `json:"customer"`
}

type SubscriptionManageResponse struct {
	UpdatePaymentMethodURL string `json:"update_payment_method_url"`
	CustomerPortalURL      string `json:"customer_portal_url"`
}

type Subscription struct {
	ID                         string      `json:"id"`
	DisplayName                string      `json:"display_name"`
	PlanID                     string      `json:"plan_id"`
	Source                     string      `json:"source"`
	Status                     string      `json:"status"`
	StatusFormatted            string      `json:"status_formatted"`
	CreatedAt                  time.Time   `json:"created_at"`
	UpdatedAt                  time.Time   `json:"updated_at"`
	RenewsAt                   time.Time   `json:"renews_at"`
	TrialEndsAt                null.Time   `json:"trial_ends_at"`
	EndsAt                     null.Time   `json:"ends_at"`
	UserID                     string      `json:"user_id"`
	LemonsqueezySubscriptionID null.String `json:"lemonsqueezy_subscription_id"`
	LemonsqueezyCustomerID     null.String `json:"lemonsqueezy_customer_id"`
	LemonsqueezyOrderID        null.String `json:"lemonsqueezy_order_id"`
	LemonsqueezyProductID      null.String `json:"lemonsqueezy_product_id"`
	LemonsqueezyVariantID      null.String `json:"lemonsqueezy_variant_id"`
	Manageable                 bool        `json:"manageable"`
}

type SubscriptionListResponse = []*Subscription

func SubscriptionToWire(subscription *model.Subscription, userID string) *Subscription {
	if subscription == nil {
		return nil
	}

	return &Subscription{
		ID:                         subscription.ID,
		DisplayName:                subscription.DisplayName,
		PlanID:                     subscription.LemonsqueezyProductID.String,
		Source:                     string(subscription.Source),
		Status:                     subscription.Status,
		StatusFormatted:            subscription.StatusFormatted,
		CreatedAt:                  subscription.CreatedAt,
		UpdatedAt:                  subscription.UpdatedAt,
		RenewsAt:                   subscription.RenewsAt,
		TrialEndsAt:                subscription.TrialEndsAt,
		EndsAt:                     subscription.EndsAt,
		UserID:                     subscription.UserID,
		LemonsqueezySubscriptionID: subscription.LemonsqueezySubscriptionID,
		LemonsqueezyCustomerID:     subscription.LemonsqueezyCustomerID,
		LemonsqueezyOrderID:        subscription.LemonsqueezyOrderID,
		LemonsqueezyProductID:      subscription.LemonsqueezyProductID,
		LemonsqueezyVariantID:      subscription.LemonsqueezyVariantID,
		Manageable:                 subscription.UserID == userID && subscription.Source == model.SubscriptionSourceLemonSqueezy && subscription.LemonsqueezySubscriptionID.Valid,
	}
}

type BillingPlan struct {
	ID                  string  `json:"id"`
	Title               string  `json:"title"`
	Description         string  `json:"description"`
	Price               float32 `json:"price"`
	PaymentAmount       int     `json:"payment_amount"`
	PremiumDurationDays int     `json:"premium_duration_days"`
	Default             bool    `json:"default"`
	Popular             bool    `json:"popular"`
	Hidden              bool    `json:"hidden"`

	LemonSqueezyProductID string `json:"lemonsqueezy_product_id"`
	LemonSqueezyVariantID string `json:"lemonsqueezy_variant_id"`

	DiscordRoleID string `json:"discord_role_id"`

	FeatureMaxCollaborators     int  `json:"feature_max_collaborators"`
	FeatureUsageCreditsPerMonth int  `json:"feature_usage_credits_per_month"`
	FeatureMaxGuilds            int  `json:"feature_max_guilds"`
	FeatureMaxCommands          int  `json:"feature_max_commands"`
	FeatureMaxVariables         int  `json:"feature_max_variables"`
	FeatureMaxMessages          int  `json:"feature_max_messages"`
	FeatureMaxEventListeners    int  `json:"feature_max_event_listeners"`
	FeaturePrioritySupport      bool `json:"feature_priority_support"`
}

type BillingPlanListResponse = []*BillingPlan
