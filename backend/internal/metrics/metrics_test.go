package metrics_test

import (
	"strings"
	"testing"

	"github.com/universal-isp/platform/internal/metrics"
)

func TestExposition(t *testing.T) {
	r := metrics.New()
	r.Help("http_requests", "API requests")
	r.Inc("http_requests", map[string]string{"method": "GET", "status": "200"})
	r.Inc("http_requests", map[string]string{"method": "GET", "status": "200"})
	r.Observe("http_latency_ms", map[string]string{"route": "/x"}, 12)
	r.Set("queue_depth", map[string]string{"q": "provision"}, 3)
	out := r.Exposition()
	if !strings.Contains(out, `http_requests{method=GET,status=200} 2`) {
		t.Fatalf("counters:\n%s", out)
	}
	if !strings.Contains(out, "http_latency_ms_sum") || !strings.Contains(out, "http_latency_ms_count") {
		t.Fatalf("latency:\n%s", out)
	}
	if !strings.Contains(out, "queue_depth") {
		t.Fatalf("gauge:\n%s", out)
	}
}
