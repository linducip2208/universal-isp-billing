package cache_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/cache"
)

func TestMemoryTTL(t *testing.T) {
	m := cache.NewMemory()
	ctx := context.Background()
	if err := m.Set(ctx, "k", "v", 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}
	if v, ok := m.Get(ctx, "k"); !ok || v != "v" {
		t.Fatal("want hit")
	}
	time.Sleep(80 * time.Millisecond)
	if _, ok := m.Get(ctx, "k"); ok {
		t.Fatal("want expiry")
	}
}

// TestLiveRedis runs against a real Redis/compatible server when
// TEST_REDIS_ADDR is set (e.g. 127.0.0.1:6379). Skipped otherwise.
func TestLiveRedis(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	r := cache.NewRedis(addr)
	ctx := context.Background()
	if err := r.Set(ctx, "isp:test", "v", 0); err != nil {
		t.Fatalf("SET: %v", err)
	}
	v, ok := r.Get(ctx, "isp:test")
	if !ok || v != "v" {
		t.Fatalf("GET=%q ok=%v", v, ok)
	}
	if err := r.Del(ctx, "isp:test"); err != nil {
		t.Fatalf("DEL: %v", err)
	}
	if _, ok := r.Get(ctx, "isp:test"); ok {
		t.Fatal("want miss after DEL")
	}
}
