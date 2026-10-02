package automation_test

import (
	"context"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/automation"
)

func rule() automation.Rule {
	return automation.Rule{ID: "r1", Enabled: true,
		When:   []automation.Condition{{Field: "x", Op: "eq", Value: 1}},
		Action: "act"}
}

func TestBreaker(t *testing.T) {
	e := automation.New()
	e.MaxFirings = 2
	e.Window = time.Minute
	n := 0
	e.RegisterAction("act", func(_ context.Context, _ automation.Rule, _ automation.Facts) error {
		n++
		return nil
	})
	facts := automation.Facts{"x": 1}
	e.Evaluate(context.Background(), []automation.Rule{rule()}, facts)
	e.Evaluate(context.Background(), []automation.Rule{rule()}, facts)
	got := e.Evaluate(context.Background(), []automation.Rule{rule()}, facts)
	if n != 2 {
		t.Fatalf("action ran %d times", n)
	}
	if len(got) != 1 || got[0] != "r1:breaker-open" {
		t.Fatalf("breaker must report open: %v", got)
	}
}

func TestApprovalAndDryRun(t *testing.T) {
	e := automation.New()
	ran := false
	e.RegisterAction("act", func(_ context.Context, _ automation.Rule, _ automation.Facts) error {
		ran = true
		return nil
	})
	e.RequireApproval("act")
	facts := automation.Facts{"x": 1}
	if dry := e.DryRun([]automation.Rule{rule()}, facts); len(dry) != 1 {
		t.Fatalf("dry=%v", dry)
	}
	got := e.Evaluate(context.Background(), []automation.Rule{rule()}, facts)
	if ran || len(got) != 1 || got[0] != "r1:approval-required" {
		t.Fatalf("must gate: ran=%v got=%v", ran, got)
	}
	if err := e.Approve(context.Background(), rule(), facts); err != nil || !ran {
		t.Fatalf("approve: %v ran=%v", err, ran)
	}
}
