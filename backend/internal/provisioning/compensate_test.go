package provisioning_test

import (
	"context"
	"errors"
	"testing"

	"github.com/universal-isp/platform/internal/provisioning"
)

func TestCompensateReverseOrder(t *testing.T) {
	var order []string
	mk := func(name string, fail bool, comp bool) provisioning.Step {
		s := provisioning.Step{Name: name, Run: func(_ context.Context, _ *provisioning.Workflow) error {
			order = append(order, "run:"+name)
			if fail {
				return errors.New("boom")
			}
			return nil
		}}
		if comp {
			nm := name
			s.Compensate = func(_ context.Context, _ *provisioning.Workflow) error {
				order = append(order, "undo:"+nm)
				return nil
			}
		}
		return s
	}
	w := &provisioning.Workflow{}
	err := provisioning.Execute(context.Background(), w,
		[]provisioning.Step{mk("a", false, true), mk("b", false, true), mk("c", true, false)})
	if err == nil {
		t.Fatal("must fail")
	}
	want := []string{"run:a", "run:b", "run:c", "run:c", "run:c", "run:c", "run:c", "undo:b", "undo:a"}
	if len(order) != len(want) {
		t.Fatalf("order=%v", order)
	}
	for i := range want {
		if order[i] != want[i] {
			t.Fatalf("order=%v", order)
		}
	}
	// audit trail completeness
	var failed, compensated, ok int
	for _, r := range w.RunLog {
		switch r.Result {
		case "failed":
			failed++
		case "compensated":
			compensated++
		case "ok":
			ok++
		}
	}
	if failed != 1 || compensated != 2 || ok != 2 {
		t.Fatalf("runlog=%+v", w.RunLog)
	}
}
