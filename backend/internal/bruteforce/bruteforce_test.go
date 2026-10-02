package bruteforce_test

import (
	"os"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/auth"
	"github.com/universal-isp/platform/internal/bruteforce"
	"github.com/universal-isp/platform/internal/cache"
)

func TestMemoryBlocks(t *testing.T) {
	m := bruteforce.NewMemory(3, time.Minute)
	for i := 0; i < 3; i++ {
		if _, blocked := m.Fail("k"); blocked {
			t.Fatalf("attempt %d must pass", i)
		}
	}
	if _, blocked := m.Fail("k"); !blocked {
		t.Fatal("4th attempt must block")
	}
	m.Reset("k")
	if _, blocked := m.Fail("k"); blocked {
		t.Fatal("reset must clear")
	}
}

func TestLiveRedisBruteforce(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	r := cache.NewRedis(addr)
	bf := &bruteforce.Redis{Do: r.Do, Max: 2, Window: time.Minute, Prefix: "isp:test:bf:"}
	key := "u-" + time.Now().Format("150405.000000000")
	for i := 0; i < 2; i++ {
		if _, blocked := bf.Fail(key); blocked {
			t.Fatal("must pass")
		}
	}
	if _, blocked := bf.Fail(key); !blocked {
		t.Fatal("must block")
	}
	bf.Reset(key)
	if _, blocked := bf.Fail(key); blocked {
		t.Fatal("reset must clear")
	}
}

func TestLiveRedisRevoker(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set")
	}
	r := cache.NewRedis(addr)
	rv := &auth.RedisRevoker{Do: r.Do}
	if rv.Revoked("nope") {
		t.Fatal("fresh must pass")
	}
	rv.Revoke("jti-1", time.Minute)
	if !rv.Revoked("jti-1") {
		t.Fatal("revoked must fail closed")
	}
}
