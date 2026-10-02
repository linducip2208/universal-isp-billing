// Package incidents: event correlation (many alerts -> one incident),
// impact tracking, lifecycle, timeline, root cause, postmortem.
// Correlation is deterministic and testable: pure functions over alerts.
package incidents

import (
	"sort"
	"time"
)

type Severity string

const (
	SevCritical Severity = "critical"
	SevMajor    Severity = "major"
	SevMinor    Severity = "minor"
	SevWarning  Severity = "warning"
)

type Status string

const (
	Open         Status = "open"
	Acknowledged Status = "acknowledged"
	Mitigating   Status = "mitigating"
	Resolved     Status = "resolved"
	Closed       Status = "closed"
)

type TimelineEntry struct {
	At    time.Time `json:"at"`
	Actor string    `json:"actor"`
	Text  string    `json:"text"`
}

type Incident struct {
	ID               string          `json:"id"`
	OrgID            string          `json:"org_id"`
	Title            string          `json:"title"`
	Severity         Severity        `json:"severity"`
	Status           Status          `json:"status"`
	ScopeKey         string          `json:"scope_key"` // correlation scope, e.g. device:<id>
	AlertIDs         []string        `json:"alert_ids"`
	ImpactedServices []string        `json:"impacted_services,omitempty"`
	ImpactedSubs     int             `json:"impacted_subscribers"`
	RootCause        string          `json:"root_cause,omitempty"`
	Postmortem       string          `json:"postmortem,omitempty"`
	Assignee         string          `json:"assignee,omitempty"`
	Timeline         []TimelineEntry `json:"timeline"`
	OpenedAt         time.Time       `json:"opened_at"`
	ResolvedAt       *time.Time      `json:"resolved_at,omitempty"`
	AckedAt          *time.Time      `json:"acked_at,omitempty"`
}

func (in *Incident) AddEvent(actor, text string) {
	in.Timeline = append(in.Timeline, TimelineEntry{At: time.Now().UTC(), Actor: actor, Text: text})
}

func (in *Incident) Ack(actor string) {
	if in.Status != Open {
		return
	}
	now := time.Now().UTC()
	in.Status = Acknowledged
	in.AckedAt = &now
	in.AddEvent(actor, "acknowledged")
}

func (in *Incident) Resolve(actor, rootCause string) {
	if in.Status == Resolved || in.Status == Closed {
		return
	}
	now := time.Now().UTC()
	in.Status = Resolved
	in.ResolvedAt = &now
	in.RootCause = rootCause
	in.AddEvent(actor, "resolved: "+rootCause)
}

// Alert is the minimal correlatable alert shape.
type Alert struct {
	ID       string    `json:"id"`
	ScopeKey string    `json:"scope_key"`
	Severity Severity  `json:"severity"`
	Title    string    `json:"title"`
	At       time.Time `json:"at"`
}

// Correlate groups alerts into incidents: same scope within window merges;
// child scopes (onu/subscriber under a failed parent) attach to the parent
// incident instead of opening new ones. Returns new + updated incidents.
func Correlate(existing []*Incident, alerts []Alert, window time.Duration, childOf map[string]string) []*Incident {
	now := time.Now().UTC()
	byScope := map[string]*Incident{}
	for _, in := range existing {
		if in.Status == Resolved || in.Status == Closed {
			continue
		}
		byScope[in.ScopeKey] = in
	}
	for _, a := range alerts {
		scope := a.ScopeKey
		if parent, ok := childOf[scope]; ok {
			if pin, ok := byScope[parent]; ok {
				pin.AlertIDs = append(pin.AlertIDs, a.ID)
				pin.AddEvent("correlator", "correlated child alert: "+a.Title)
				continue
			}
			scope = parent
		}
		if in, ok := byScope[scope]; ok && now.Sub(in.OpenedAt) < window {
			in.AlertIDs = append(in.AlertIDs, a.ID)
			if sevRank(a.Severity) > sevRank(in.Severity) {
				in.Severity = a.Severity
			}
			continue
		}
		in := &Incident{ID: "inc-" + a.ID, OrgID: "", Title: a.Title, Severity: a.Severity,
			Status: Open, ScopeKey: scope, AlertIDs: []string{a.ID}, OpenedAt: now}
		in.AddEvent("correlator", "opened from alert: "+a.Title)
		existing = append(existing, in)
		byScope[scope] = in
	}
	sort.Slice(existing, func(i, j int) bool { return existing[i].OpenedAt.Before(existing[j].OpenedAt) })
	return existing
}

func sevRank(s Severity) int {
	switch s {
	case SevCritical:
		return 4
	case SevMajor:
		return 3
	case SevMinor:
		return 2
	default:
		return 1
	}
}
