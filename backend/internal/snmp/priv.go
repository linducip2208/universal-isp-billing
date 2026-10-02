package snmp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
)

// Privacy (RFC 3826, AES-128 CFB): privKey = first 16 octets of the
// localized auth key. salt(8) = engineBoots(4) || random(4); IV(16) =
// engineBoots(4) || engineTime(4) || salt(8). Interop E2E pending
// (SNMP_E2E); round-trip + tamper covered by tests.

func privKey16(localized []byte) ([]byte, error) {
	if len(localized) < 16 {
		return nil, errors.New("localized key too short for privacy")
	}
	return localized[:16], nil
}

// EncryptScoped encrypts BER scopedPDU bytes, returning ciphertext + salt.
func EncryptScoped(localized []byte, boots uint32, engineTime uint32, scoped []byte) (ct, salt []byte, err error) {
	key, err := privKey16(localized)
	if err != nil {
		return nil, nil, err
	}
	salt = make([]byte, 8)
	binary.BigEndian.PutUint32(salt[:4], boots)
	if _, err := rand.Read(salt[4:]); err != nil {
		return nil, nil, err
	}
	iv := make([]byte, 16)
	binary.BigEndian.PutUint32(iv[:4], boots)
	binary.BigEndian.PutUint32(iv[4:8], engineTime)
	copy(iv[8:], salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}
	ct = make([]byte, len(scoped))
	cipher.NewCFBEncrypter(block, iv).XORKeyStream(ct, scoped)
	return ct, salt, nil
}

// DecryptScoped reverses EncryptScoped; wrong key/salt yields garbage or error.
func DecryptScoped(localized []byte, boots uint32, engineTime uint32, salt, ct []byte) ([]byte, error) {
	key, err := privKey16(localized)
	if err != nil {
		return nil, err
	}
	if len(salt) != 8 {
		return nil, errors.New("salt must be 8 octets")
	}
	iv := make([]byte, 16)
	binary.BigEndian.PutUint32(iv[:4], boots)
	binary.BigEndian.PutUint32(iv[4:8], engineTime)
	copy(iv[8:], salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	pt := make([]byte, len(ct))
	cipher.NewCFBDecrypter(block, iv).XORKeyStream(pt, ct)
	return pt, nil
}
