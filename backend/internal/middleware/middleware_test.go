package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/middleware"
	"github.com/universal-isp/platform/internal/rbac"
)

func TestPerOrgRateLimit(t *testing.T) {
	h := middleware.PerOrgRateLimit(2, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != 200 {
			t.Fatalf("req %d: %d", i, rec.Code)
		}
	}
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Fatalf("want 429, got %d", rec.Code)
	}
	// different org (via ctx) gets its own bucket
	req2 := httptest.NewRequest("GET", "/", nil).WithContext(rbac.WithOrg(httptest.NewRequest("GET", "/", nil).Context(), "other"))
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != 200 {
		t.Fatalf("other org must pass, got %d", rec2.Code)
	}
}
