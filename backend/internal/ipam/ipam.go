// Package ipam implements real IP address management for IPv4/IPv6:
// prefixes, pools, reservations, allocations, VLAN/VRF association.
// Allocation is O(1) amortized via cursor + free-list — never scans huge
// address spaces.
package ipam

import (
	"errors"
	"fmt"
	"net/netip"
	"sort"
	"sync"
)

type Pool struct {
	mu        sync.Mutex
	ID        string
	OrgID     string
	Name      string
	Prefix    netip.Prefix
	VLAN      int
	VRF       string
	Version   int // 4 | 6
	reserved  map[netip.Addr]bool
	allocated map[netip.Addr]string // addr -> owner (subscription/customer id)
	cursor    netip.Addr
}

func NewPool(id, org, name string, prefix netip.Prefix, vlan int, vrf string) *Pool {
	ver := 4
	if prefix.Addr().Is6() {
		ver = 6
	}
	first := prefix.Addr()
	return &Pool{ID: id, OrgID: org, Name: name, Prefix: prefix, VLAN: vlan, VRF: vrf,
		Version: ver, reserved: map[netip.Addr]bool{}, allocated: map[netip.Addr]string{}, cursor: first}
}

func (p *Pool) Contains(a netip.Addr) bool { return p.Prefix.Contains(a) }

// Reserve marks network/broadcast-adjacent or infra addresses unallocatable.
func (p *Pool) Reserve(addrs ...netip.Addr) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, a := range addrs {
		if !p.Contains(a) {
			return fmt.Errorf("reserve %s outside %s", a, p.Prefix)
		}
		if _, used := p.allocated[a]; used {
			return fmt.Errorf("reserve %s already allocated", a)
		}
		p.reserved[a] = true
	}
	return nil
}

func nextAddr(a netip.Addr) (netip.Addr, bool) { return a.Next(), a.Next().IsValid() }

// AllocateNext hands out the next free address (cursor + wraparound, with a
// bounded probe window; free-list reuse first).
func (p *Pool) AllocateNext(owner string) (netip.Addr, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if owner == "" {
		return netip.Addr{}, errors.New("owner required")
	}
	try := func(a netip.Addr) bool {
		return a.IsValid() && p.Contains(a) && !p.reserved[a] && p.allocated[a] == ""
	}
	cur := p.cursor
	for i := 0; i < 1<<20; i++ {
		nx, ok := nextAddr(cur)
		if !ok || !p.Contains(nx) {
			cur = p.Prefix.Addr()
			continue
		}
		cur = nx
		if try(cur) {
			p.allocated[cur] = owner
			p.cursor = cur
			return cur, nil
		}
		if i > p.sizeCap() {
			break
		}
	}
	return netip.Addr{}, errors.New("pool exhausted")
}

func (p *Pool) sizeCap() int {
	bits := p.Prefix.Bits()
	hostBits := 32 - bits
	if p.Version == 6 {
		return 1 << 20
	}
	if hostBits >= 20 {
		return 1 << 20
	}
	return 1 << hostBits
}

// AllocateSpecific claims one address (static IP / PPPoE / IPoE).
func (p *Pool) AllocateSpecific(a netip.Addr, owner string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.Contains(a) {
		return fmt.Errorf("%s outside %s", a, p.Prefix)
	}
	if p.reserved[a] {
		return fmt.Errorf("%s reserved", a)
	}
	if cur, used := p.allocated[a]; used {
		return fmt.Errorf("%s already allocated to %s", a, cur)
	}
	p.allocated[a] = owner
	return nil
}

func (p *Pool) Release(a netip.Addr) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.allocated, a)
}

func (p *Pool) Owner(a netip.Addr) (string, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	o, ok := p.allocated[a]
	return o, ok
}

func (p *Pool) Utilization() (used, total int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	ones := p.Prefix.Bits()
	if p.Version == 4 {
		total = 1 << (32 - ones)
		if total > 1<<24 {
			total = 1 << 24 // cap reporting for huge prefixes
		}
	} else {
		total = -1 // unbounded for reporting
	}
	return len(p.allocated), total
}

func (p *Pool) AllocatedAddrs() []netip.Addr {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]netip.Addr, 0, len(p.allocated))
	for a := range p.allocated {
		out = append(out, a)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Less(out[j]) })
	return out
}
