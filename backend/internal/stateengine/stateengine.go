// Package stateengine: generic desired-vs-actual state management for ANY
// resource kind (subscriber, device, VLAN, ONU, RADIUS user...). It computes
// classified drift, explains it, builds approval-gated reconciliation plans,
// executes with verify, and records everything. Never modifies anything
// without an explicit approved plan.
package stateengine

import (
	"context"
	"fmt"
	"sort"
	"time"
)

type DriftKind string

const (
	DriftMissing DriftKind = "MISSING" // desired, absent actual
	DriftWrong   DriftKind = "WRONG_VALUE"
	DriftExtra   DriftKind = "UNEXPECTED" // present actual, absent desired
	DriftStale   DriftKind = "STALE"      // actual older than max age
	DriftUnknown DriftKind = "UNKNOWN"
	DriftNone    DriftKind = "NONE"
)

type Resource struct {
	Kind     string            `json:"kind"`
	ID       string            `json:"id"`
	Desired  map[string]string `json:"desired"`
	Actual   map[string]string `json:"actual,omitempty"`
	ActualAt *time.Time        `json:"actual_at,omitempty"`
}

type Drift struct {
	Kind     DriftKind `json:"kind"`
	Resource string    `json:"resource"`
	Field    string    `json:"field,omitempty"`
	Want     string    `json:"want,omitempty"`
	Got      string    `json:"got,omitempty"`
	Reason   string    `json:"reason"`
}

// Detect compares desired vs actual with STALE awareness.
func Detect(r Resource, maxAge time.Duration, now time.Time) []Drift {
	if len(r.Desired) == 0 && len(r.Actual) == 0 {
		return []Drift{{Kind: DriftNone, Resource: r.ID, Reason: "no desired or actual state"}}
	}
	if len(r.Actual) == 0 {
		return []Drift{{Kind: DriftMissing, Resource: r.ID, Reason: "desired but absent on device"}}
	}
	var out []Drift
	if maxAge > 0 && r.ActualAt != nil && now.Sub(*r.ActualAt) > maxAge {
		out = append(out, Drift{Kind: DriftStale, Resource: r.ID,
			Reason: fmt.Sprintf("actual state older than %s", maxAge)})
	}
	fields := map[string]bool{}
	for k := range r.Desired {
		fields[k] = true
	}
	for k := range r.Actual {
		fields[k] = true
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		want, wok := r.Desired[k]
		got, gok := r.Actual[k]
		switch {
		case wok && !gok:
			out = append(out, Drift{Kind: DriftMissing, Resource: r.ID, Field: k, Want: want, Reason: "field desired but absent"})
		case !wok && gok:
			out = append(out, Drift{Kind: DriftExtra, Resource: r.ID, Field: k, Got: got, Reason: "field present but not desired"})
		case want != got:
			out = append(out, Drift{Kind: DriftWrong, Resource: r.ID, Field: k, Want: want, Got: got,
				Reason: fmt.Sprintf("%s should be %q, is %q", k, want, got)})
		}
	}
	if len(out) == 0 {
		return []Drift{{Kind: DriftNone, Resource: r.ID, Reason: "in sync"}}
	}
	return out
}

type Plan struct {
	Resource string   `json:"resource"`
	Actions  []string `json:"actions"`
	Approved bool     `json:"approved"`
	By       string   `json:"approved_by,omitempty"`
}

// Propose builds human-readable correction actions from drift (no execution).
func Propose(drifts []Drift) Plan {
	p := Plan{}
	for _, d := range drifts {
		switch d.Kind {
		case DriftNone:
		case DriftMissing:
			p.Actions = append(p.Actions, fmt.Sprintf("create %s %s=%s", d.Resource, d.Field, d.Want))
		case DriftWrong:
			p.Actions = append(p.Actions, fmt.Sprintf("set %s %s=%s (was %s)", d.Resource, d.Field, d.Want, d.Got))
		case DriftExtra:
			p.Actions = append(p.Actions, fmt.Sprintf("remove %s %s (undesired)", d.Resource, d.Field))
		case DriftStale:
			p.Actions = append(p.Actions, fmt.Sprintf("re-poll %s", d.Resource))
		default:
			p.Actions = append(p.Actions, fmt.Sprintf("investigate %s: %s", d.Resource, d.Reason))
		}
		if d.Resource != "" && p.Resource == "" {
			p.Resource = d.Resource
		}
	}
	return p
}

// Executor applies one action string; Verify confirms the outcome.
// Implementations are connector-backed and audited by callers.
type Executor interface {
	Apply(ctx context.Context, action string) error
	Verify(ctx context.Context, resource string) error
}

// Reconcile runs an APPROVED plan: apply -> verify per action, stop on first
// failure (rollback is the caller's compensation workflow).
func Reconcile(ctx context.Context, ex Executor, plan Plan) error {
	if !plan.Approved {
		return fmt.Errorf("plan for %s requires approval", plan.Resource)
	}
	for _, a := range plan.Actions {
		if err := ex.Apply(ctx, a); err != nil {
			return fmt.Errorf("apply %q: %w", a, err)
		}
		if err := ex.Verify(ctx, plan.Resource); err != nil {
			return fmt.Errorf("verify %s: %w", plan.Resource, err)
		}
	}
	return nil
}
