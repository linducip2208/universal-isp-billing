// Package alerts: severity levels, dedup keys, open/resolve lifecycle.
package alerts

import "time"

type Severity string

const (
	Info     Severity = "info"
	Warning  Severity = "warning"
	Critical Severity = "critical"
)

type Alert struct {
	ID         string     `json:"id"`
	OrgID      string     `json:"org_id"`
	Severity   Severity   `json:"severity"`
	Title      string     `json:"title"`
	Body       string     `json:"body,omitempty"`
	DedupKey   string     `json:"dedup_key"`
	Status     string     `json:"status"` // open | acked | resolved
	CreatedAt  time.Time  `json:"created_at"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
}

func (a *Alert) Ack() {
	if a.Status == "open" {
		a.Status = "acked"
	}
}

func (a *Alert) Resolve() {
	now := time.Now().UTC()
	a.Status = "resolved"
	a.ResolvedAt = &now
}
