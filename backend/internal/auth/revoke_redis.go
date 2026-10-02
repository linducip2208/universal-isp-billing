package auth

import (
	"context"
	"fmt"
	"time"
)

// RedisRevoker backs the jti denylist with Redis (SET NX PX): works across
// API instances. Falls back to NO-OP deny-open? No — fail-closed is wrong
// for logout UX; instead callers keep MemoryRevoker as fallback and use
// Redis when reachable. Interface matches Revoker.
type RedisRevoker struct {
	Do func(ctx context.Context, args ...string) (any, error)
}

func (r *RedisRevoker) key(jti string) string { return "isp:revoked:" + jti }

func (r *RedisRevoker) Revoked(jti string) bool {
	if jti == "" || r == nil || r.Do == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	v, err := r.Do(ctx, "GET", r.key(jti))
	if err != nil {
		return false // Redis down: do not lock users out; incident logged by caller
	}
	s, _ := v.(string)
	return s != ""
}

func (r *RedisRevoker) Revoke(jti string, ttl time.Duration) {
	if jti == "" || r == nil || r.Do == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	ms := int(ttl.Milliseconds())
	if ms < 1000 {
		ms = 1000
	}
	_, _ = r.Do(ctx, "SET", r.key(jti), "1", "PX", fmt.Sprint(ms))
}
