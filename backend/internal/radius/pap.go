package radius

import (
	"bytes"
	"crypto/md5"
	"errors"
)

// PAP password hiding, RFC 2865 section 5.2. The User-Password attribute on
// the wire is MD5-chained ciphertext, NOT plaintext — a real NAS always
// sends ciphertext, so the server must decrypt before verifying.

// EncryptPAP hides password for an Access-Request with requestAuth.
func EncryptPAP(secret string, requestAuth [16]byte, password string) ([]byte, error) {
	if len(password) == 0 {
		return nil, errors.New("empty password")
	}
	if len(password) > 128 {
		return nil, errors.New("password exceeds 128 octets")
	}
	// Zero-pad to a multiple of 16.
	padded := []byte(password)
	if r := len(padded) % 16; r != 0 {
		padded = append(padded, make([]byte, 16-r)...)
	}
	out := make([]byte, 0, len(padded))
	prev := requestAuth[:]
	for i := 0; i < len(padded); i += 16 {
		h := md5.New()
		h.Write([]byte(secret))
		h.Write(prev)
		b := h.Sum(nil)
		block := make([]byte, 16)
		for j := 0; j < 16; j++ {
			block[j] = padded[i+j] ^ b[j]
		}
		out = append(out, block...)
		prev = out[len(out)-16:]
	}
	return out, nil
}

// DecryptPAP reverses EncryptPAP. Trailing zero padding is stripped.
func DecryptPAP(secret string, requestAuth [16]byte, ciphertext []byte) (string, error) {
	if len(ciphertext) == 0 || len(ciphertext)%16 != 0 {
		return "", errors.New("malformed User-Password")
	}
	if len(ciphertext) > 128 {
		return "", errors.New("User-Password exceeds 128 octets")
	}
	out := make([]byte, 0, len(ciphertext))
	prev := requestAuth[:]
	for i := 0; i < len(ciphertext); i += 16 {
		h := md5.New()
		h.Write([]byte(secret))
		h.Write(prev)
		b := h.Sum(nil)
		for j := 0; j < 16; j++ {
			out = append(out, ciphertext[i+j]^b[j])
		}
		prev = ciphertext[i : i+16]
	}
	return string(bytes.TrimRight(out, "\x00")), nil
}
