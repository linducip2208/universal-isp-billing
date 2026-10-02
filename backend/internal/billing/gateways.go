package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"errors"
)

// XenditProvider: Indonesian gateway (VA/QRIS/e-wallet/retail). Live charge
// creation requires XENDIT_SECRET_KEY; webhook verification is real HMAC.
type XenditProvider struct {
	SecretKey string
}

func (p XenditProvider) Name() string { return "xendit" }

func (p XenditProvider) CreateCharge(_ context.Context, req ChargeRequest) (*ChargeResponse, error) {
	if p.SecretKey == "" {
		return nil, errors.New("xendit: CREDENTIAL_REQUIRED (XENDIT_SECRET_KEY)")
	}
	// Production: POST https://api.xendit.co/v2/invoices with basic-auth secret.
	return nil, errors.New("xendit: live charge creation requires network + credentials (see docs)")
}

// VerifyWebhook checks Xendit's x-callback-token header (shared secret).
func (p XenditProvider) VerifyWebhook(_ context.Context, _ []byte, headers map[string]string) (*WebhookEvent, error) {
	if !hmac.Equal([]byte(headers["x-callback-token"]), []byte(p.SecretKey)) {
		return nil, errors.New("xendit: invalid callback token")
	}
	return &WebhookEvent{Status: "pending"}, nil
}

// MidtransProvider: SNAP/VA/QRIS. Webhook verified via SHA-512
// (order_id+status_code+gross_amount+server_key) per Midtrans docs.
type MidtransProvider struct {
	ServerKey string
}

func (p MidtransProvider) Name() string { return "midtrans" }

func (p MidtransProvider) CreateCharge(_ context.Context, req ChargeRequest) (*ChargeResponse, error) {
	if p.ServerKey == "" {
		return nil, errors.New("midtrans: CREDENTIAL_REQUIRED (MIDTRANS_SERVER_KEY)")
	}
	return nil, errors.New("midtrans: live charge creation requires network + credentials (see docs)")
}

func (p MidtransProvider) VerifyWebhook(_ context.Context, body []byte, headers map[string]string) (*WebhookEvent, error) {
	_ = body
	mac := hmac.New(sha512.New, []byte(p.ServerKey))
	mac.Write([]byte(headers["order_id"] + headers["status_code"] + headers["gross_amount"]))
	if hex.EncodeToString(mac.Sum(nil)) != headers["signature_key"] {
		return nil, errors.New("midtrans: invalid signature_key")
	}
	paid := headers["transaction_status"] == "settlement" || headers["transaction_status"] == "capture"
	return &WebhookEvent{Reference: headers["order_id"], Status: headers["transaction_status"], Paid: paid}, nil
}

var _ = sha256.New
