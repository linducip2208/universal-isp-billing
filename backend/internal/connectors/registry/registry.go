package registry

import (
	"fmt"
	"sort"
	"sync"

	"github.com/universal-isp/platform/internal/connectors/sdk"
)

type Factory func(cfg map[string]string) (sdk.NetworkConnector, error)

type Entry struct {
	Vendor           string
	ProductFamily    string
	Models           []string
	ConnectionType   sdk.ConnectionType
	Protocols        []string
	AuthMethods      []string
	Limitations      []string
	ConnectorVersion string
	Status           sdk.Verification
	DocURL           string
	VerifiedAt       string
	Evidence         string
	Factory          Factory
}

// Describe builds the public connector identity card (no secrets, no factory).
func (e Entry) Describe() sdk.Descriptor {
	return sdk.Descriptor{
		Vendor: e.Vendor, ProductFamily: e.ProductFamily, Models: e.Models,
		ConnectionType: e.ConnectionType, Protocols: e.Protocols, AuthMethods: e.AuthMethods,
		Limitations: e.Limitations, ConnectorVersion: e.ConnectorVersion,
		Status: e.Status, DocURL: e.DocURL, VerifiedAt: e.VerifiedAt, Evidence: e.Evidence,
	}
}

// Descriptors returns the public identity cards of all registered connectors.
func Descriptors() []sdk.Descriptor {
	out := []sdk.Descriptor{}
	for _, e := range List() {
		out = append(out, e.Describe())
	}
	return out
}

var (
	mu       sync.RWMutex
	registry = map[string]Entry{}
)

func key(vendor, family string, ct sdk.ConnectionType) string {
	return vendor + "|" + family + "|" + string(ct)
}

// Register adds a connector factory. Safe for init() use.
func Register(e Entry) {
	mu.Lock()
	defer mu.Unlock()
	registry[key(e.Vendor, e.ProductFamily, e.ConnectionType)] = e
}

func Lookup(vendor, family string, ct sdk.ConnectionType) (Entry, bool) {
	mu.RLock()
	defer mu.RUnlock()
	e, ok := registry[key(vendor, family, ct)]
	return e, ok
}

func List() []Entry {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Entry, 0, len(registry))
	for _, e := range registry {
		e.Factory = nil
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Vendor == out[j].Vendor {
			return out[i].ProductFamily < out[j].ProductFamily
		}
		return out[i].Vendor < out[j].Vendor
	})
	return out
}

func Create(vendor, family string, ct sdk.ConnectionType, cfg map[string]string) (sdk.NetworkConnector, error) {
	mu.RLock()
	e, ok := registry[key(vendor, family, ct)]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("no connector registered for %s/%s/%s", vendor, family, ct)
	}
	if e.Factory == nil {
		return nil, fmt.Errorf("connector %s/%s/%s has no factory", vendor, family, ct)
	}
	return e.Factory(cfg)
}
