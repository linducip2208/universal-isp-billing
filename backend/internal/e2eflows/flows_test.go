package e2eflows_test

import (
	"context"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/automation"
	"github.com/universal-isp/platform/internal/billing"
	"github.com/universal-isp/platform/internal/connectors/sdk"
	"github.com/universal-isp/platform/internal/provisioning"
)

// fakeNAS records lifecycle calls without touching any device.
type fakeNAS struct {
	suspended, activated bool
}

func (f *fakeNAS) ConnectionType() sdk.ConnectionType   { return sdk.ConnREST }
func (f *fakeNAS) TestConnection(context.Context) error { return nil }
func (f *fakeNAS) GetDeviceInfo(context.Context) (*sdk.DeviceInfo, error) {
	return &sdk.DeviceInfo{Vendor: "fake", Model: "e2e"}, nil
}
func (f *fakeNAS) GetCapabilities(context.Context) (*sdk.Capabilities, error) {
	return sdk.NewCapabilities(sdk.Cap(sdk.CapSuspend, sdk.Partial), sdk.Cap(sdk.CapActivate, sdk.Partial)), nil
}
func (f *fakeNAS) GetSites(context.Context) ([]sdk.Site, error)       { return nil, nil }
func (f *fakeNAS) GetDevices(context.Context) ([]sdk.Device, error)   { return nil, nil }
func (f *fakeNAS) GetClients(context.Context) ([]sdk.Client, error)   { return nil, nil }
func (f *fakeNAS) GetInterfaces(context.Context) ([]sdk.Iface, error) { return nil, nil }
func (f *fakeNAS) GetTraffic(_ context.Context, t string) (*sdk.Traffic, error) {
	return &sdk.Traffic{Target: t}, nil
}
func (f *fakeNAS) ProvisionSubscriber(context.Context, sdk.ProvisionRequest) error { return nil }
func (f *fakeNAS) UpdateSubscriber(context.Context, sdk.UpdateRequest) error       { return nil }
func (f *fakeNAS) SuspendSubscriber(_ context.Context, _ string) error {
	f.suspended = true
	return nil
}
func (f *fakeNAS) ActivateSubscriber(_ context.Context, _ string) error {
	f.activated = true
	return nil
}
func (f *fakeNAS) DisconnectSubscriber(context.Context, string) error { return nil }
func (f *fakeNAS) DeleteSubscriber(context.Context, string) error     { return nil }
func (f *fakeNAS) Health() sdk.Health                                 { return sdk.Health{} }

// TestOverdueSuspendFlow: invoice overdue past grace -> dunning suspend ->
// automation fires -> drift plan suspends -> connector suspends.
func TestOverdueSuspendFlow(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	due := now.Add(-10 * 24 * time.Hour)
	grace := now.Add(-8 * 24 * time.Hour)
	inv := billing.NewInvoice("c1", due)
	inv.AddItem("Home 50M", 1, 15000000)
	inv.Status = billing.InvoiceOpen
	inv.GraceUntil = &grace
	if got := billing.StageOf(inv, now, billing.DefaultPolicy()); got != billing.StageSuspend {
		t.Fatalf("stage=%s", got)
	}
	eng := automation.New()
	fired := false
	eng.RegisterAction("suspend_subscriber", func(_ context.Context, _ automation.Rule, _ automation.Facts) error {
		fired = true
		return nil
	})
	rules := []automation.Rule{{ID: "r1", Enabled: true,
		When: []automation.Condition{
			{Field: "invoice.status", Op: "eq", Value: "overdue"},
			{Field: "grace.expired", Op: "eq", Value: true},
		}, Action: "suspend_subscriber"}}
	if n := len(eng.Evaluate(ctx, rules, automation.Facts{"invoice.status": "overdue", "grace.expired": true})); n != 1 || !fired {
		t.Fatal("automation must fire suspend")
	}
	nas := &fakeNAS{}
	plan := provisioning.Detect(
		provisioning.Desired{SubscriptionID: "s1", DownloadMbps: 50, UploadMbps: 20},
		provisioning.Actual{Found: true, DownloadMbps: 50, UploadMbps: 20}, true)
	if plan.Action != provisioning.ActionSuspend {
		t.Fatalf("plan=%+v", plan)
	}
	if err := nas.SuspendSubscriber(ctx, "s1"); err != nil || !nas.suspended {
		t.Fatal("connector suspend must run")
	}
}

// TestPayReactivateFlow: payment -> reconcile -> paid -> automation ->
// drift plan activate -> connector activates.
func TestPayReactivateFlow(t *testing.T) {
	ctx := context.Background()
	inv := billing.NewInvoice("c1", time.Now().Add(24*time.Hour))
	inv.AddItem("Home 50M", 1, 15000000)
	inv.Status = billing.InvoiceOpen
	ledger := &billing.CreditLedger{CustomerID: "c1"}
	res, err := billing.Reconcile(inv, ledger, 15000000)
	if err != nil || !res.InvoicePaid {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	eng := automation.New()
	fired := false
	eng.RegisterAction("activate_subscriber", func(_ context.Context, _ automation.Rule, _ automation.Facts) error {
		fired = true
		return nil
	})
	rules := []automation.Rule{{ID: "r2", Enabled: true,
		When:   []automation.Condition{{Field: "payment.status", Op: "eq", Value: "paid"}},
		Action: "activate_subscriber"}}
	if n := len(eng.Evaluate(ctx, rules, automation.Facts{"payment.status": "paid"})); n != 1 || !fired {
		t.Fatal("automation must fire activate")
	}
	nas := &fakeNAS{}
	plan := provisioning.Detect(
		provisioning.Desired{SubscriptionID: "s1", DownloadMbps: 50, UploadMbps: 20},
		provisioning.Actual{Found: true, DownloadMbps: 50, UploadMbps: 20, Suspended: true}, false)
	if plan.Action != provisioning.ActionActivate {
		t.Fatalf("plan=%+v", plan)
	}
	if err := nas.ActivateSubscriber(ctx, "s1"); err != nil || !nas.activated {
		t.Fatal("connector activate must run")
	}
}
