package snmp

import (
	"bytes"
	"testing"
)

func TestLocalizeDeterministic(t *testing.T) {
	eng := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
	k1 := LocalizeKey("maplesyrup", eng, AuthMD5)
	k2 := LocalizeKey("maplesyrup", eng, AuthMD5)
	if !bytes.Equal(k1, k2) || len(k1) != 16 {
		t.Fatal("localization must be deterministic 16-byte")
	}
	k3 := LocalizeKey("maplesyrup", []byte{9, 9, 9}, AuthMD5)
	if bytes.Equal(k1, k3) {
		t.Fatal("engineID must change the key")
	}
	ks := LocalizeKey("maplesyrup", eng, AuthSHA)
	if len(ks) != 20 {
		t.Fatal("SHA localization must be 20-byte")
	}
}

func TestV3RoundTripAndTamper(t *testing.T) {
	eng := []byte{1, 2, 3, 4, 5}
	key := LocalizeKey("secret123", eng, AuthSHA)
	user := USMUser{Username: "bob", Auth: AuthSHA, AuthKey: key, EngineID: eng}
	msg := EncodeV3Auth(user, 7, ".1.3.6.1.2.1.1.1.0")
	if err := VerifyV3Auth(msg, key, AuthSHA); err != nil {
		t.Fatalf("verify: %v", err)
	}
	bad := append([]byte{}, msg...)
	bad[len(bad)-1] ^= 0xFF
	if err := VerifyV3Auth(bad, key, AuthSHA); err == nil {
		t.Fatal("tampered message must fail")
	}
	wrong := LocalizeKey("other", eng, AuthSHA)
	if err := VerifyV3Auth(msg, wrong, AuthSHA); err == nil {
		t.Fatal("wrong key must fail")
	}
	// Scoped PDU must still decode after auth wrap.
	res, err := decResponseScoped(msg)
	if err != nil {
		t.Fatalf("scoped decode: %v", err)
	}
	if len(res.vbs) != 1 || res.vbs[0].oid != ".1.3.6.1.2.1.1.1.0" {
		t.Fatalf("vbs=%+v", res.vbs)
	}
}
