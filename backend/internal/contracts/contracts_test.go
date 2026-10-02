package contracts_test

import (
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/contracts"
)

func TestBreachWithExclusion(t *testing.T) {
	day := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	period := contracts.Window{Start: day, End: day.Add(24 * time.Hour)}
	outages := []contracts.Window{{Start: day.Add(2 * time.Hour), End: day.Add(3 * time.Hour)}}
	maint := []contracts.Window{{Start: day.Add(2 * time.Hour), End: day.Add(3 * time.Hour)}}
	pol := contracts.SLAPolicy{UptimePct: 99.5, MaintenanceExcl: true}
	if breached, up := contracts.Breach(pol, outages, period, maint); breached || up != 100 {
		t.Fatalf("excluded outage must not breach: %v %v", breached, up)
	}
	pol.MaintenanceExcl = false
	if breached, up := contracts.Breach(pol, outages, period, nil); !breached || up >= 99.5 {
		t.Fatalf("1h/24h must breach 99.5: %v %v", breached, up)
	}
}
