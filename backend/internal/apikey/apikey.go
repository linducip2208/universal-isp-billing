// Package apikey: API key issuance and verification. Only SHA-256 hashes
// are stored; the raw key is shown once at creation.
package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
)

func Generate() (raw, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = "isp_" + base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(raw))
	return raw, hex.EncodeToString(sum[:]), nil
}

func Verify(raw, hash string) error {
	sum := sha256.Sum256([]byte(raw))
	want, err := hex.DecodeString(hash)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare(sum[:], want) != 1 {
		return errors.New("invalid api key")
	}
	return nil
}
