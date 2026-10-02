package monitoring_test

import (
	"context"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/connectors/sdk"
	"github.com/universal-isp/platform/internal/devices"
	"github.com/universal-isp/platform/internal/monitoring"
)

type okConn struct{}

func (okConn) ConnectionType() sdk.ConnectionType   { return sdk.ConnREST }
func (okConn) TestConnection(context.Context) error { return nil }
func (okConn) GetDeviceInfo(context.Context) (*sdk.DeviceInfo, error) {
	return &sdk.DeviceInfo{Vendor: "T", CPUPercent: 11, MemoryPercent: 22, UptimeSecs: 99}, nil
}
func (okConn) GetCapabilities(context.Context) (*sdk.Capabilities, error) {
	return &sdk.Capabilities{}, nil
}
func (okConn) GetSites(context.Context) ([]sdk.Site, error)       { return nil, nil }
func (okConn) GetDevices(context.Context) ([]sdk.Device, error)   { return nil, nil }
func (okConn) GetClients(context.Context) ([]sdk.Client, error)   { return nil, nil }
func (okConn) GetInterfaces(context.Context) ([]sdk.Iface, error) { return nil, nil }
func (okConn) GetTraffic(_ context.Context, tgt string) (*sdk.Traffic, error) {
	return &sdk.Traffic{Target: tgt}, nil
}
func (okConn) ProvisionSubscriber(context.Context, sdk.ProvisionRequest) error { return nil }
func (okConn) UpdateSubscriber(context.Context, sdk.UpdateRequest) error       { return nil }
func (okConn) SuspendSubscriber(context.Context, string) error                 { return nil }
func (okConn) ActivateSubscriber(context.Context, string) error                { return nil }
func (okConn) DisconnectSubscriber(context.Context, string) error              { return nil }
func (okConn) DeleteSubscriber(context.Context, string) error                  { return nil }
func (okConn) Health() sdk.Health                                              { return sdk.Health{} }

func TestPollAll(t *testing.T) {
	srcs := []monitoring.Source{
		{Device: devices.Device{ID: "d1"}, Connector: okConn{}},
		{Device: devices.Device{ID: "d2"}, Connector: okConn{}},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	got := monitoring.PollAll(ctx, srcs, 1, nil)
	if len(got) != 2 || !got[0].Online || got[0].CPUPct != 11 {
		t.Fatalf("bad samples %+v", got)
	}
}
