package auth

import (
	"sync"
	"time"
)

// Revoker reports whether a token ID (jti) was revoked (logout).
type Revoker interface {
	Revoked(jti string) bool
	Revoke(jti string, ttl time.Duration)
}

// MemoryRevoker is the in-process denylist. Multi-instance deployments back
// this with Redis (SET NX PX on jti) behind the same interface.
type MemoryRevoker struct {
	mu sync.Mutex
	m  map[string]time.Time
}

func NewMemoryRevoker() *MemoryRevoker { return &MemoryRevoker{m: map[string]time.Time{}} }

func (r *MemoryRevoker) Revoked(jti string) bool {
	if jti == "" {
		return false
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	exp, ok := r.m[jti]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(r.m, jti)
		return false
	}
	return true
}

func (r *MemoryRevoker) Revoke(jti string, ttl time.Duration) {
	if jti == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.m) > 100000 {
		now := time.Now()
		for k, e := range r.m {
			if now.After(e) {
				delete(r.m, k)
			}
		}
	}
	r.m[jti] = time.Now().Add(ttl)
}
