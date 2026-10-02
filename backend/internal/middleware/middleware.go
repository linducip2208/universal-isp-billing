package middleware

import (
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
			log.Info("http", "method", r.Method, "path", r.URL.Path, "dur_ms", time.Since(start).Milliseconds(), "ip", r.RemoteAddr)
		})
	}
}

func JWT(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/v1/auth/") || r.URL.Path == "/health" || r.URL.Path == "/ready" || r.URL.Path == "/live" || r.URL.Path == "/metrics" {
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
			ctx := rbac.WithRoles(r.Context(), cl.Roles)
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
