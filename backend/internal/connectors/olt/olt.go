// Package olt registers OLT/FTTH families (Huawei, ZTE, Nokia, FiberHome,
// BDCOM, VSOL, C-Data, ...) via generic transports with honest capability states.
package olt

import (
	"context"
	"time"

	"github.com/universal-isp/platform/internal/connectors/sdk"
)

type dev struct {
	vendor, family string
	ct             sdk.ConnectionType
	cfg            map[string]string
}

func (d *dev) ConnectionType() sdk.ConnectionType { return d.ct }
func (d *dev) TestConnection(_ context.Context) error {
	if d.cfg["host"] == "" {
		return errMissing("host")
	}
	return nil
}
func (d *dev) GetDeviceInfo(_ context.Context) (*sdk.DeviceInfo, error) {
	return &sdk.DeviceInfo{Vendor: d.vendor, Model: d.family}, nil
}
func (d *dev) GetCapabilities(_ context.Context) (*sdk.Capabilities, error) {
	note := "vendor adapter PLANNED for " + d.vendor + "/" + d.family
	return &sdk.Capabilities{Items: []sdk.CapabilityInfo{
		{ID: sdk.CapOLT, Status: sdk.Planned, Note: note},
		{ID: sdk.CapONU, Status: sdk.Planned, Note: note},
		{ID: sdk.CapOpticalPower, Status: sdk.Planned, Note: note},
		{ID: sdk.CapProvisioning, Status: sdk.Planned, Note: note},
		{ID: sdk.CapVLAN, Status: sdk.Planned, Note: note},
	}}, nil
}
func (d *dev) GetSites(_ context.Context) ([]sdk.Site, error)       { return nil, nil }
func (d *dev) GetDevices(_ context.Context) ([]sdk.Device, error)   { return nil, nil }
func (d *dev) GetClients(_ context.Context) ([]sdk.Client, error)   { return nil, nil }
func (d *dev) GetInterfaces(_ context.Context) ([]sdk.Iface, error) { return nil, nil }
func (d *dev) GetTraffic(_ context.Context, t string) (*sdk.Traffic, error) {
	return &sdk.Traffic{Target: t, SampledAt: time.Now()}, nil
}
func (d *dev) ProvisionSubscriber(_ context.Context, r sdk.ProvisionRequest) error {
	if r.IdempotencyKey == "" {
		return errMissing("idempotency_key")
	}
	return missing("olt provision for " + d.vendor + "/" + d.family + " is PLANNED")
}
func (d *dev) UpdateSubscriber(_ context.Context, _ sdk.UpdateRequest) error {
	return missing("olt update for " + d.vendor + "/" + d.family + " is PLANNED")
}
func (d *dev) SuspendSubscriber(_ context.Context, _ string) error {
	return missing("olt suspend for " + d.vendor + "/" + d.family + " is PLANNED")
}
func (d *dev) ActivateSubscriber(_ context.Context, _ string) error {
	return missing("olt activate for " + d.vendor + "/" + d.family + " is PLANNED")
}
func (d *dev) DisconnectSubscriber(_ context.Context, _ string) error {
	return missing("olt disconnect for " + d.vendor + "/" + d.family + " is PLANNED")
}
func (d *dev) DeleteSubscriber(_ context.Context, _ string) error {
	return missing("olt delete for " + d.vendor + "/" + d.family + " is PLANNED")
}
func (d *dev) Health() sdk.Health {
	return sdk.Health{ConnectionType: d.ct, AuthType: "password/snmp", Vendor: d.vendor, Model: d.family}
}

type missing string

func (e missing) Error() string { return "CREDENTIAL_REQUIRED: " + string(e) }
func errMissing(k string) error { return missing(k) }

func Mk(vendor, family string, ct sdk.ConnectionType) func(map[string]string) (sdk.NetworkConnector, error) {
	return func(cfg map[string]string) (sdk.NetworkConnector, error) {
		return &dev{vendor: vendor, family: family, ct: ct, cfg: cfg}, nil
	}
}
