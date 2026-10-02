package topology_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/topology"
)

func fixture() *topology.Graph {
	g := &topology.Graph{}
	g.AddNode(topology.Node{ID: "core-1", Kind: topology.NodeRouter, Label: "core"})
	g.AddNode(topology.Node{ID: "olt-1", Kind: topology.NodeOLT, Label: "olt"})
	g.AddNode(topology.Node{ID: "onu-1", Kind: topology.NodeONU, Label: "onu1"})
	g.AddNode(topology.Node{ID: "onu-2", Kind: topology.NodeONU, Label: "onu2"})
	g.Link("core-1", "olt-1", topology.EdgeFiber)
	g.Link("olt-1", "onu-1", topology.EdgeFiber)
	g.Link("olt-1", "onu-2", topology.EdgeFiber)
	return g
}

func TestImpactOf(t *testing.T) {
	g := fixture()
	affected, byKind := g.ImpactOf("olt-1")
	if len(affected) != 2 {
		t.Fatalf("affected=%v", affected)
	}
	if byKind[topology.NodeONU] != 2 {
		t.Fatalf("byKind=%v", byKind)
	}
	if aff, _ := g.ImpactOf("onu-1"); len(aff) != 0 {
		t.Fatalf("leaf should affect nothing: %v", aff)
	}
}

func TestPathTo(t *testing.T) {
	g := fixture()
	p := g.PathTo("onu-2")
	if len(p) != 3 || p[0] != "core-1" || p[2] != "onu-2" {
		t.Fatalf("path=%v", p)
	}
}
