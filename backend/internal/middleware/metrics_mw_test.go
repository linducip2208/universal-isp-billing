package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/universal-isp/platform/internal/metrics"
	"github.com/universal-isp/platform/internal/middleware"
)

func TestMetricsCounts(t *testing.T) {
	reg := metrics.New()
	h := middleware.Metrics(reg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
	}))
	req := httptest.NewRequest("GET", "/api/v1/devices/11111111-1111-1111-1111-111111111111", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	out := reg.Exposition()
	if !strings.Contains(out, `route=/api/v1/devices/{id}`) {
		t.Fatalf("route not normalized:\n%s", out)
	}
	if !strings.Contains(out, `status=201`) {
		t.Fatalf("status missing:\n%s", out)
	}
	// nil registry passes through
	h2 := middleware.Metrics(nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	rec2 := httptest.NewRecorder()
	h2.ServeHTTP(rec2, httptest.NewRequest("GET", "/", nil))
	if rec2.Code != 200 {
		t.Fatal("nil registry must pass through")
	}
}
