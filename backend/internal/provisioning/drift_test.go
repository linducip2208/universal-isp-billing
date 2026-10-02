package provisioning_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/provisioning"
)

func TestDrift(t *testing.T) {
	want := provisioning.Desired{SubscriptionID: "s1", DownloadMbps: 100, UploadMbps: 50}
	if p := provisioning.Detect(want, provisioning.Actual{Found: false}, false); p.Action != provisioning.ActionCreate {
		t.Fatalf("plan=%+v", p)
	}
	got := provisioning.Actual{Found: true, DownloadMbps: 50, UploadMbps: 50}
	if p := provisioning.Detect(want, got, false); p.Action != provisioning.ActionUpdate {
		t.Fatalf("plan=%+v", p)
	}
	same := provisioning.Actual{Found: true, DownloadMbps: 100, UploadMbps: 50}
	if p := provisioning.Detect(want, same, false); p.Action != provisioning.ActionNone {
		t.Fatalf("plan=%+v", p)
	}
	if p := provisioning.Detect(want, same, true); p.Action != provisioning.ActionSuspend {
		t.Fatalf("plan=%+v", p)
	}
	susp := provisioning.Actual{Found: true, DownloadMbps: 100, UploadMbps: 50, Suspended: true}
	if p := provisioning.Detect(want, susp, false); p.Action != provisioning.ActionActivate {
		t.Fatalf("plan=%+v", p)
	}
}
