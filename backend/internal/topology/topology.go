// Package topology: site/router/BNG/switch/AP/OLT/ONU/server/controller
// nodes with physical/logical/WAN/LAN/VLAN/fiber/wireless edges, generated
// from discovered inventory.
package topology

type NodeKind string

const (
	NodeSite       NodeKind = "site"
	NodeRouter     NodeKind = "router"
	NodeBNG        NodeKind = "bng"
	NodeSwitch     NodeKind = "switch"
	NodeAP         NodeKind = "ap"
	NodeOLT        NodeKind = "olt"
	NodeONU        NodeKind = "onu"
	NodeServer     NodeKind = "server"
	NodeController NodeKind = "controller"
)

type EdgeKind string

const (
	EdgePhysical EdgeKind = "physical"
	EdgeLogical  EdgeKind = "logical"
	EdgeWAN      EdgeKind = "wan"
	EdgeLAN      EdgeKind = "lan"
	EdgeVLAN     EdgeKind = "vlan"
	EdgeFiber    EdgeKind = "fiber"
	EdgeWireless EdgeKind = "wireless"
)

type Node struct {
	ID     string            `json:"id"`
	Kind   NodeKind          `json:"kind"`
	Label  string            `json:"label"`
	Status string            `json:"status,omitempty"`
	Meta   map[string]string `json:"meta,omitempty"`
}

type Edge struct {
	From string   `json:"from"`
	To   string   `json:"to"`
	Kind EdgeKind `json:"kind"`
}

type Graph struct {
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

func (g *Graph) AddNode(n Node) {
	for _, e := range g.Nodes {
		if e.ID == n.ID {
			return
		}
	}
	g.Nodes = append(g.Nodes, n)
}

func (g *Graph) Link(from, to string, kind EdgeKind) {
	g.Edges = append(g.Edges, Edge{From: from, To: to, Kind: kind})
}
