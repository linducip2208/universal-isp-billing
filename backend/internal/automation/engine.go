package automation

import (
	"context"
	"fmt"
	"time"
)

// Rule: IF <conditions> THEN <action>. Conditions are evaluated against Facts.
type Condition struct {
	Field string `json:"field"` // e.g. "invoice.status", "device.status", "ap.offline_minutes", "cpu"
	Op    string `json:"op"`    // eq | neq | gt | gte | lt | lte
	Value any    `json:"value"`
}

type Rule struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	When    []Condition       `json:"when"`
	Action  string            `json:"action"` // suspend_subscriber | activate_subscriber | create_alert | notify_noc
	Params  map[string]string `json:"params,omitempty"`
	Enabled bool              `json:"enabled"`
}

type Facts map[string]any

func num(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}

func EvalCond(f Facts, c Condition) bool {
	actual, ok := f[c.Field]
	if !ok {
		return false
	}
	switch c.Op {
	case "eq":
		return fmt.Sprint(actual) == fmt.Sprint(c.Value)
	case "neq":
		return fmt.Sprint(actual) != fmt.Sprint(c.Value)
	case "gt", "gte", "lt", "lte":
		a, ok1 := num(actual)
		b, ok2 := num(c.Value)
		if !ok1 || !ok2 {
			return false
		}
		switch c.Op {
		case "gt":
			return a > b
		case "gte":
			return a >= b
		case "lt":
			return a < b
		case "lte":
			return a <= b
		}
	}
	return false
}

type ActionFunc func(ctx context.Context, rule Rule, facts Facts) error

type Engine struct {
	actions map[string]ActionFunc
	// Circuit breaker: max firings per rule per window (0 = unlimited).
	// Approval-required actions never auto-execute (see NeedsApproval).
	MaxFirings int
	Window     time.Duration
	firedAt    map[string][]time.Time
	approvals  map[string]bool
}

func New() *Engine {
	return &Engine{actions: map[string]ActionFunc{}, firedAt: map[string][]time.Time{}, approvals: map[string]bool{}}
}

func (e *Engine) RegisterAction(name string, fn ActionFunc) { e.actions[name] = fn }

// RequireApproval marks an action as approval-only: Evaluate records the
// firing but does NOT invoke the action until Approve runs it explicitly.
func (e *Engine) RequireApproval(action string) { e.approvals[action] = true }

// Approve executes a previously-would-fire action exactly once per call.
func (e *Engine) Approve(ctx context.Context, rule Rule, facts Facts) error {
	fn, ok := e.actions[rule.Action]
	if !ok {
		return fmt.Errorf("no action %q", rule.Action)
	}
	return fn(ctx, rule, facts)
}

// DryRun returns rules that WOULD fire without invoking any action.
func (e *Engine) DryRun(rules []Rule, facts Facts) []string {
	var out []string
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		match := true
		for _, c := range r.When {
			if !EvalCond(facts, c) {
				match = false
				break
			}
		}
		if match {
			out = append(out, r.ID)
		}
	}
	return out
}

// Evaluate returns fired rules; each fired rule's action is invoked unless
// approval-gated or breaker-tripped.
func (e *Engine) Evaluate(ctx context.Context, rules []Rule, facts Facts) []string {
	var fired []string
	for _, r := range rules {
		if !r.Enabled {
			continue
		}
		match := true
		for _, c := range r.When {
			if !EvalCond(facts, c) {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		if e.approvals[r.Action] {
			fired = append(fired, r.ID+":approval-required")
			continue
		}
		if e.MaxFirings > 0 && !e.breakerAllow(r.ID) {
			fired = append(fired, r.ID+":breaker-open")
			continue
		}
		fired = append(fired, r.ID)
		if fn, ok := e.actions[r.Action]; ok {
			_ = fn(ctx, r, facts)
		}
	}
	return fired
}

func (e *Engine) breakerAllow(id string) bool {
	now := time.Now()
	var keep []time.Time
	for _, t := range e.firedAt[id] {
		if now.Sub(t) < e.Window {
			keep = append(keep, t)
		}
	}
	if len(keep) >= e.MaxFirings {
		e.firedAt[id] = keep
		return false
	}
	e.firedAt[id] = append(keep, now)
	return true
}
