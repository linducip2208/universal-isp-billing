// Package twin is the network digital twin: READ-ONLY simulation and
// analysis over inventory snapshots. It NEVER touches devices — it imports
// only the topology model (compile-time guarantee: no connectors import).
package twin

import (
	"github.com/universal-isp/platform/internal/topology"
)

// Snapshot is a point-in-time twin: graph + per-node state + link load.
type Snapshot struct {
	Graph  *topology.Graph
	Status map[string]string  // nodeID -> online|offline|degraded
	Load   map[string]float64 // edge "from>to" -> 0..1 utilization
}

// FailureResult is a what-if failure simulation outcome.
type FailureResult struct {
	FailedNode string                    `json:"failed_node"`
	Affected   []string                  `json:"affected_nodes"`
	ByKind     map[topology.NodeKind]int `json:"by_kind"`
}

// SimulateFailure answers "what breaks if node goes down" without touching
// anything real: graph dependents filtered to currently-online nodes.
func SimulateFailure(snap *Snapshot, nodeID string) FailureResult {
	affected, byKind := snap.Graph.ImpactOf(nodeID)
	var online []string
	filtered := map[topology.NodeKind]int{}
	for _, a := range affected {
		st := snap.Status[a]
		if st == "" || st == "online" {
			online = append(online, a)
			for _, n := range snap.Graph.Nodes {
				if n.ID == a {
					filtered[n.Kind]++
				}
			}
		}
	}
	_ = byKind
	return FailureResult{FailedNode: nodeID, Affected: online, ByKind: filtered}
}

// HotLinks returns edges above a utilization threshold (capacity watch).
func HotLinks(snap *Snapshot, threshold float64) []string {
	var out []string
	for e, load := range snap.Load {
		if load >= threshold {
			out = append(out, e)
		}
	}
	return out
}

// CapacityHeadroom reports per-kind online ratios (0..1) for planning.
func CapacityHeadroom(snap *Snapshot) map[topology.NodeKind]float64 {
	total := map[topology.NodeKind]int{}
	up := map[topology.NodeKind]int{}
	for _, n := range snap.Graph.Nodes {
		total[n.Kind]++
		if st := snap.Status[n.ID]; st == "" || st == "online" {
			up[n.Kind]++
		}
	}
	out := map[topology.NodeKind]float64{}
	for k, t := range total {
		out[k] = float64(up[k]) / float64(t)
	}
	return out
}
