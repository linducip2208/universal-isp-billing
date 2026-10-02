package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DoFunc abstracts a Redis-like INCR/EXPIRE backend for tests and wiring.
type DoFunc func(ctx context.Context, args ...string) (any, error)

// GlobalLimit enforces a sliding-window quota per key (default: client IP)
// in shared state (Redis) so it holds across API instances. do==nil or any
// backend error fails OPEN (availability over strictness; Redis health is
// monitored separately) — documented, deliberate.
func GlobalLimit(do DoFunc, max int, win time.Duration, keyPrefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if do == nil {
				next.ServeHTTP(w, r)
				return
			}
			ip, _, _ := strings.Cut(r.RemoteAddr, ":")
			key := keyPrefix + ip
			ctx, cancel := context.WithTimeout(r.Context(), 300*time.Millisecond)
			defer cancel()
			v, err := do(ctx, "INCR", key)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			n, _ := v.(int64)
			if n == 1 {
				_, _ = do(ctx, "EXPIRE", key, strconv.Itoa(int(win.Seconds())))
			}
			if int(n) > max {
				Error(w, http.StatusTooManyRequests, "global rate limited", r.Header.Get("X-Request-ID"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

var _ = fmt.Sprint
