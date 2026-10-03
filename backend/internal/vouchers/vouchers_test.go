package vouchers_test

import (
	"strings"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/vouchers"
)

func TestGenerateUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		c, err := vouchers.GenerateCode(8)
		if err != nil {
			t.Fatal(err)
		}
		if len(c) != 8 || seen[c] {
			t.Fatalf("bad/dup code %q", c)
		}
		seen[c] = true
		for _, ch := range c {
			if strings.ContainsRune("0O1I", ch) {
				t.Fatalf("ambiguous char in %q", c)
			}
		}
	}
	if _, err := vouchers.GenerateCode(0); err == nil {
		t.Fatal("zero length must fail")
	}
}

func TestRedeemable(t *testing.T) {
	now := time.Now()
	if err := vouchers.Redeemable(vouchers.Voucher{Status: "active"}, now); err != nil {
		t.Fatal(err)
	}
	if err := vouchers.Redeemable(vouchers.Voucher{Status: "used"}, now); err == nil {
		t.Fatal("used must fail")
	}
	past := now.Add(-time.Hour)
	if err := vouchers.Redeemable(vouchers.Voucher{Status: "active", ExpiresAt: &past}, now); err == nil {
		t.Fatal("expired must fail")
	}
}
