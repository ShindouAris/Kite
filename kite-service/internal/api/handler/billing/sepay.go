package billing

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type sepayClient struct {
	baseURL     string
	bearerToken string
	httpClient  *http.Client
}

func newSePayClient(baseURL, bearerToken string) *sepayClient {
	return &sepayClient{
		baseURL:     strings.TrimRight(baseURL, "/"),
		bearerToken: strings.TrimSpace(bearerToken),
		httpClient:  &http.Client{Timeout: 20 * time.Second},
	}
}

type sepayCreateOrderRequest struct {
	VAPrefix       string `json:"va_prefix,omitempty"`
	OrderCode      string `json:"order_code,omitempty"`
	Amount         int    `json:"amount,omitempty"`
	VAHolderName   string `json:"va_holder_name,omitempty"`
	Duration       int    `json:"duration,omitempty"`
	WithQRCode     int    `json:"with_qrcode,omitempty"`
	QRCodeTemplate string `json:"qrcode_template,omitempty"`
}

type sepayOrder struct {
	ID                string `json:"id"`
	OrderCode         string `json:"order_code"`
	VANumber          string `json:"va_number"`
	VAHolderName      string `json:"va_holder_name"`
	Amount            int    `json:"amount"`
	Status            string `json:"status"`
	BankName          string `json:"bank_name"`
	AccountHolderName string `json:"account_holder_name"`
	AccountNumber     string `json:"account_number"`
	ExpiredAt         string `json:"expired_at"`
	QRCode            string `json:"qr_code"`
	QRCodeURL         string `json:"qr_code_url"`
}

type sepayResponse struct {
	Status  string     `json:"status"`
	Message string     `json:"message"`
	Data    sepayOrder `json:"data"`
}

func (c *sepayClient) CreateOrder(ctx context.Context, bankAccountXID string, req sepayCreateOrderRequest) (*sepayOrder, error) {
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/bank-accounts/%s/orders", bankAccountXID), req)
}

func (c *sepayClient) GetOrder(ctx context.Context, bankAccountXID, orderID string) (*sepayOrder, error) {
	return c.do(ctx, http.MethodGet, fmt.Sprintf("/bank-accounts/%s/orders/%s", bankAccountXID, orderID), nil)
}

func (c *sepayClient) CancelOrder(ctx context.Context, bankAccountXID, orderID string) error {
	_, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/bank-accounts/%s/orders/%s", bankAccountXID, orderID), nil)
	return err
}

func (c *sepayClient) do(ctx context.Context, method string, path string, body interface{}) (*sepayOrder, error) {
	if c == nil {
		return nil, fmt.Errorf("sepay client is not configured")
	}

	var payload io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to encode sepay request: %w", err)
		}
		payload = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create sepay request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.bearerToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call sepay api: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read sepay response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("sepay api returned %s: %s", resp.Status, strings.TrimSpace(string(bodyBytes)))
	}

	var payloadResp sepayResponse
	if err := json.Unmarshal(bodyBytes, &payloadResp); err != nil {
		return nil, fmt.Errorf("failed to decode sepay response: %w", err)
	}

	if !strings.EqualFold(payloadResp.Status, "success") && payloadResp.Status != "" {
		if payloadResp.Message == "" {
			payloadResp.Message = "sepay request failed"
		}
		return nil, fmt.Errorf("%s", payloadResp.Message)
	}

	return &payloadResp.Data, nil
}
