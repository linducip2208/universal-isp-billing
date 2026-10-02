package crm_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/crm"
)

func TestPipeline(t *testing.T) {
	l := &crm.Lead{Stage: crm.StageLead}
	if err := l.Advance(crm.Quoted); err == nil {
		t.Fatal("skip stages must fail")
	}
	for _, s := range []crm.Stage{crm.Prospect, crm.Surveyed, crm.Quoted, crm.Installation} {
		if err := l.Advance(s); err != nil {
			t.Fatalf("->%s: %v", s, err)
		}
	}
	if err := l.Convert("c-1"); err != nil {
		t.Fatal(err)
	}
	if l.Stage != crm.Won || l.CustomerID != "c-1" {
		t.Fatalf("lead=%+v", l)
	}
	l2 := &crm.Lead{Stage: crm.StageLead}
	if err := l2.Convert("c-9"); err == nil {
		t.Fatal("early convert must fail")
	}
}
