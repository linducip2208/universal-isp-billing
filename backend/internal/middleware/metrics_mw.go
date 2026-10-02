package middleware

import (
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/universal-isp/platform/internal/metrics"
)

var uuidSeg = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)
var numSeg = regexp.MustCompile(`(^|/)[0-9]+($|/)`)

// normalizeRoute bounds label cardinality: UUIDs/numbers become {id}.
func normalizeRoute(p string) string {
	p = uuidSeg.ReplaceAllString(p, "{id}")
	return numSeg.ReplaceAllString(p, "${1}{id}${2}")
}

// Metrics records per-route request counts + latency into reg (nil-safe).
func Metrics(reg *metrics.Registry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if reg == nil {
				next.ServeHTTP(w, r)
				return
			}
			start := time.Now()
			rec := &statusRec{ResponseWriter: w, code: 200}
			next.ServeHTTP(rec, r)
			route := normalizeRoute(r.URL.Path)
			l := map[string]string{"method": r.Method, "route": route, "status": strconv.Itoa(rec.code)}
			reg.Inc("http_requests", l)
			reg.Observe("http_latency_ms", map[string]string{"method": r.Method, "route": route}, float64(time.Since(start).Milliseconds()))
		})
	}
}

type statusRec struct {
	http.ResponseWriter
	code int
}

func (s *statusRec) WriteHeader(c int) {
	s.code = c
	s.ResponseWriter.WriteHeader(c)
}
