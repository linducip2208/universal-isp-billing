// Package webhooks dispatches signed event deliveries with retry.
// Each delivery is HMAC-SHA256 signed (X-ISP-Signature) so receivers can
// verify authenticity. Delivery itself rides the jobs.Queue for retry/DLQ.
package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"time"
)

type Endpoint struct {
	URL    string
	Secret string
	Events []string
}

func Sign(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func Verify(secret string, body []byte, sig string) bool {
	want := Sign(secret, body)
	return hmac.Equal([]byte(want), []byte(sig))
}

// Deliver POSTs body once with signature headers and a 10s timeout.
func Deliver(ctx context.Context, ep Endpoint, eventType string, body []byte) (int, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ep.URL, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-ISP-Event", eventType)
	req.Header.Set("X-ISP-Signature", "sha256="+Sign(ep.Secret, body))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
