package webhooks_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/webhooks"
)

func TestSignVerify(t *testing.T) {
	sig := webhooks.Sign("s3cret", []byte(`{"a":1}`))
	if !webhooks.Verify("s3cret", []byte(`{"a":1}`), sig) {
		t.Fatal("should verify")
	}
	if webhooks.Verify("wrong", []byte(`{"a":1}`), sig) {
		t.Fatal("wrong secret must fail")
	}
}
