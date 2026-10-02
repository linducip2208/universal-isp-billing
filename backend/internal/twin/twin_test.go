package twin_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/topology"
	"github.com/universal-isp/platform/internal/twin"
)

func snap() *twin.Snapshot {
	g := &topology.Graph{}
	g.AddNode(topology.Node{ID: "core", Kind: topology.NodeRouter})
	g.AddNode(topology.Node{ID: "olt", Kind: topology.NodeOLT})
	g.AddNode(topology.Node{ID: "onu1", Kind: topology.NodeONU})
	g.Link("core", "olt", topology.EdgeFiber)
	g.Link("olt", "onu1", topology.EdgeFiber)
	return &twin.Snapshot{Graph: g,
		Status: map[string]string{"core": "online", "olt": "online", "onu1": "online"},
		Load:   map[string]float64{"core>olt": 0.92, "olt>onu1": 0.2}}
}

func TestSimulateFailure(t *testing.T) {
	r := twin.SimulateFailure(snap(), "olt")
	if len(r.Affected) != 1 || r.ByKind[topology.NodeONU] != 1 {
		t.Fatalf("r=%+v", r)
	}
	r2 := twin.SimulateFailure(snap(), "onu1")
	if len(r2.Affected) != 0 {
		t.Fatalf("leaf: %+v", r2)
	}
}

func TestHotLinksAndHeadroom(t *testing.T) {
	s := snap()
	if hot := twin.HotLinks(s, 0.8); len(hot) != 1 || hot[0] != "core>olt" {
		t.Fatalf("hot=%v", hot)
	}
	h := twin.CapacityHeadroom(s)
	if h[topology.NodeONU] != 1.0 {
		t.Fatalf("headroom=%v", h)
	}
}
