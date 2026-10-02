package auth_test

import (
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/auth"
)

func TestJWT(t *testing.T) {
	tok, err := auth.Sign("test-secret-12345678", "admin", "org1", []string{"superadmin"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	cl, err := auth.Verify("test-secret-12345678", tok)
	if err != nil {
		t.Fatal(err)
	}
	if cl.Subject != "admin" {
		t.Fatal("subject mismatch")
	}
	if _, err := auth.Verify("wrong-secret-xxxxxxxx", tok); err == nil {
		t.Fatal("wrong secret should fail")
	}
}
