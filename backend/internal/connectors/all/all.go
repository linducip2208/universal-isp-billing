// Package all registers every connector family exactly once.
// Both cmd/server and cmd/ispctl call RegisterAll so the CLI and the API
// always agree on vendors, statuses, and capabilities.
package all

import (
	"time"

	"github.com/universal-isp/platform/internal/connectors/cloud"
	"github.com/universal-isp/platform/internal/connectors/generic"
	"github.com/universal-isp/platform/internal/connectors/mikrotik"
	"github.com/universal-isp/platform/internal/connectors/olt"
	"github.com/universal-isp/platform/internal/connectors/registry"
	"github.com/universal-isp/platform/internal/connectors/routers"
	"github.com/universal-isp/platform/internal/connectors/sdk"
)

const ConnectorVersion = "1.0.0"

func RegisterAll() {
	mk := func(cfg map[string]string) (sdk.NetworkConnector, error) {
		return mikrotik.New(mikrotik.Config{Host: cfg["host"], Username: cfg["username"], Password: cfg["password"], Timeout: 10 * time.Second}), nil
	}
	mkREST := func(cfg map[string]string) (sdk.NetworkConnector, error) {
		return mikrotik.New(mikrotik.Config{Host: cfg["host"], Username: cfg["username"], Password: cfg["password"], UseREST: true, UseTLS: true, Timeout: 10 * time.Second}), nil
	}
	// Tier 1: MikroTik — PARTIAL (protocol-tested; hardware E2E pending).
	registry.Register(registry.Entry{Vendor: "MikroTik", ProductFamily: "RouterOS",
		Models: []string{"CCR", "CRS", "RB", "CHR"}, ConnectionType: sdk.ConnCLI,
		Protocols: []string{"routeros-api", "routeros-api-ssl"}, AuthMethods: []string{"password"},
		Limitations:      []string{"hardware E2E pending (MIKROTIK_E2E)"},
		ConnectorVersion: ConnectorVersion, Status: sdk.Partial,
		DocURL: "docs/connectors/mikrotik.md", Evidence: "mock-transport protocol tests", Factory: mk})
	registry.Register(registry.Entry{Vendor: "MikroTik", ProductFamily: "RouterOS-REST",
		ConnectionType: sdk.ConnREST, Protocols: []string{"https"}, AuthMethods: []string{"basic"},
		Limitations:      []string{"hardware E2E pending (MIKROTIK_E2E)"},
		ConnectorVersion: ConnectorVersion, Status: sdk.Partial,
		DocURL: "docs/connectors/mikrotik.md", Evidence: "mock-transport protocol tests", Factory: mkREST})
	// Tier 2: cloud — REQUIRES_VENDOR_ACCESS (adapters fail closed without credentials).
	clouds := []struct {
		v, f, doc string
		fn        func(map[string]string) (sdk.NetworkConnector, error)
	}{
		{"Ruijie", "Cloud", "docs/connectors/ruijie-cloud.md", cloud.NewRuijieCloud},
		{"Ruijie", "Reyee", "docs/connectors/reyee.md", cloud.NewReyee},
		{"Ubiquiti", "UniFi", "docs/connectors/unifi.md", cloud.NewUniFi},
		{"TP-Link", "Omada", "docs/connectors/omada.md", cloud.NewOmada},
		{"Cisco", "Meraki", "docs/connectors/meraki.md", cloud.NewMeraki},
		{"Aruba", "Central", "docs/connectors/aruba-central.md", cloud.NewArubaCentral},
		{"Juniper", "Mist", "docs/connectors/mist.md", cloud.NewMist},
		{"Cambium", "cnMaestro", "docs/connectors/cnmaestro.md", cloud.NewCnMaestro},
	}
	for _, c := range clouds {
		registry.Register(registry.Entry{Vendor: c.v, ProductFamily: c.f, ConnectionType: sdk.ConnCloudAPI,
			Protocols: []string{"https"}, AuthMethods: []string{"oauth2", "api-key"},
			Limitations:      []string{"vendor partnership/credentials required"},
			ConnectorVersion: ConnectorVersion, Status: sdk.RequiresVendorAccess,
			DocURL: c.doc, Factory: c.fn})
	}
	// Tier 3: routers — PLANNED (fail-closed command templates).
	routerFams := []struct {
		v, f, key string
		ct        sdk.ConnectionType
	}{
		{"Cisco", "IOS/XE", "cisco-ios", sdk.ConnSSH},
		{"Cisco", "SmallBusiness", "cisco-smb", sdk.ConnREST},
		{"Juniper", "Junos", "juniper-junos", sdk.ConnNETCONF},
		{"Huawei", "VRP", "huawei-vrp", sdk.ConnSSH},
		{"ZTE", "ZXAN", "zte-zxhn", sdk.ConnSSH},
		{"Nokia", "SR OS", "nokia-sros", sdk.ConnNETCONF},
		{"VyOS", "VyOS", "vyos", sdk.ConnRESTCONF},
		{"Ubiquiti", "EdgeRouter", "edge-router", sdk.ConnSSH},
		{"Fortinet", "FortiOS", "fortinet", sdk.ConnREST},
		{"Aruba", "CX", "aruba-cx", sdk.ConnREST},
		{"Ruijie", "Router", "ruijie-router", sdk.ConnSSH},
		{"D-Link", "Smart", "dlink-smart", sdk.ConnSNMP},
	}
	for _, r := range routerFams {
		fn, ok := routers.Factories[r.key]
		if !ok {
			panic("missing router factory: " + r.key)
		}
		registry.Register(registry.Entry{Vendor: r.v, ProductFamily: r.f, ConnectionType: r.ct,
			Protocols: []string{string(r.ct)}, AuthMethods: []string{"password", "key"},
			Limitations:      []string{"per-model command templates not yet implemented; fail-closed"},
			ConnectorVersion: ConnectorVersion, Status: sdk.Planned,
			DocURL: "docs/connectors/generic.md", Factory: fn})
	}
	// Tier 4: OLT/FTTH — PLANNED.
	oltFams := [][2]string{{"Huawei", "MA5600T"}, {"ZTE", "C320"}, {"Nokia", "7360"}, {"FiberHome", "AN5116"}, {"BDCOM", "P3310"}, {"VSOL", "V1600"}, {"C-Data", "FD1608"}, {"Dasan", "V8240"}, {"Raisecom", "ISC"},
		{"Zyxel", "OLT1408"}, {"Calix", "E7"}, {"ADTRAN", "TA5000"}, {"DZS", "Velocity"}}
	for _, o := range oltFams {
		v, f := o[0], o[1]
		registry.Register(registry.Entry{Vendor: v, ProductFamily: f, ConnectionType: sdk.ConnSNMP,
			Protocols: []string{"snmp", "ssh"}, AuthMethods: []string{"community", "password"},
			Limitations:      []string{"vendor adapter not yet implemented; fail-closed"},
			ConnectorVersion: ConnectorVersion, Status: sdk.Planned,
			DocURL: "docs/connectors/olt.md", Factory: olt.Mk(v, f, sdk.ConnSNMP)})
	}
	// Generic standards — PARTIAL (real protocol code, interop E2E pending).
	registry.Register(registry.Entry{Vendor: "Generic", ProductFamily: "RADIUS", ConnectionType: sdk.ConnRADIUS,
		Protocols: []string{"radius", "coa"}, AuthMethods: []string{"shared-secret"},
		Limitations:      []string{"NAS interop E2E pending (RADIUS_E2E)"},
		ConnectorVersion: ConnectorVersion, Status: sdk.Partial,
		DocURL: "docs/connectors/generic.md", Evidence: "RFC packet codec unit tests",
		Factory: func(cfg map[string]string) (sdk.NetworkConnector, error) {
			return generic.NewRADIUS(cfg["nas_addr"], cfg["secret"]), nil
		}})
	registry.Register(registry.Entry{Vendor: "Generic", ProductFamily: "SNMP", ConnectionType: sdk.ConnSNMP,
		Protocols: []string{"snmp"}, AuthMethods: []string{"community"},
		Limitations:      []string{"v2c poller implemented; v3 USM roadmap; agent interop E2E pending (SNMP_E2E)"},
		ConnectorVersion: ConnectorVersion, Status: sdk.Partial,
		DocURL: "docs/connectors/generic.md",
		Factory: func(cfg map[string]string) (sdk.NetworkConnector, error) {
			return generic.NewSNMP(cfg["target"], cfg["community"]), nil
		}})
	registry.Register(registry.Entry{Vendor: "Generic", ProductFamily: "REST", ConnectionType: sdk.ConnGenericHTTP,
		Protocols: []string{"https"}, AuthMethods: []string{"token", "basic"},
		ConnectorVersion: ConnectorVersion, Status: sdk.Partial,
		DocURL: "docs/connectors/generic.md",
		Factory: func(cfg map[string]string) (sdk.NetworkConnector, error) {
			return generic.NewREST(generic.RESTConfig{BaseURL: cfg["base_url"], Token: cfg["token"]}), nil
		}})
}
