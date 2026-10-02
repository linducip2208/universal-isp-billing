package contract_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/connectors/all"
	"github.com/universal-isp/platform/internal/connectors/contract"
	"github.com/universal-isp/platform/internal/connectors/registry"
)

func TestAllRegisteredConnectors(t *testing.T) {
	all.RegisterAll()
	entries := registry.Raw()
	if len(entries) == 0 {
		t.Fatal("registry empty")
	}
	failed := 0
	for _, e := range entries {
		res := contract.Check(e.Vendor+"/"+e.ProductFamily, e)
		if !res.Passed {
			failed++
			t.Errorf("%s/%s: %v", res.Vendor, res.Family, res.Failures)
		}
	}
	t.Logf("contract: %d connectors checked, %d failed", len(entries), failed)
}
