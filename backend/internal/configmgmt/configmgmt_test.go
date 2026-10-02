package configmgmt_test

import (
	"strings"
	"testing"

	"github.com/universal-isp/platform/internal/configmgmt"
)

func TestDiffAndLifecycle(t *testing.T) {
	d := configmgmt.Diff("a\nb\nc", "a\nc\nd")
	if !strings.Contains(d, "+ d") || !strings.Contains(d, "- b") {
		t.Fatalf("diff=%q", d)
	}
	c := &configmgmt.Change{Status: "proposed"}
	if err := c.MarkApplied(); err == nil {
		t.Fatal("unapproved apply must fail")
	}
	if err := c.Approve("netops"); err != nil {
		t.Fatal(err)
	}
	if err := c.MarkApplied(); err != nil {
		t.Fatal(err)
	}
	if err := c.Rollback(); err != nil {
		t.Fatal(err)
	}
	if hits := configmgmt.ComplianceCheck("service telnet\npassword 123", []string{"service telnet"}); len(hits) != 1 {
		t.Fatalf("hits=%v", hits)
	}
}
