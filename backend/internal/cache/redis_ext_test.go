package cache_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/cache"
)

func liveRedis(t *testing.T) *cache.Redis {
	t.Helper()
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	return cache.NewRedis(addr)
}

func TestLiveLock(t *testing.T) {
	r := liveRedis(t)
	ctx := context.Background()
	l1, ok, err := r.AcquireLock(ctx, "isp:test:lock", 10*time.Second)
	if err != nil || !ok {
		t.Fatalf("acquire: ok=%v err=%v", ok, err)
	}
	if _, ok2, err := r.AcquireLock(ctx, "isp:test:lock", 10*time.Second); err != nil || ok2 {
		t.Fatalf("double acquire: ok=%v err=%v", ok2, err)
	}
	if err := l1.Refresh(ctx); err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if err := l1.Release(ctx); err != nil {
		t.Fatalf("release: %v", err)
	}
	l2, ok, err := r.AcquireLock(ctx, "isp:test:lock", 10*time.Second)
	if err != nil || !ok {
		t.Fatalf("re-acquire: ok=%v err=%v", ok, err)
	}
	_ = l2.Release(ctx)
}

func TestLiveStream(t *testing.T) {
	r := liveRedis(t)
	ctx := context.Background()
	s := cache.NewStream(r, "isp:test:stream", "g1", "c1")
	if err := s.EnsureGroup(ctx); err != nil {
		t.Fatalf("group: %v", err)
	}
	idem := fmt.Sprintf("idem-%d", time.Now().UnixNano())
	id, err := s.Publish(ctx, idem, map[string]string{"kind": "provision", "id": "j1"})
	if err != nil || id == "" {
		t.Fatalf("publish: id=%q err=%v", id, err)
	}
	if _, err := s.Publish(ctx, idem, map[string]string{"kind": "provision"}); err != cache.ErrDuplicate {
		t.Fatalf("dup publish must ErrDuplicate, got %v", err)
	}
	msgs, err := s.Read(ctx, 10, 500*time.Millisecond)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	found := false
	for _, m := range msgs {
		if m.Fields["id"] == "j1" {
			found = true
			if err := s.Ack(ctx, m.ID); err != nil {
				t.Fatalf("ack: %v", err)
			}
		}
	}
	if !found {
		t.Fatal("published message not read back")
	}
}
