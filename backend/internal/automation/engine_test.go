package automation_test

import (
	"context"
	"testing"

	"github.com/universal-isp/platform/internal/automation"
)

func TestOverdueSuspendRule(t *testing.T) {
	e := automation.New()
	fired := false
	e.RegisterAction("suspend_subscriber", func(_ context.Context, _ automation.Rule, _ automation.Facts) error {
		fired = true
		return nil
	})
	rules := []automation.Rule{{
		ID: "r1", Name: "suspend on overdue", Enabled: true,
		When: []automation.Condition{
			{Field: "invoice.status", Op: "eq", Value: "overdue"},
			{Field: "grace.expired", Op: "eq", Value: true},
		},
		Action: "suspend_subscriber",
	}}
	got := e.Evaluate(context.Background(), rules, automation.Facts{"invoice.status": "overdue", "grace.expired": true})
	if len(got) != 1 || !fired {
		t.Fatal("rule should fire")
	}
	got = e.Evaluate(context.Background(), rules, automation.Facts{"invoice.status": "paid", "grace.expired": true})
	if len(got) != 0 {
		t.Fatal("paid invoice must not fire")
	}
}
