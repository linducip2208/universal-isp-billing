// Package metrics: minimal Prometheus-compatible in-process metrics
// (counters + latency sum/count + gauges). No external deps. Instances are
// explicit (no global registry) — wire one from main into middleware and /metrics.
package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type labels map[string]string

func key(l labels) string {
	ks := make([]string, 0, len(l))
	for k := range l {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	var sb strings.Builder
	for _, k := range ks {
		sb.WriteString(k + "=" + l[k] + ",")
	}
	return sb.String()
}

type Registry struct {
	mu       sync.Mutex
	counters map[string]map[string]float64
	sums     map[string]map[string]float64
	counts   map[string]map[string]float64
	gauges   map[string]map[string]float64
	help     map[string]string
}

func New() *Registry {
	return &Registry{counters: map[string]map[string]float64{}, sums: map[string]map[string]float64{},
		counts: map[string]map[string]float64{}, gauges: map[string]map[string]float64{}, help: map[string]string{}}
}

func (r *Registry) Inc(name string, l labels) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.counters[name]
	if !ok {
		m = map[string]float64{}
		r.counters[name] = m
	}
	m[key(l)]++
}

func (r *Registry) Observe(name string, l labels, v float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := key(l)
	sm, ok := r.sums[name]
	if !ok {
		sm = map[string]float64{}
		r.sums[name] = sm
	}
	sm[k] += v
	cm, ok := r.counts[name]
	if !ok {
		cm = map[string]float64{}
		r.counts[name] = cm
	}
	cm[k]++
}

func (r *Registry) Set(name string, l labels, v float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	m, ok := r.gauges[name]
	if !ok {
		m = map[string]float64{}
		r.gauges[name] = m
	}
	m[key(l)] = v
}

func (r *Registry) Help(name, text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.help[name] = text
}

// Exposition renders Prometheus text format 0.0.4.
func (r *Registry) Exposition() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	var sb strings.Builder
	emit := func(name, typ string, m map[string]float64) {
		if len(m) == 0 {
			return
		}
		if h, ok := r.help[name]; ok {
			fmt.Fprintf(&sb, "# HELP %s %s\n", name, h)
		}
		fmt.Fprintf(&sb, "# TYPE %s %s\n", name, typ)
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&sb, "%s{%s} %v\n", name, strings.TrimSuffix(k, ","), m[k])
		}
	}
	names := map[string]bool{}
	for n := range r.counters {
		names[n] = true
	}
	for n := range r.sums {
		names[n] = true
	}
	for n := range r.gauges {
		names[n] = true
	}
	ordered := make([]string, 0, len(names))
	for n := range names {
		ordered = append(ordered, n)
	}
	sort.Strings(ordered)
	for _, n := range ordered {
		if m, ok := r.counters[n]; ok {
			emit(n, "counter", m)
		}
		if s, ok := r.sums[n]; ok {
			emit(n+"_sum", "counter", s)
			emit(n+"_count", "counter", r.counts[n])
		}
		if g, ok := r.gauges[n]; ok {
			emit(n, "gauge", g)
		}
	}
	return sb.String()
}
