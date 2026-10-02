package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
)

// SecretsBox encrypts network credentials at rest using AES-256-GCM.
// Key must be 32 bytes; if empty key material is provided, DeriveKey
// stretches the passphrase with SHA-256 (operators should supply a
// proper 32-byte key via SECRETS_KEY in production).
type SecretsBox struct{ gcm cipher.AEAD }

func DeriveKey(passphrase string) ([]byte, error) {
	if passphrase == "" {
		return nil, errors.New("empty secrets key")
	}
	sum := sha256.Sum256([]byte(passphrase))
	return sum[:], nil
}

func NewSecretsBox(key []byte) (*SecretsBox, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("secrets key must be 32 bytes, got %d", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SecretsBox{gcm: gcm}, nil
}

// NoopBox stores reversible-obfuscated values for dev/test only.
type NoopBox struct{}

func (NoopBox) Encrypt(plain string) (string, error) { return "noop:" + plain, nil }
func (NoopBox) Decrypt(c string) (string, error) {
	const p = "noop:"
	if len(c) < len(p) {
		return "", errors.New("bad ciphertext")
	}
	return c[len(p):], nil
}

func (s *SecretsBox) Encrypt(plain string) (string, error) {
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ct := s.gcm.Seal(nonce, nonce, []byte(plain), nil)
	return base64.StdEncoding.EncodeToString(ct), nil
}

func (s *SecretsBox) Decrypt(encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	ns := s.gcm.NonceSize()
	if len(raw) < ns {
		return "", errors.New("ciphertext too short")
	}
	pt, err := s.gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return "", err
	}
	return string(pt), nil
}
