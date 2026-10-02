package snmp

import (
	"bytes"
	"testing"
)

func TestPrivRoundTrip(t *testing.T) {
	eng := []byte{1, 2, 3, 4}
	key := LocalizeKey("privpass", eng, AuthSHA)
	pt := []byte{0x30, 0x0b, 0x04, 0x01, 0x00, 0x04, 0x01, 0x00, 0xA0, 0x03, 0x02, 0x01, 0x01}
	ct, salt, err := EncryptScoped(key, 7, 999, pt)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(ct, pt) {
		t.Fatal("must actually encrypt")
	}
	back, err := DecryptScoped(key, 7, 999, salt, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(back, pt) {
		t.Fatal("round-trip mismatch")
	}
	// salt uniqueness -> different ciphertexts
	ct2, _, err := EncryptScoped(key, 7, 999, pt)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(ct, ct2) {
		t.Fatal("salt must randomize ciphertext")
	}
	// wrong key -> garbage, not plaintext
	wrong := LocalizeKey("other", eng, AuthSHA)
	bad, err := DecryptScoped(wrong, 7, 999, salt, ct)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(bad, pt) {
		t.Fatal("wrong key must not recover plaintext")
	}
}
