package billing

import "context"

// PaymentProvider abstracts gateways (Xendit/Midtrans/DOKU/bank/QRIS/manual).
// Implementations must verify webhooks with provider signatures and be idempotent.
type PaymentProvider interface {
	Name() string
	// CreateCharge returns provider reference + checkout URL/QR payload.
	CreateCharge(ctx context.Context, req ChargeRequest) (*ChargeResponse, error)
	// VerifyWebhook authenticates and parses a provider callback.
	VerifyWebhook(ctx context.Context, rawBody []byte, headers map[string]string) (*WebhookEvent, error)
}

type ChargeRequest struct {
	InvoiceID      string `json:"invoice_id"`
	Amount         Money  `json:"amount"`
	Currency       string `json:"currency"`
	Method         string `json:"method"` // va | qris | ewallet | card | bank_transfer | manual
	CustomerRef    string `json:"customer_ref"`
	IdempotencyKey string `json:"idempotency_key"`
}

type ChargeResponse struct {
	Reference   string `json:"reference"`
	CheckoutURL string `json:"checkout_url,omitempty"`
	QRPayload   string `json:"qr_payload,omitempty"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

type WebhookEvent struct {
	Reference string `json:"reference"`
	InvoiceID string `json:"invoice_id"`
	Amount    Money  `json:"amount"`
	Status    string `json:"status"` // paid | failed | pending
	Paid      bool   `json:"paid"`
}

// ManualProvider records offline/manual payments (bank transfer receipt, cash).
type ManualProvider struct{}

func (ManualProvider) Name() string { return "manual" }
func (ManualProvider) CreateCharge(_ context.Context, req ChargeRequest) (*ChargeResponse, error) {
	return &ChargeResponse{Reference: "MANUAL-" + req.InvoiceID}, nil
}
func (ManualProvider) VerifyWebhook(_ context.Context, _ []byte, _ map[string]string) (*WebhookEvent, error) {
	return &WebhookEvent{Status: "pending"}, nil
}
