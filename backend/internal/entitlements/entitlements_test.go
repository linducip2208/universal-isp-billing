package entitlements_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/entitlements"
)

func TestCheck(t *testing.T) {
	v := entitlements.Check(entitlements.Community, entitlements.Usage{Devices: 11, Subscribers: 10})
	if len(v) != 1 || v[0].Resource != "devices" {
		t.Fatalf("v=%+v", v)
	}
	if len(entitlements.Check(entitlements.Enterprise, entitlements.Usage{Devices: 1 << 20})) != 0 {
		t.Fatal("enterprise unlimited")
	}
	if len(entitlements.Check("nope", entitlements.Usage{})) != 0 {
		t.Fatal("unknown plan falls back to community, zero usage passes")
	}
}
