package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/universal-isp/platform/internal/auth"
	"github.com/universal-isp/platform/internal/rbac"
)

func Chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func Logging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			// Never log tokens, passwords, or secrets.
			log.Info("http", "method", r.Method, "path", r.URL.Path,
				"dur_ms", time.Since(start).Milliseconds(),
				"req_id", r.Header.Get("X-Request-ID"))
		})
	}
}

// RequestID attaches/propagates X-Request-ID for audit correlation.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			var b [8]byte
			_, _ = rand.Read(b[:])
			id = hex.EncodeToString(b[:])
		}
		w.Header().Set("X-Request-ID", id)
		r.Header.Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

// Error writes the consistent API error envelope.
func Error(w http.ResponseWriter, code int, msg, reqID string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":"` + msg + `","request_id":"` + reqID + `"}`))
}

// PerOrgRateLimit enforces per-organization (fallback: per-IP) quotas so one
// tenant or key cannot starve others. Separate from the global RateLimit.
func PerOrgRateLimit(n int, win time.Duration) func(http.Handler) http.Handler {
	type key struct{ org, ip string }
	var mu sync.Mutex
	hits := map[key][]time.Time{}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			k := key{org: rbac.OrgOf(r.Context()), ip: r.RemoteAddr}
			now := time.Now()
			mu.Lock()
			var keep []time.Time
			for _, t := range hits[k] {
				if now.Sub(t) < win {
					keep = append(keep, t)
				}
			}
			if len(keep) >= n {
				mu.Unlock()
				Error(w, http.StatusTooManyRequests, "organization rate limited", r.Header.Get("X-Request-ID"))
				return
			}
			hits[k] = append(keep, now)
			mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}

// CORS enforces an explicit origin allowlist. Empty allowlist = same-origin
// only (no ACAO headers emitted).
func CORS(allowed []string) func(http.Handler) http.Handler {
	set := map[string]bool{}
	for _, o := range allowed {
		set[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" && set[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func JWT(secret string, revoker auth.Revoker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/api/v1/auth/login" || r.URL.Path == "/api/v1/auth/mfa/verify" || strings.HasPrefix(r.URL.Path, "/api/v1/payments/webhook/") || r.URL.Path == "/health" || r.URL.Path == "/ready" || r.URL.Path == "/live" || r.URL.Path == "/metrics" {
				next.ServeHTTP(w, r)
				return
			}
			h := r.Header.Get("Authorization")
			if !strings.HasPrefix(h, "Bearer ") {
				http.Error(w, `{"error":"missing bearer token"}`, http.StatusUnauthorized)
				return
			}
			cl, err := auth.Verify(secret, strings.TrimPrefix(h, "Bearer "))
			if err != nil {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			if revoker != nil && revoker.Revoked(cl.ID) {
				http.Error(w, `{"error":"token revoked"}`, http.StatusUnauthorized)
				return
			}
			ctx := rbac.WithRoles(r.Context(), cl.Roles)
			ctx = rbac.WithOrg(ctx, cl.Organization)
			ctx = auth.WithClaims(ctx, cl)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Require(p rbac.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rbac.Can(r.Context(), p) {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type limiter struct {
	mu   sync.Mutex
	hits map[string][]time.Time
	n    int
	win  time.Duration
}

func RateLimit(n int, win time.Duration) func(http.Handler) http.Handler {
	l := &limiter{hits: map[string][]time.Time{}, n: n, win: win}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			now := time.Now()
			l.mu.Lock()
			hs := l.hits[ip]
			var keep []time.Time
			for _, t := range hs {
				if now.Sub(t) < win {
					keep = append(keep, t)
				}
			}
			if len(keep) >= l.n {
				l.mu.Unlock()
				http.Error(w, `{"error":"rate limited"}`, http.StatusTooManyRequests)
				return
			}
			l.hits[ip] = append(keep, now)
			l.mu.Unlock()
			next.ServeHTTP(w, r)
		})
	}
}
