// Package contracts: B2B/wholesale contracts, SLA policies, timers and
// breach detection with maintenance-window exclusions.
package contracts

import (
	"time"
)

type Contract struct {
	ID         string    `json:"id"`
	OrgID      string    `json:"org_id"`
	CustomerID string    `json:"customer_id"`
	Kind       string    `json:"kind"` // dedicated | transit | vlan | wifi_managed | colo | wholesale
	SLAID      string    `json:"sla_id,omitempty"`
	StartsAt   time.Time `json:"starts_at"`
	EndsAt     time.Time `json:"ends_at"`
	MRCcents   int64     `json:"mrc_cents"`
	Status     string    `json:"status"`
}

type SLAPolicy struct {
	ID              string        `json:"id"`
	OrgID           string        `json:"org_id"`
	Name            string        `json:"name"`
	UptimePct       float64       `json:"uptime_pct"`
	ResponseSLA     time.Duration `json:"response_sla"`
	ResolutionSLA   time.Duration `json:"resolution_sla"`
	MaintenanceExcl bool          `json:"maintenance_exclusion"`
}

type Window struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Downtime accumulates outage minutes inside a period, excluding maintenance
// windows when the policy says so. Pure + testable.
func Downtime(outages []Window, period Window, maintenance []Window, exclude bool) time.Duration {
	var total time.Duration
	for _, o := range outages {
		s, e := maxTime(o.Start, period.Start), minTime(o.End, period.End)
		if !e.After(s) {
			continue
		}
		d := e.Sub(s)
		if exclude {
			for _, m := range maintenance {
				ms, me := maxTime(m.Start, s), minTime(m.End, e)
				if me.After(ms) {
					d -= me.Sub(ms)
				}
			}
		}
		if d > 0 {
			total += d
		}
	}
	return total
}

// Breach reports whether uptime target was missed and by how much.
func Breach(policy SLAPolicy, outages []Window, period Window, maintenance []Window) (breached bool, uptimePct float64) {
	down := Downtime(outages, period, maintenance, policy.MaintenanceExcl)
	span := period.End.Sub(period.Start)
	if span <= 0 {
		return false, 100
	}
	uptimePct = 100 * (1 - float64(down)/float64(span))
	return uptimePct < policy.UptimePct, uptimePct
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
func minTime(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
