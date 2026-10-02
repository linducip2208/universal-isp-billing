package security_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/security"
)

func TestSecretsBox(t *testing.T) {
	k, _ := security.DeriveKey("test-passphrase")
	box, err := security.NewSecretsBox(k)
	if err != nil {
		t.Fatal(err)
	}
	ct, err := box.Encrypt("s3cr3t")
	if err != nil {
		t.Fatal(err)
	}
	pt, err := box.Decrypt(ct)
	if err != nil {
		t.Fatal(err)
	}
	if pt != "s3cr3t" {
		t.Fatal("roundtrip failed")
	}
	if ct == "s3cr3t" {
		t.Fatal("not encrypted")
	}
}
