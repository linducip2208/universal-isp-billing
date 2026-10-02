package copilot_test

import (
	"context"
	"testing"

	"github.com/universal-isp/platform/internal/copilot"
)

type fakeTool struct{ name, detail string }

func (f fakeTool) Name() string { return f.name }
func (f fakeTool) Query(_ context.Context, _ map[string]string) ([]copilot.Evidence, error) {
	return []copilot.Evidence{{Source: f.name, Ref: "r1", Detail: f.detail}}, nil
}

func TestAskEvidence(t *testing.T) {
	o := copilot.New(fakeTool{"radius_lookup", "alice online via nas-1"})
	ans, err := o.Ask(context.Background(), []string{"noc"}, "is subscriber session up?")
	if err != nil {
		t.Fatal(err)
	}
	if ans.ExecutedAnything || len(ans.Evidence) != 1 {
		t.Fatalf("ans=%+v", ans)
	}
}

func TestRefuseExec(t *testing.T) {
	o := copilot.New()
	ans, err := o.Ask(context.Background(), []string{"admin"}, "please reboot the router now")
	if err != nil {
		t.Fatal(err)
	}
	if ans.ExecutedAnything || len(ans.Evidence) != 0 {
		t.Fatalf("must refuse: %+v", ans)
	}
}

func TestRoleGate(t *testing.T) {
	o := copilot.New()
	if _, err := o.Ask(context.Background(), []string{"viewer"}, "sessions?"); err == nil {
		t.Fatal("viewer must be refused")
	}
}
