package jobs

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Production hardening: priority, per-device locks, per-connector rate
// limits, graceful shutdown. No goroutine fan-out here — one worker drains
// serially per queue; scale by running worker processes, not threads.

// Priority: higher value runs first. Default 0.
func (q *Queue) DrainCtx(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		q.mu.Lock()
		if len(q.pending) == 0 {
			q.mu.Unlock()
			return
		}
		sort.SliceStable(q.pending, func(i, j int) bool { return q.pending[i].Priority > q.pending[j].Priority })
		j := q.pending[0]
		q.pending = q.pending[1:]
		q.mu.Unlock()

		if key := j.RateKey; key != "" {
			limiters.mu.Lock()
			lm, ok := limiters.m[key]
			if !ok {
				lm = newBucket(5, 10) // default 5/s burst 10 per connector
				limiters.m[key] = lm
			}
			limiters.mu.Unlock()
			lm.wait(ctx)
		}
		if j.DeviceID != "" {
			devLocks.lock(j.DeviceID)
		}
		q.runOne(ctx, j)
		if j.DeviceID != "" {
			devLocks.unlock(j.DeviceID)
		}
	}
}

func (q *Queue) runOne(ctx context.Context, j *Job) {
	h, ok := q.handlers[j.Kind]
	if !ok {
		q.fail(j, errNoHandler(j.Kind))
		return
	}
	j.Status = "running"
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := h(ctx, j); err != nil {
		j.Attempts++
		j.LastError = err.Error()
		if j.Attempts >= j.MaxAttempts {
			j.Status = "dead"
			q.mu.Lock()
			q.dead = append(q.dead, j)
			q.mu.Unlock()
		} else {
			j.Status = "queued"
			q.mu.Lock()
			q.pending = append(q.pending, j)
			q.mu.Unlock()
			time.Sleep(time.Duration(j.Attempts*attemptBackoff(j.Attempts)) * time.Millisecond)
		}
		return
	}
	j.Status = "done"
}

func attemptBackoff(n int) int {
	ms := 100 << (n - 1)
	if ms > 8000 {
		ms = 8000
	}
	return ms
}

// --- per-device locks ---

type deviceLocks struct {
	mu sync.Mutex
	m  map[string]chan struct{}
}

var devLocks = deviceLocks{m: map[string]chan struct{}{}}

func (d *deviceLocks) lock(id string) {
	d.mu.Lock()
	ch, ok := d.m[id]
	if !ok {
		ch = make(chan struct{}, 1)
		ch <- struct{}{}
		d.m[id] = ch
	}
	d.mu.Unlock()
	<-ch
}

func (d *deviceLocks) unlock(id string) {
	d.mu.Lock()
	ch := d.m[id]
	d.mu.Unlock()
	ch <- struct{}{}
}

// --- token bucket rate limiters ---

type bucket struct {
	mu     sync.Mutex
	tokens float64
	rate   float64
	burst  float64
	last   time.Time
}

var limiters = struct {
	mu sync.Mutex
	m  map[string]*bucket
}{m: map[string]*bucket{}}

func newBucket(rate, burst float64) *bucket {
	return &bucket{tokens: burst, rate: rate, burst: burst, last: time.Now()}
}

func (b *bucket) wait(ctx context.Context) {
	for {
		b.mu.Lock()
		now := time.Now()
		b.tokens += now.Sub(b.last).Seconds() * b.rate
		if b.tokens > b.burst {
			b.tokens = b.burst
		}
		b.last = now
		if b.tokens >= 1 {
			b.tokens--
			b.mu.Unlock()
			return
		}
		b.mu.Unlock()
		select {
		case <-ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
	}
}
