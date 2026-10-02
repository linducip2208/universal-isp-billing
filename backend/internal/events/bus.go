package events

import (
	"context"
	"sync"
	"time"
)

// Event is the internal domain event envelope.
type Event struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	OccurredAt   time.Time      `json:"occurred_at"`
	Organization string         `json:"organization,omitempty"`
	Actor        string         `json:"actor,omitempty"`
	Resource     string         `json:"resource,omitempty"`
	ResourceID   string         `json:"resource_id,omitempty"`
	Payload      map[string]any `json:"payload,omitempty"`
}

type Handler func(ctx context.Context, e Event)

// Bus is a lightweight in-process event bus (NATS-ready seam:
// replace with a durable transport without touching publishers).
type Bus struct {
	mu   sync.RWMutex
	subs map[string][]Handler
}

func New() *Bus { return &Bus{subs: map[string][]Handler{}} }

func (b *Bus) Subscribe(eventType string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[eventType] = append(b.subs[eventType], h)
}

func (b *Bus) Publish(ctx context.Context, e Event) {
	if e.OccurredAt.IsZero() {
		e.OccurredAt = time.Now().UTC()
	}
	b.mu.RLock()
	hs := append([]Handler{}, b.subs[e.Type]...)
	b.mu.RUnlock()
	for _, h := range hs {
		h(ctx, e)
	}
}
