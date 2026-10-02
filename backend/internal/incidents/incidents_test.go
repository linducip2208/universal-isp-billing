package incidents_test

import (
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/incidents"
)

func TestCorrelateMergesSameScope(t *testing.T) {
	now := time.Now()
	alerts := []incidents.Alert{
		{ID: "a1", ScopeKey: "device:olt-1", Severity: incidents.SevCritical, Title: "OLT down", At: now},
		{ID: "a2", ScopeKey: "device:olt-1", Severity: incidents.SevMajor, Title: "OLT interfaces down", At: now.Add(time.Minute)},
	}
	got := incidents.Correlate(nil, alerts, 15*time.Minute, nil)
	if len(got) != 1 {
		t.Fatalf("want 1 incident, got %d", len(got))
	}
	if len(got[0].AlertIDs) != 2 || got[0].Severity != incidents.SevCritical {
		t.Fatalf("inc=%+v", got[0])
	}
}

func TestCorrelateChildAbsorbed(t *testing.T) {
	now := time.Now()
	alerts := []incidents.Alert{
		{ID: "a1", ScopeKey: "device:olt-1", Severity: incidents.SevCritical, Title: "OLT down", At: now},
		{ID: "a2", ScopeKey: "onu:serial-9", Severity: incidents.SevMajor, Title: "ONU unreachable", At: now.Add(time.Minute)},
		{ID: "a3", ScopeKey: "sub:s-42", Severity: incidents.SevWarning, Title: "session terminated", At: now.Add(2 * time.Minute)},
	}
	childOf := map[string]string{"onu:serial-9": "device:olt-1", "sub:s-42": "device:olt-1"}
	got := incidents.Correlate(nil, alerts, 15*time.Minute, childOf)
	if len(got) != 1 || len(got[0].AlertIDs) != 3 {
		t.Fatalf("1 incident with 3 alerts expected, got %+v", got)
	}
}

func TestLifecycle(t *testing.T) {
	in := &incidents.Incident{ID: "i1", Status: incidents.Open}
	in.Ack("noc")
	if in.Status != incidents.Acknowledged || in.AckedAt == nil {
		t.Fatal("ack failed")
	}
	in.Resolve("noc", "fiber cut")
	if in.Status != incidents.Resolved || in.RootCause != "fiber cut" {
		t.Fatal("resolve failed")
	}
	if len(in.Timeline) != 2 {
		t.Fatalf("timeline=%d", len(in.Timeline))
	}
}
