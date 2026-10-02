package registry

import (
	"fmt"
	"sort"
	"sync"

	"github.com/universal-isp/platform/internal/connectors/sdk"
)

type Factory func(cfg map[string]string) (sdk.NetworkConnector, error)

type Entry struct {
	Vendor         string
	ProductFamily  string
	ConnectionType sdk.ConnectionType
	Status         sdk.CapabilityState
	DocURL         string
	Factory        Factory
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
