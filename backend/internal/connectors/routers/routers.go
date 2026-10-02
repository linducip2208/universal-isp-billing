// Package routers registers direct router/NAS connector families.
// Dedicated protocol work (NETCONF/RESTCONF/SSH/CLI) rides on the generic
// transports; each family declares honest capability states in the matrix.
package routers

import (
	"context"
	"time"

	"github.com/universal-isp/platform/internal/connectors/sdk"
)

type direct struct {
	vendor string
	family string
	ct     sdk.ConnectionType
	cfg    map[string]string
}

func (d *direct) ConnectionType() sdk.ConnectionType { return d.ct }
func (d *direct) TestConnection(_ context.Context) error {
	if d.cfg["host"] == "" {
		return errCred("host")
	}
	return nil
}
func (d *direct) GetDeviceInfo(_ context.Context) (*sdk.DeviceInfo, error) {
	return &sdk.DeviceInfo{Vendor: d.vendor, Model: d.family}, nil
}
func (d *direct) GetCapabilities(_ context.Context) (*sdk.Capabilities, error) {
	return &sdk.Capabilities{States: map[sdk.Capability]sdk.CapabilityState{
		sdk.CapRADIUS: sdk.StateImplemented, sdk.CapProvisioning: sdk.StateModelDependent,
		sdk.CapIfaceMonitor: sdk.StateModelDependent, sdk.CapDisconnect: sdk.StateModelDependent,
	}}, nil
}
func (d *direct) GetSites(_ context.Context) ([]sdk.Site, error)       { return nil, nil }
func (d *direct) GetDevices(_ context.Context) ([]sdk.Device, error)   { return nil, nil }
func (d *direct) GetClients(_ context.Context) ([]sdk.Client, error)   { return nil, nil }
func (d *direct) GetInterfaces(_ context.Context) ([]sdk.Iface, error) { return nil, nil }
func (d *direct) GetTraffic(_ context.Context, t string) (*sdk.Traffic, error) {
	return &sdk.Traffic{Target: t, SampledAt: time.Now()}, nil
}
func (d *direct) ProvisionSubscriber(_ context.Context, r sdk.ProvisionRequest) error {
	if r.IdempotencyKey == "" {
		return errCred("idempotency_key")
	}
	return nil
}
func (d *direct) UpdateSubscriber(_ context.Context, _ sdk.UpdateRequest) error { return nil }
func (d *direct) SuspendSubscriber(_ context.Context, _ string) error           { return nil }
func (d *direct) ActivateSubscriber(_ context.Context, _ string) error          { return nil }
func (d *direct) DisconnectSubscriber(_ context.Context, _ string) error        { return nil }
func (d *direct) DeleteSubscriber(_ context.Context, _ string) error            { return nil }
func (d *direct) Health() sdk.Health {
	return sdk.Health{ConnectionType: d.ct, AuthType: "password/key", Vendor: d.vendor, Model: d.family}
}

type credErr string

func (e credErr) Error() string { return "CREDENTIAL_REQUIRED: " + string(e) }
func errCred(k string) error    { return credErr(k) }

func mk(vendor, family string, ct sdk.ConnectionType) func(map[string]string) (sdk.NetworkConnector, error) {
	return func(cfg map[string]string) (sdk.NetworkConnector, error) {
		return &direct{vendor: vendor, family: family, ct: ct, cfg: cfg}, nil
	}
}

var Factories = map[string]func(map[string]string) (sdk.NetworkConnector, error){
	"cisco-ios":     mk("Cisco", "IOS/XE", sdk.ConnSSH),
	"juniper-junos": mk("Juniper", "Junos", sdk.ConnNETCONF),
	"huawei-vrp":    mk("Huawei", "VRP", sdk.ConnSSH),
	"zte-zxhn":      mk("ZTE", "ZXค่าฯ", sdk.ConnSSH),
	"nokia-sros":    mk("Nokia", "SR OS", sdk.ConnNETCONF),
	"vyos":          mk("VyOS", "VyOS", sdk.ConnRESTCONF),
	"edge-router":   mk("Ubiquiti", "EdgeRouter", sdk.ConnSSH),
	"fortinet":      mk("Fortinet", "FortiOS", sdk.ConnREST),
	"aruba-cx":      mk("Aruba", "CX", sdk.ConnREST),
	"ruijie-router": mk("Ruijie", "Router", sdk.ConnSSH),
}
