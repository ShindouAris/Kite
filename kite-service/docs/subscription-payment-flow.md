# Phân Tích Luồng Thanh Toán Subscription

## Tổng Quan

Hệ thống thanh toán subscription hiện dùng luồng **SePay QR + webhook** làm đường chính.

Các mục **Manual/Webhook** và **LemonSqueezy** bên dưới chỉ còn để tham khảo lịch sử; chúng không nằm trong luồng thanh toán mới.

---

## Luồng Thanh Toán Chi Tiết

### 1. Khởi Tạo Checkout (Bước 1)

**Endpoint:** `POST /v1/apps/{appID}/billing/checkout`

**Handler:** `internal/api/handler/billing/checkout.go:17`

```go
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
    // ... tạo invoice number và các field checkout
}
```

**Flow chi tiết:**

1. **Validate Plan**: Kiểm tra plan tồn tại qua `planManager.PlanByID(planID)`
2. **Tạo Invoice Number**: Sinh unique invoice theo format:
   ```
   {TransferCodePrefix}-{AppID}-{PlanID}-{UniqueID}
   ```
   VD: `KITE-app123-plan-basic-abc123`

3. **Tạo Return URLs**: 
   - `success_url`: `/{baseURL}/apps/{appID}/premium?payment=success&plan_id={planID}&invoice={invoiceNumber}`
   - `error_url`: `/{baseURL}/apps/{appID}/premium?payment=error&plan_id={planID}&invoice={invoiceNumber}`
   - `cancel_url`: `/{baseURL}/apps/{appID}/premium?payment=cancel&plan_id={planID}&invoice={invoiceNumber}`

4. **Sign Data**: Tạo HMAC-SHA256 signature cho checkout request

**Response trả về** (`internal/api/wire/billing.go:56`):
```go
type BillingCheckoutResponse struct {
    ActionURL          string                 // URL của SePay để redirect
    Method             string                 // "POST"
    PaymentID          string                 // Invoice number
    OrderInvoiceNumber string                 // Invoice number
    SuccessURL         string
    ErrorURL           string
    CancelURL          string
    Fields             []BillingCheckoutField // Các field ẩn cho form POST
}
```

---

### 2. Người Dùng Thanh Toán

Người dùng được redirect đến `ActionURL` của SePay với method POST, chứa các field:
- `merchant`: SePay Merchant ID
- `currency`: "VND"
- `order_amount`: Số tiền
- `operation`: "PURCHASE"
- `order_description`: Mô tả đơn hàng
- `order_invoice_number`: Invoice number đã tạo
- `success_url`, `error_url`, `cancel_url`: Các URL callback
- `signature`: HMAC signature

Người dùng hoàn tất thanh toán trên cổng SePay.

---

### 3. Kiểm Tra Trạng Thái Thanh Toán (Bước 2)

**Endpoint:** `GET /v1/apps/{appID}/billing/checkouts/{paymentID}?plan_id={planID}`

**Handler:** `internal/api/handler/billing/checkout_status.go:15`

```go
func (h *BillingHandler) HandleAppCheckoutStatus(c *handler.Context) (*wire.BillingCheckoutStatusResponse, error) {
    paymentID := strings.TrimSpace(c.Param("paymentID"))
    planID := strings.TrimSpace(c.Query("plan_id"))
    
    // 1. Validate input
    if paymentID == "" || planID == "" {
        return nil, handler.ErrBadRequest("invalid_request", "payment_id and plan_id are required")
    }
    
    // 2. Lấy thông tin order từ SePay API
    order, err := h.sepay.GetOrder(c.Context(), h.config.SePayBankAccountXID, paymentID)
    if err != nil {
        return nil, err
    }
    
    // 3. Kiểm tra trạng thái thanh toán
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
    
    // 4. Verify số tiền
    expectedAmount := plan.PaymentAmount
    if expectedAmount <= 0 {
        expectedAmount = int(plan.Price)
    }
    if order.Amount != expectedAmount {
        return nil, fmt.Errorf("amount mismatch: expected %d got %d", expectedAmount, order.Amount)
    }
    
    // 5. Tạo subscription
    subscription, err := h.subscriptionStore.UpsertLemonSqueezySubscription(c.Context(), model.Subscription{
        ID:                         util.UniqueID(),
        DisplayName:                plan.Title,
        Source:                     model.SubscriptionSourceSePay,
        Status:                     "active",
        StatusFormatted:            "Active",
        RenewsAt:                   renewsAt,
        // ... các field khác
    })
    
    // 6. Tạo entitlement
    entitlement := model.Entitlement{
        ID:             util.UniqueID(),
        Type:           "subscription",
        SubscriptionID: null.StringFrom(subscription.ID),
        AppID:          c.App.ID,
        PlanID:         plan.ID,
        // ...
    }
    
    _, err = h.entitlementStore.UpsertSubscriptionEntitlement(c.Context(), entitlement)
}
```

**Flow chi tiết:**

1. **Validate Request**: Kiểm tra `paymentID` và `planID` không rỗng

2. **Gọi SePay API**: Sử dụng `sepayClient.GetOrder()` để lấy trạng thái order
   - **File**: `internal/api/handler/billing/sepay.go:63`
   ```go
   func (c *sepayClient) GetOrder(ctx context.Context, bankAccountXID, orderID string) (*sepayOrder, error) {
       return c.do(ctx, http.MethodGet, fmt.Sprintf("/bank-accounts/%s/orders/%s", bankAccountXID, orderID), nil)
   }
   ```

3. **Kiểm tra trạng thái**:
   - `Paid` hoặc `Captured` = thanh toán thành công
   - Các trạng thái khác = chưa thanh toán, trả về `Paid: false`

4. **Verify số tiền**: So sánh số tiền thực tế với số tiền expected từ plan

5. **Tạo Subscription** (nếu thanh toán thành công):
   - **Store Interface**: `internal/store/subscription.go:14`
   ```go
   UpsertLemonSqueezySubscription(ctx context.Context, sub model.Subscription) (*model.Subscription, error)
   ```
   - **DB Implementation**: `internal/db/postgres/store_subscription.go:63`
   ```go
   func (c *Client) UpsertLemonSqueezySubscription(ctx context.Context, sub model.Subscription) (*model.Subscription, error) {
       row, err := c.Q.UpsertLemonSqueezySubscription(ctx, pgmodel.UpsertLemonSqueezySubscriptionParams{...})
   }
   ```

6. **Tạo Entitlement**:
   - **Store Interface**: `internal/store/entitlement.go:13`
   ```go
   UpsertSubscriptionEntitlement(ctx context.Context, entitlement model.Entitlement) (*model.Entitlement, error)
   ```
   - **DB Implementation**: `internal/db/postgres/store_entitlement.go:44`
   ```go
   func (c *Client) UpsertSubscriptionEntitlement(ctx context.Context, entitlement model.Entitlement) (*model.Entitlement, error) {
   ```

**Response** (`internal/api/wire/billing.go:67`):
```go
type BillingCheckoutStatusResponse struct {
    PaymentID           string // Invoice number
    Status              string // Trạng thái từ SePay
    Paid                bool   // Đã thanh toán hay chưa
    Amount              int    // Số tiền
    SubscriptionCreated bool   // Subscription đã được tạo chưa
}
```

---

## Các Phương Thức Thanh Toán Khác

### A. SePay IPN (Instant Payment Notification)

**Endpoint:** `POST /v1/billing/sepay/ipn`

**Handler:** `internal/api/handler/billing/sepay_ipn.go:18`

Được SePay gọi khi có thanh toán thành công. Flow tương tự nhưng:
- Parse `order_invoice_number` để lấy `AppID`, `PlanID`, `Nonce`
- Verify secret key từ header `X-Secret-Key`
- Kiểm tra `notification_type == "ORDER_PAID"`

```go
func (h *BillingHandler) HandleSePayIPN(c *handler.Context, body json.RawMessage) (*wire.BillingWebhookResponse, error) {
    // 1. Verify secret key
    if strings.TrimSpace(c.Header("X-Secret-Key")) != strings.TrimSpace(h.config.SePaySecretKey) {
        return nil, handler.ErrUnauthorized("unauthorized", "invalid sepay secret key")
    }
    
    // 2. Parse request
    var req wire.BillingSePayIPNRequest
    // ...
    
    // 3. Chỉ xử lý ORDER_PAID
    if !strings.EqualFold(req.NotificationType, "ORDER_PAID") {
        return &wire.BillingWebhookResponse{}, nil
    }
    
    // 4. Parse transfer code và tạo subscription
    // ...
}
```

### B. Manual Webhook (Chuyển khoản thủ công)

**Endpoint:** `POST /v1/billing/webhook`

**Handler:** `internal/api/handler/billing/webhook.go:17`

Nhận thông báo thanh toán từ các kênh khác (không phải SePay):
- Verify HMAC signature từ header `X-HMAC-Signature`
- Parse `description` để lấy transfer code
- Parse số tiền từ `amount`

```go
func (h *BillingHandler) HandleBillingWebhook(c *handler.Context, body json.RawMessage) (*wire.BillingWebhookResponse, error) {
    // 1. Verify HMAC signature
    signature := c.Header("X-HMAC-Signature")
    if !payment.VerifyHMAC(body, signature, h.config.WebhookHMACSecret) {
        return nil, fmt.Errorf("failed to verify webhook signature")
    }
    
    // 2. Parse request
    var req wire.BillingPaymentWebhookRequest
    // ...
    
    // 3. Parse transfer code từ description
    code, ok := payment.ParseTransferCode(req.Description, h.config.TransferCodePrefix)
    // Format: "{Prefix}-{AppID}-{PlanID}-{Nonce}"
    
    // 4. Verify amount và tạo subscription
    // ...
}
```

### C. Utility Functions

**File:** `internal/api/handler/billing/payment/payment.go`

```go
// Parse transfer code từ description
func ParseTransferCode(description string, prefix string) (*TransferCodeParts, bool) {
    // Format: {prefix}-{app_id}-{plan_id}-{nonce}
    // VD: KITE-app123-basic-xyz
}

// Parse amount VND từ string
func ParseAmountVND(raw string) (int, error) {
    // Xử lý các format: "100,000", "100000", "100.000"
}

// Verify HMAC signature
func VerifyHMAC(payload []byte, signature string, secret string) bool {
    // Verify webhook signature từ LemonSqueezy
}
```

---

## Sơ Đồ Luồng

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        SUBSCRIPTION PAYMENT FLOW                             │
└─────────────────────────────────────────────────────────────────────────────┘

     ┌──────────────┐     ┌─────────────────┐     ┌───────────────────┐
     │   Frontend   │────▶│   API Checkout  │────▶│    SePay Portal   │
     └──────────────┘     │ POST /checkout  │     │                   │
                          └─────────────────┘     └───────────────────┘
                                   │                          │
                                   ▼                          ▼
                          ┌─────────────────┐     ┌───────────────────┐
                          │ Create Invoice  │     │ User Completes    │
                          │ Number & Sign   │     │ Payment           │
                          └─────────────────┘     └───────────────────┘
                                                        │
                                                        ▼
                                         ┌──────────────────────────────┐
                                         │ 3rd Party Polls Checkout      │
                                         │ GET /checkouts/{paymentID}  │
                                         └──────────────────────────────┘
                                                  │
                                                  ▼
                                   ┌──────────────────────────────┐
                                   │    API Calls SePay API       │
                                   │    GetOrder(paymentID)      │
                                   └──────────────────────────────┘
                                                  │
                                    ┌────────────┴────────────┐
                                    │                       │
                                    ▼                       ▼
                            ┌──────────────┐         ┌──────────────┐
                            │  Not Paid   │         │    Paid      │
                            │  (Polling)  │         │              │
                            └──────────────┘         ▼──────────────┘
                                                  ┌────────────────────┐
                                                  │ Validate Amount   │
                                                  └────────────────────┘
                                                          │
                                                          ▼
                                                  ┌────────────────────┐
                                                  │ Create Subscription│
                                                  │ via Store          │
                                                  └────────────────────┘
                                                          │
                                                          ▼
                                                  ┌────────────────────┐
                                                  │ Create Entitlement │
                                                  │ via Store          │
                                                  └────────────────────┘
                                                          │
                                                          ▼
                                                  ┌────────────────────┐
                                                  │ Return Success     │
                                                  │ Response           │
                                                  └────────────────────┘
```

---

## Database Schema

### Subscription Table

**SQL Generated:** `internal/db/postgres/pgmodel/subscriptions.sql.go`

```sql
CREATE TABLE subscriptions (
    id                          TEXT PRIMARY KEY,
    display_name                TEXT NOT NULL,
    source                      TEXT NOT NULL,
    status                      TEXT NOT NULL,
    status_formatted            TEXT NOT NULL,
    created_at                  TIMESTAMPTZ NOT NULL,
    updated_at                  TIMESTAMPTZ NOT NULL,
    renews_at                   TIMESTAMPTZ NOT NULL,
    trial_ends_at               TIMESTAMPTZ,
    ends_at                     TIMESTAMPTZ,
    user_id                     TEXT NOT NULL,
    lemonsqueezy_subscription_id TEXT,
    lemonsqueezy_customer_id    TEXT,
    lemonsqueezy_order_id       TEXT,
    lemonsqueezy_product_id    TEXT,
    lemonsqueezy_variant_id    TEXT
);
```

### Entitlement Table

**SQL Generated:** `internal/db/postgres/pgmodel/entitlements.sql.go`

```sql
CREATE TABLE entitlements (
    id             TEXT PRIMARY KEY,
    type           TEXT NOT NULL,
    subscription_id TEXT,
    app_id         TEXT NOT NULL,
    plan_id        TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL,
    ends_at        TIMESTAMPTZ
);
```

---

## Models

### Subscription Model

**File:** `internal/model/`

```go
type Subscription struct {
    ID                         string
    DisplayName                string
    Source                     SubscriptionSource // "sepay", "manual", "lemonsqueezy"
    Status                     string              // "active", "cancelled", "expired"
    StatusFormatted            string
    RenewsAt                   time.Time
    TrialEndsAt                null.Time
    EndsAt                     null.Time
    CreatedAt                  time.Time
    UpdatedAt                  time.Time
    UserID                     string
    LemonsqueezySubscriptionID null.String
    LemonsqueezyCustomerID     null.String
    LemonsqueezyOrderID        null.String
    LemonsqueezyProductID      null.String
    LemonsqueezyVariantID      null.String
}
```

### Entitlement Model

```go
type Entitlement struct {
    ID             string
    Type           string              // "subscription"
    SubscriptionID null.String
    AppID          string
    PlanID         string
    CreatedAt      time.Time
    UpdatedAt      time.Time
    EndsAt         null.Time            // Null = vô hạn
}
```

---

## Route Registration

**File:** `internal/api/routes.go:177-180`

```go
appBillingGroup := appGroup.Group("/billing")
appBillingGroup.Get("/subscriptions", handler.Typed(billingHandler.HandleAppSubscriptionList))
appBillingGroup.Post("/checkout", handler.TypedWithBody(billingHandler.HandleAppCheckout))
appBillingGroup.Get("/checkouts/{paymentID}", handler.Typed(billingHandler.HandleAppCheckoutStatus))
appBillingGroup.Get("/features", handler.Typed(billingHandler.HandleFeaturesGet))
```

---

## Key Files Summary

| File | Purpose |
|------|---------|
| `internal/api/handler/billing/checkout.go` | Khởi tạo checkout, tạo invoice number |
| `internal/api/handler/billing/checkout_status.go` | Kiểm tra trạng thái, tạo subscription |
| `internal/api/handler/billing/sepay.go` | SePay API client |
| `internal/api/handler/billing/sepay_ipn.go` | SePay IPN handler |
| `internal/api/handler/billing/webhook.go` | Manual webhook handler |
| `internal/api/handler/billing/payment/payment.go` | Utility functions (parse code, verify HMAC) |
| `internal/api/handler/billing/handler.go` | Billing handler initialization |
| `internal/store/subscription.go` | Subscription store interface |
| `internal/store/entitlement.go` | Entitlement store interface |
| `internal/db/postgres/store_subscription.go` | Subscription DB implementation |
| `internal/db/postgres/store_entitlement.go` | Entitlement DB implementation |
| `internal/api/wire/billing.go` | Wire DTOs |
