package auth_test

import (
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/auth"
)

func TestRevoke(t *testing.T) {
	r := auth.NewMemoryRevoker()
	if r.Revoked("abc") {
		t.Fatal("fresh jti must pass")
	}
	r.Revoke("abc", time.Hour)
	if !r.Revoked("abc") {
		t.Fatal("revoked jti must fail")
	}
	r.Revoke("gone", time.Millisecond)
	time.Sleep(5 * time.Millisecond)
	if r.Revoked("gone") {
		t.Fatal("expired revocation must pass")
	}
	r.Revoke("", time.Hour) // no-op, must not panic
}

func TestSignHasJTI(t *testing.T) {
	tok, err := auth.Sign("test-secret-12345678", "admin", "org1", []string{"superadmin"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	cl, err := auth.Verify("test-secret-12345678", tok)
	if err != nil {
		t.Fatal(err)
	}
	if cl.ID == "" {
		t.Fatal("token must carry jti")
	}
}
