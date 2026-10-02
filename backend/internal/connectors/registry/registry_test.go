package registry_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/connectors/registry"
	"github.com/universal-isp/platform/internal/connectors/sdk"
)

func TestRegistryEmptyByDefault(t *testing.T) {
	if _, ok := registry.Lookup("Nope", "Nope", sdk.ConnREST); ok {
		t.Fatal("should miss")
	}
}

func TestRegisterAndCreate(t *testing.T) {
	registry.Register(registry.Entry{Vendor: "TestV", ProductFamily: "F1", ConnectionType: sdk.ConnREST, Status: sdk.StateImplemented,
		Factory: func(cfg map[string]string) (sdk.NetworkConnector, error) { return nil, nil }})
	if _, ok := registry.Lookup("TestV", "F1", sdk.ConnREST); !ok {
		t.Fatal("should find")
	}
}
