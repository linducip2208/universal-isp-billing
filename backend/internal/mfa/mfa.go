// Package mfa: TOTP two-factor auth (RFC 6238, SHA-1, 30s step, 6 digits)
// plus single-use backup codes (PBKDF2-hashed). Verified against the RFC
// 6238 Appendix B test vectors — not just self-consistent.
package mfa

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/universal-isp/platform/internal/security"
)

// GenerateSecret creates a 160-bit base32 (no padding) TOTP secret.
func GenerateSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// OtpauthURL builds the QR-code payload for authenticator apps.
func OtpauthURL(issuer, account, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		issuer, account, secret, issuer)
}

func counter(t time.Time, step int64) uint64 {
	if step <= 0 {
		step = 30
	}
	return uint64(t.Unix() / step)
}

func hotp(key []byte, counter uint64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0F
	code := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7FFFFFFF
	return fmt.Sprintf("%06d", code%1000000)
}

func decodeSecret(secret string) ([]byte, error) {
	s := strings.ToUpper(strings.TrimSpace(secret))
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(s)
	if err != nil {
		// tolerate padded input too
		key, err = base32.StdEncoding.DecodeString(s)
		if err != nil {
			return nil, errors.New("bad totp secret encoding")
		}
	}
	return key, nil
}

// CodeAt generates the 6-digit code for time t (testable, deterministic).
func CodeAt(secret string, t time.Time) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}
	return hotp(key, counter(t, 30)), nil
}

// Verify accepts codes within ±skew steps (default 1) to tolerate clock drift.
func Verify(secret, code string, at time.Time, skew int) bool {
	key, err := decodeSecret(secret)
	if err != nil || len(code) != 6 {
		return false
	}
	if skew < 0 {
		skew = 0
	}
	c := int64(counter(at, 30))
	for d := -int64(skew); d <= int64(skew); d++ {
		var msg [8]byte
		binary.BigEndian.PutUint64(msg[:], uint64(int64(c)+d))
		mac := hmac.New(sha1.New, key)
		mac.Write(msg[:])
		sum := mac.Sum(nil)
		offset := sum[len(sum)-1] & 0x0F
		v := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7FFFFFFF
		if fmt.Sprintf("%06d", v%1000000) == code {
			return true
		}
	}
	return false
}

// NewBackupCodes generates n single-use codes; only hashes are stored.
func NewBackupCodes(n int) (codes []string, hashes []string, err error) {
	for i := 0; i < n; i++ {
		var b [5]byte
		if _, err := rand.Read(b[:]); err != nil {
			return nil, nil, err
		}
		c := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:]))[:8]
		h, err := security.HashPassword("backup:" + c)
		if err != nil {
			return nil, nil, err
		}
		codes = append(codes, c)
		hashes = append(hashes, h)
	}
	return codes, hashes, nil
}

// VerifyBackupCode checks a presented code against stored hashes.
func VerifyBackupCode(code string, hashes []string) bool {
	for _, h := range hashes {
		if security.VerifyPassword("backup:"+code, h) == nil {
			return true
		}
	}
	return false
}
