package mfa_test

import (
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/mfa"
)

// RFC 6238 Appendix B vectors: secret "12345678901234567890" (SHA-1),
// T = 59 -> 287082, T = 1111111109 -> 081804, T = 2000000000 -> 279037.
func TestRFC6238Vectors(t *testing.T) {
	secret := "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ" // base32 of "12345678901234567890"
	cases := []struct {
		unix string
		want string
	}{
		{"59", "287082"},
		{"1111111109", "081804"},
		{"2000000000", "279037"},
	}
	for _, c := range cases {
		var ts int64
		for _, ch := range c.unix {
			ts = ts*10 + int64(ch-'0')
		}
		got, err := mfa.CodeAt(secret, time.Unix(ts, 0).UTC())
		if err != nil || got != c.want {
			t.Fatalf("T=%s: got %q want %q err=%v", c.unix, got, c.want, err)
		}
		if !mfa.Verify(secret, c.want, time.Unix(ts, 0).UTC(), 1) {
			t.Fatalf("T=%s: verify failed", c.unix)
		}
	}
	if mfa.Verify(secret, "000000", time.Unix(59, 0).UTC(), 1) {
		t.Fatal("wrong code must fail")
	}
}

func TestSecretAndBackup(t *testing.T) {
	s, err := mfa.GenerateSecret()
	if err != nil || len(s) < 16 {
		t.Fatalf("secret: %q %v", s, err)
	}
	now := time.Now()
	code, err := mfa.CodeAt(s, now)
	if err != nil || !mfa.Verify(s, code, now, 1) {
		t.Fatalf("self code: %q %v", code, err)
	}
	if mfa.Verify(s, code, now.Add(5*time.Minute), 0) {
		t.Fatal("far-future code must fail with skew 0")
	}
	codes, hashes, err := mfa.NewBackupCodes(5)
	if err != nil || len(codes) != 5 {
		t.Fatal(err)
	}
	if !mfa.VerifyBackupCode(codes[2], hashes) {
		t.Fatal("backup code must verify")
	}
	if mfa.VerifyBackupCode("ZZZZZZZZ", hashes) {
		t.Fatal("wrong backup must fail")
	}
	u := mfa.OtpauthURL("ISP", "admin", s)
	if len(u) < 20 || u[:15] != "otpauth://totp/" {
		t.Fatalf("url=%q", u)
	}
}
