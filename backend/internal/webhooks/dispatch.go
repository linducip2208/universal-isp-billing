package webhooks

import (
	"context"
	"sync"
	"time"
)

// Dispatcher matches events to endpoint subscriptions and delivers with
// per-endpoint timeout, recording history for replay and dead-letter review.
type Dispatcher struct {
	mu        sync.Mutex
	endpoints []Endpoint
	history   []Delivery
	maxHist   int
}

type Delivery struct {
	At        time.Time `json:"at"`
	Endpoint  string    `json:"endpoint"`
	Event     string    `json:"event"`
	Status    int       `json:"status"`
	Error     string    `json:"error,omitempty"`
	LatencyMs int64     `json:"latency_ms"`
}

func NewDispatcher(eps []Endpoint) *Dispatcher {
	return &Dispatcher{endpoints: eps, maxHist: 500}
}

// matches supports exact and prefix subscriptions ("invoice.*").
func matches(sub, event string) bool {
	if sub == event {
		return true
	}
	if len(sub) > 2 && sub[len(sub)-2:] == ".*" {
		prefix := sub[:len(sub)-1]
		return len(event) >= len(prefix) && event[:len(prefix)] == prefix
	}
	return false
}

// Dispatch delivers body to all matching endpoints, recording each attempt.
func (d *Dispatcher) Dispatch(ctx context.Context, event string, body []byte) []Delivery {
	d.mu.Lock()
	eps := append([]Endpoint{}, d.endpoints...)
	d.mu.Unlock()
	var out []Delivery
	for _, ep := range eps {
		hit := false
		for _, e := range ep.Events {
			if matches(e, event) {
				hit = true
				break
			}
		}
		if !hit {
			continue
		}
		start := time.Now()
		status, err := Deliver(ctx, ep, event, body)
		dl := Delivery{At: start, Endpoint: ep.URL, Event: event, Status: status, LatencyMs: time.Since(start).Milliseconds()}
		if err != nil {
			dl.Error = err.Error()
		}
		out = append(out, dl)
		d.record(dl)
	}
	return out
}

func (d *Dispatcher) record(dl Delivery) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.history = append(d.history, dl)
	if len(d.history) > d.maxHist {
		d.history = d.history[len(d.history)-d.maxHist:]
	}
}

// History returns recent deliveries (newest last).
func (d *Dispatcher) History() []Delivery {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]Delivery{}, d.history...)
}

// Replay redelivers one recorded delivery by endpoint+event with a new body.
func (d *Dispatcher) Replay(ctx context.Context, endpoint, event string, body []byte) (Delivery, bool) {
	d.mu.Lock()
	var ep *Endpoint
	for i := range d.endpoints {
		if d.endpoints[i].URL == endpoint {
			ep = &d.endpoints[i]
			break
		}
	}
	d.mu.Unlock()
	if ep == nil {
		return Delivery{}, false
	}
	start := time.Now()
	status, err := Deliver(ctx, *ep, event, body)
	dl := Delivery{At: start, Endpoint: endpoint, Event: event, Status: status, LatencyMs: time.Since(start).Milliseconds()}
	if err != nil {
		dl.Error = err.Error()
	}
	d.record(dl)
	return dl, true
}
