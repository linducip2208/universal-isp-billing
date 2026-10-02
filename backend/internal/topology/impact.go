package topology

// Impact analysis answers: "who is affected if this node goes down?"
// Pure graph traversal over the inventory-built Graph — no device I/O.

func (g *Graph) downstream(from string) []string {
	adj := map[string][]string{}
	for _, e := range g.Edges {
		adj[e.From] = append(adj[e.From], e.To)
	}
	var out []string
	seen := map[string]bool{from: true}
	queue := []string{from}
	for len(queue) > 0 {
		n := queue[0]
		queue = queue[1:]
		for _, m := range adj[n] {
			if !seen[m] {
				seen[m] = true
				out = append(out, m)
				queue = append(queue, m)
			}
		}
	}
	return out
}

// ImpactOf returns all nodes downstream of id (dependents) plus the node
// kind breakdown for NOC display.
func (g *Graph) ImpactOf(id string) (affected []string, byKind map[NodeKind]int) {
	affected = g.downstream(id)
	byKind = map[NodeKind]int{}
	kindOf := map[string]NodeKind{}
	for _, n := range g.Nodes {
		kindOf[n.ID] = n.Kind
	}
	for _, a := range affected {
		byKind[kindOf[a]]++
	}
	return affected, byKind
}

// PathTo traces one upstream chain from id toward roots (for root-cause hints).
func (g *Graph) PathTo(id string) []string {
	parent := map[string]string{}
	for _, e := range g.Edges {
		if _, ok := parent[e.To]; !ok {
			parent[e.To] = e.From
		}
	}
	var path []string
	for cur := id; cur != ""; {
		path = append([]string{cur}, path...)
		cur = parent[cur]
		if len(path) > 100 {
			break
		}
	}
	return path
}
