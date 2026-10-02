package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type Claims struct {
	Subject      string   `json:"sub"`
	Organization string   `json:"org,omitempty"`
	Roles        []string `json:"roles,omitempty"`
	ExpiresAt    int64    `json:"exp"`
	IssuedAt     int64    `json:"iat"`
}

// Sign issues an HS256 JWT (stdlib-only, no external deps).
func Sign(secret, subject, org string, roles []string, ttl time.Duration) (string, error) {
	now := time.Now()
	c := Claims{Subject: subject, Organization: org, Roles: roles, IssuedAt: now.Unix(), ExpiresAt: now.Add(ttl).Unix()}
	hb, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	pb, _ := json.Marshal(c)
	enc := func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }
	unsigned := enc(hb) + "." + enc(pb)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return unsigned + "." + sig, nil
}

func Verify(secret, token string) (*Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("malformed token")
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(want), []byte(parts[2])) {
		return nil, errors.New("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}
	var c Claims
	if err := json.Unmarshal(payload, &c); err != nil {
		return nil, err
	}
	if time.Now().Unix() > c.ExpiresAt {
		return nil, errors.New("token expired")
	}
	return &c, nil
}
