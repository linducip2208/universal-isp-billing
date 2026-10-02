// Package bruteforce: distributed login-attempt counting (memory + Redis)
// for failed-login protection and brute-force defense.
package bruteforce

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Tracker interface {
	// Fail records a failed attempt; returns attempts in window and whether blocked.
	Fail(key string) (attempts int, blocked bool)
	Reset(key string)
}

type Memory struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

func NewMemory(max int, window time.Duration) *Memory {
	return &Memory{hits: map[string][]time.Time{}, max: max, window: window}
}

func (m *Memory) Fail(key string) (int, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	var keep []time.Time
	for _, t := range m.hits[key] {
		if now.Sub(t) < m.window {
			keep = append(keep, t)
		}
	}
	keep = append(keep, now)
	m.hits[key] = keep
	return len(keep), len(keep) > m.max
}

func (m *Memory) Reset(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.hits, key)
}

// Redis tracker: INCR + EXPIRE window per key (works across instances).
type Redis struct {
	Do     func(ctx context.Context, args ...string) (any, error)
	Max    int
	Window time.Duration
	Prefix string
}

func (r *Redis) key(k string) string { return r.Prefix + k }

func (r *Redis) Fail(key string) (int, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	v, err := r.Do(ctx, "INCR", r.key(key))
	if err != nil {
		return 0, false // Redis down: do not block logins; monitor Redis separately
	}
	n, _ := v.(int64)
	if n == 1 {
		_, _ = r.Do(ctx, "EXPIRE", r.key(key), fmt.Sprint(int(r.Window.Seconds())))
	}
	return int(n), int(n) > r.Max
}

func (r *Redis) Reset(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, _ = r.Do(ctx, "DEL", r.key(key))
}
