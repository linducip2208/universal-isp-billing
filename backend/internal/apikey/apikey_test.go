package apikey_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/apikey"
)

func TestRoundTrip(t *testing.T) {
	raw, hash, err := apikey.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if err := apikey.Verify(raw, hash); err != nil {
		t.Fatal(err)
	}
	if err := apikey.Verify(raw+"x", hash); err == nil {
		t.Fatal("tampered key must fail")
	}
}
