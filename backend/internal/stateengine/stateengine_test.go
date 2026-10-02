package stateengine_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/stateengine"
)

func TestDetectClassify(t *testing.T) {
	now := time.Now()
	old := now.Add(-time.Hour)
	r := stateengine.Resource{Kind: "sub", ID: "s1",
		Desired:  map[string]string{"speed": "100", "vlan": "10"},
		Actual:   map[string]string{"speed": "20", "extra": "x"},
		ActualAt: &old}
	d := stateengine.Detect(r, 30*time.Minute, now)
	kinds := map[stateengine.DriftKind]int{}
	for _, x := range d {
		kinds[x.Kind]++
	}
	if kinds[stateengine.DriftWrong] != 1 || kinds[stateengine.DriftMissing] != 1 ||
		kinds[stateengine.DriftExtra] != 1 || kinds[stateengine.DriftStale] != 1 {
		t.Fatalf("drifts=%+v", d)
	}
	p := stateengine.Propose(d)
	if !p.Approved && len(p.Actions) != 4 {
		t.Fatalf("plan=%+v", p)
	}
	if err := stateengine.Reconcile(context.Background(), nil, p); err == nil {
		t.Fatal("unapproved plan must refuse")
	}
}

type fakeEx struct {
	applied  []string
	failOn   string
	verified int
}

func (f *fakeEx) Apply(_ context.Context, a string) error {
	f.applied = append(f.applied, a)
	if a == f.failOn {
		return errors.New("device error")
	}
	return nil
}
func (f *fakeEx) Verify(_ context.Context, _ string) error {
	f.verified++
	return nil
}

func TestReconcile(t *testing.T) {
	p := stateengine.Plan{Resource: "s1", Actions: []string{"a1", "a2"}, Approved: true, By: "netops"}
	fx := &fakeEx{}
	if err := stateengine.Reconcile(context.Background(), fx, p); err != nil {
		t.Fatal(err)
	}
	if len(fx.applied) != 2 || fx.verified != 2 {
		t.Fatalf("fx=%+v", fx)
	}
	fx2 := &fakeEx{failOn: "a1"}
	if err := stateengine.Reconcile(context.Background(), fx2, p); err == nil {
		t.Fatal("failure must stop")
	}
	if len(fx2.applied) != 1 {
		t.Fatalf("must stop at first failure: %+v", fx2.applied)
	}
}
