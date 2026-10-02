package jobs_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/jobs"
)

func TestPriorityOrder(t *testing.T) {
	q := jobs.New()
	var mu sync.Mutex
	var order []string
	q.Register("rec", func(_ context.Context, j *jobs.Job) error {
		mu.Lock()
		order = append(order, j.ID)
		mu.Unlock()
		return nil
	})
	_ = q.Enqueue(&jobs.Job{ID: "low", Kind: "rec", Priority: 1})
	_ = q.Enqueue(&jobs.Job{ID: "high", Kind: "rec", Priority: 9})
	q.DrainCtx(context.Background())
	if len(order) != 2 || order[0] != "high" {
		t.Fatalf("order=%v", order)
	}
}

func TestDeviceSerialization(t *testing.T) {
	q := jobs.New()
	var mu sync.Mutex
	inflight := map[string]bool{}
	bad := false
	q.Register("dev", func(_ context.Context, j *jobs.Job) error {
		mu.Lock()
		if inflight[j.DeviceID] {
			bad = true
		}
		inflight[j.DeviceID] = true
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		mu.Lock()
		inflight[j.DeviceID] = false
		mu.Unlock()
		return nil
	})
	_ = q.Enqueue(&jobs.Job{ID: "a", Kind: "dev", DeviceID: "d1"})
	_ = q.Enqueue(&jobs.Job{ID: "b", Kind: "dev", DeviceID: "d1"})
	_ = q.Enqueue(&jobs.Job{ID: "c", Kind: "dev", DeviceID: "d2"})
	q.DrainCtx(context.Background())
	if bad {
		t.Fatal("concurrent jobs on same device")
	}
	if q.Pending() != 0 {
		t.Fatal("queue should drain")
	}
}
