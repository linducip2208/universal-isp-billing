// Package configmgmt: "Git for network" — snapshots, diffs, approvals,
// scheduled deployment inside maintenance windows, rollback, compliance checks.
package configmgmt

import (
	"errors"
	"strings"
	"time"
)

type Snapshot struct {
	ID       string    `json:"id"`
	OrgID    string    `json:"org_id"`
	DeviceID string    `json:"device_id"`
	Version  int       `json:"version"`
	Body     string    `json:"body"`
	TakenAt  time.Time `json:"taken_at"`
	TakenBy  string    `json:"taken_by"`
}

type Change struct {
	ID          string     `json:"id"`
	OrgID       string     `json:"org_id"`
	DeviceID    string     `json:"device_id"`
	Summary     string     `json:"summary"`
	Diff        string     `json:"diff"`
	Status      string     `json:"status"` // proposed | approved | scheduled | applied | rolled_back | rejected
	ProposedBy  string     `json:"proposed_by"`
	ApprovedBy  string     `json:"approved_by,omitempty"`
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
}

// Diff renders a simple line diff (added/removed) between two snapshots.
// Deterministic and dependency-free.
func Diff(oldBody, newBody string) string {
	oldLines := map[string]bool{}
	for _, l := range strings.Split(oldBody, "\n") {
		oldLines[strings.TrimSpace(l)] = true
	}
	var sb strings.Builder
	for _, l := range strings.Split(newBody, "\n") {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if !oldLines[t] {
			sb.WriteString("+ " + t + "\n")
		}
	}
	newLines := map[string]bool{}
	for _, l := range strings.Split(newBody, "\n") {
		newLines[strings.TrimSpace(l)] = true
	}
	for _, l := range strings.Split(oldBody, "\n") {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if !newLines[t] {
			sb.WriteString("- " + t + "\n")
		}
	}
	return sb.String()
}

func (c *Change) Approve(by string) error {
	if c.Status != "proposed" {
		return errors.New("only proposed changes can be approved")
	}
	c.Status = "approved"
	c.ApprovedBy = by
	return nil
}

func (c *Change) MarkApplied() error {
	if c.Status != "approved" && c.Status != "scheduled" {
		return errors.New("change must be approved/scheduled before apply")
	}
	c.Status = "applied"
	return nil
}

func (c *Change) Rollback() error {
	if c.Status != "applied" {
		return errors.New("only applied changes can roll back")
	}
	c.Status = "rolled_back"
	return nil
}

// ComplianceCheck flags forbidden lines (e.g. plaintext secrets, open telnet).
func ComplianceCheck(body string, forbidden []string) []string {
	var hits []string
	low := strings.ToLower(body)
	for _, f := range forbidden {
		if strings.Contains(low, strings.ToLower(f)) {
			hits = append(hits, f)
		}
	}
	return hits
}
