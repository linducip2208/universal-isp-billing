package ipam_test

import (
	"net/netip"
	"testing"

	"github.com/universal-isp/platform/internal/ipam"
)

func TestAllocateRelease(t *testing.T) {
	p := ipam.NewPool("p1", "o1", "cg-nat", netip.MustParsePrefix("100.64.0.0/24"), 100, "inet")
	if err := p.Reserve(netip.MustParseAddr("100.64.0.1")); err != nil {
		t.Fatal(err)
	}
	a1, err := p.AllocateNext("sub-1")
	if err != nil {
		t.Fatal(err)
	}
	if a1.String() == "100.64.0.1" {
		t.Fatal("reserved handed out")
	}
	a2, err := p.AllocateNext("sub-2")
	if err != nil {
		t.Fatal(err)
	}
	if a1 == a2 {
		t.Fatal("duplicate allocation")
	}
	if err := p.AllocateSpecific(a1, "sub-x"); err == nil {
		t.Fatal("double allocation must fail")
	}
	if err := p.AllocateSpecific(netip.MustParseAddr("100.64.0.77"), "sub-3"); err != nil {
		t.Fatal(err)
	}
	p.Release(a1)
	if _, ok := p.Owner(a1); ok {
		t.Fatal("release failed")
	}
	if _, err := p.AllocateNext(""); err == nil {
		t.Fatal("owner required")
	}
	used, total := p.Utilization()
	if used != 2 || total != 256 {
		t.Fatalf("util=%d/%d", used, total)
	}
}

func TestOutsidePrefix(t *testing.T) {
	p := ipam.NewPool("p1", "o1", "x", netip.MustParsePrefix("10.0.0.0/30"), 0, "")
	if err := p.AllocateSpecific(netip.MustParseAddr("10.0.1.1"), "s"); err == nil {
		t.Fatal("outside prefix must fail")
	}
}
