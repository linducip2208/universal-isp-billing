package provisioning_test

import (
	"context"
	"errors"
	"testing"

	"github.com/universal-isp/platform/internal/connectors/sdk"
	"github.com/universal-isp/platform/internal/provisioning"
)

type flaky struct{ n int }

func (f *flaky) ConnectionType() sdk.ConnectionType   { return sdk.ConnREST }
func (f *flaky) TestConnection(context.Context) error { return nil }
func (f *flaky) GetDeviceInfo(context.Context) (*sdk.DeviceInfo, error) {
	return &sdk.DeviceInfo{}, nil
}
func (f *flaky) GetCapabilities(context.Context) (*sdk.Capabilities, error) {
	return &sdk.Capabilities{}, nil
}
func (f *flaky) GetSites(context.Context) ([]sdk.Site, error)       { return nil, nil }
func (f *flaky) GetDevices(context.Context) ([]sdk.Device, error)   { return nil, nil }
func (f *flaky) GetClients(context.Context) ([]sdk.Client, error)   { return nil, nil }
func (f *flaky) GetInterfaces(context.Context) ([]sdk.Iface, error) { return nil, nil }
func (f *flaky) GetTraffic(_ context.Context, t string) (*sdk.Traffic, error) {
	return &sdk.Traffic{Target: t}, nil
}
func (f *flaky) ProvisionSubscriber(_ context.Context, _ sdk.ProvisionRequest) error {
	f.n++
	if f.n < 3 {
		return errors.New("boom")
	}
	return nil
}
func (f *flaky) UpdateSubscriber(context.Context, sdk.UpdateRequest) error { return nil }
func (f *flaky) SuspendSubscriber(context.Context, string) error           { return nil }
func (f *flaky) ActivateSubscriber(context.Context, string) error          { return nil }
func (f *flaky) DisconnectSubscriber(context.Context, string) error        { return nil }
func (f *flaky) DeleteSubscriber(context.Context, string) error            { return nil }
func (f *flaky) Health() sdk.Health                                        { return sdk.Health{} }

func TestRetry(t *testing.T) {
	c := &flaky{}
	w := &provisioning.Workflow{Connector: c, Provision: sdk.ProvisionRequest{SubscriberID: "s1", Username: "u", IdempotencyKey: "k"}}
	if err := provisioning.Execute(context.Background(), w, provisioning.ProvisionSubscriberWorkflow()); err != nil {
		t.Fatal(err)
	}
	if c.n != 3 {
		t.Fatalf("attempts=%d", c.n)
	}
}
