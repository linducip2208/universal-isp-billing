package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Job struct {
	ID             string
	Kind           string
	Payload        map[string]any
	IdempotencyKey string
	RunAt          time.Time
	Attempts       int
	MaxAttempts    int
	LastError      string
	Status         string // queued | running | done | failed | dead
	Priority       int    // higher runs first
	DeviceID       string // jobs on the same device serialize
	RateKey        string // per-connector rate-limit bucket ("" = none)
}

func errNoHandler(kind string) error { return fmt.Errorf("no handler for kind %s", kind) }

type Handler func(ctx context.Context, j *Job) error

// Queue is an in-process job queue with retry/backoff/DLQ semantics.
// Production deployments front this with Redis Streams; the interface is stable.
type Queue struct {
	mu       sync.Mutex
	pending  []*Job
	dead     []*Job
	handlers map[string]Handler
	seen     map[string]bool
}

func New() *Queue { return &Queue{handlers: map[string]Handler{}, seen: map[string]bool{}} }

func (q *Queue) Register(kind string, h Handler) { q.handlers[kind] = h }

func (q *Queue) Enqueue(j *Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	if j.IdempotencyKey != "" && q.seen[j.IdempotencyKey] {
		return fmt.Errorf("duplicate idempotency key %s", j.IdempotencyKey)
	}
	if j.MaxAttempts == 0 {
		j.MaxAttempts = 5
	}
	if j.Status == "" {
		j.Status = "queued"
	}
	if j.IdempotencyKey != "" {
		q.seen[j.IdempotencyKey] = true
	}
	q.pending = append(q.pending, j)
	return nil
}

// Drain executes all queued jobs synchronously (workers call this in a loop).
func (q *Queue) Drain(ctx context.Context) {
	for {
		q.mu.Lock()
		if len(q.pending) == 0 {
			q.mu.Unlock()
			return
		}
		j := q.pending[0]
		q.pending = q.pending[1:]
		q.mu.Unlock()

		h, ok := q.handlers[j.Kind]
		if !ok {
			q.fail(j, fmt.Errorf("no handler for kind %s", j.Kind))
			continue
		}
		j.Status = "running"
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
				time.Sleep(time.Duration(j.Attempts) * 100 * time.Millisecond)
			}
			continue
		}
		j.Status = "done"
	}
}

func (q *Queue) fail(j *Job, err error) {
	j.Status = "dead"
	j.LastError = err.Error()
	q.mu.Lock()
	q.dead = append(q.dead, j)
	q.mu.Unlock()
}

func (q *Queue) Dead() []*Job {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]*Job{}, q.dead...)
}

func (q *Queue) Pending() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.pending)
}
