package security

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

// Password hashing: PBKDF2-SHA256, 210k iterations, 128-bit salt (stdlib +
// x/crypto KDF only — no cgo, no external C deps). Stored format:
// "pbkdf2-sha256$<iter>$<salt-b64>$<hash-b64>".
const (
	pbkdf2Iter = 210000
	saltLen    = 16
	keyLen     = 32
)

func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password too short (min 8)")
	}
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	dk := pbkdf2.Key([]byte(password), salt, pbkdf2Iter, keyLen, sha256.New)
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s",
		pbkdf2Iter,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(dk)), nil
}

func VerifyPassword(password, encoded string) error {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return errors.New("unknown password hash format")
	}
	var iter int
	if _, err := fmt.Sscanf(parts[1], "%d", &iter); err != nil || iter < 100000 {
		return errors.New("weak password hash parameters")
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return errors.New("bad salt encoding")
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return errors.New("bad hash encoding")
	}
	got := pbkdf2.Key([]byte(password), salt, iter, len(want), sha256.New)
	if subtle.ConstantTimeCompare(got, want) != 1 {
		return errors.New("invalid password")
	}
	return nil
}
