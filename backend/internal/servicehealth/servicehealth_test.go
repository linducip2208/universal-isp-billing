package servicehealth_test

import (
	"context"
	"testing"

	"github.com/universal-isp/platform/internal/servicehealth"
)

func TestComposeWorstWins(t *testing.T) {
	ok := func(_ context.Context, _ string) servicehealth.Section {
		return servicehealth.Section{Source: "radius", Status: "ok", Detail: "session up"}
	}
	down := func(_ context.Context, _ string) servicehealth.Section {
		return servicehealth.Section{Source: "onu", Status: "down", Detail: "rx -31dBm"}
	}
	r := servicehealth.Compose(context.Background(), "s1", nil)
	if r.Overall != "ok" {
		t.Fatalf("empty must be ok: %+v", r)
	}
	r2 := servicehealth.Compose(context.Background(), "s1", []servicehealth.Fetcher{ok, down})
	if r2.Overall != "down" || len(r2.Sections) != 2 {
		t.Fatalf("r=%+v", r2)
	}
}
