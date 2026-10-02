package security_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/security"
)

func TestPasswordRoundTrip(t *testing.T) {
	h, err := security.HashPassword("correct-horse-9")
	if err != nil {
		t.Fatal(err)
	}
	if err := security.VerifyPassword("correct-horse-9", h); err != nil {
		t.Fatal(err)
	}
	if err := security.VerifyPassword("wrong", h); err == nil {
		t.Fatal("wrong password must fail")
	}
	if _, err := security.HashPassword("short"); err == nil {
		t.Fatal("short password must fail")
	}
	if err := security.VerifyPassword("x", "bcrypt$..."); err == nil {
		t.Fatal("unknown format must fail")
	}
}
