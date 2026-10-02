package cache_test

import (
	"context"
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
