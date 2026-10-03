// Package vouchers: hotspot/voucher codes — generation, expiry, single
// redemption. Redemption is enforced in the DB (status transition guarded
// by WHERE status='active') so concurrent redemptions cannot double-spend.
package vouchers

import (
	"crypto/rand"
	"errors"
	"strings"
	"time"
)

type Voucher struct {
	ID            string     `json:"id"`
	Code          string     `json:"code"`
	PackageID     string     `json:"package_id,omitempty"`
	DurationHours int        `json:"duration_hours"`
	Status        string     `json:"status"` // active | used | expired | revoked
	UsedBy        string     `json:"used_by,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
}

// GenerateCode creates an unambiguous human-typable code (no 0/O/1/I).
func GenerateCode(n int) (string, error) {
	const alphabet = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"
	if n <= 0 {
		return "", errors.New("length required")
	}
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, c := range b {
		sb.WriteByte(alphabet[int(c)%len(alphabet)])
	}
	return sb.String(), nil
}

// Redeemable reports whether the voucher can be consumed now (pure check;
// the DB enforces it transitionally).
func Redeemable(v Voucher, now time.Time) error {
	if v.Status != "active" {
		return errors.New("voucher not active")
	}
	if v.ExpiresAt != nil && now.After(*v.ExpiresAt) {
		return errors.New("voucher expired")
	}
	return nil
}
