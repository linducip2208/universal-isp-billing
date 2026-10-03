package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/universal-isp/platform/internal/middleware"
	"github.com/universal-isp/platform/internal/rbac"
)

type fakeKeys struct{}

func (fakeKeys) VerifyAPIKey(_ context.Context, raw string) (string, string, []string, error) {
	if raw == "good" {
		return "k1", "org-9", []string{"billing:read"}, nil
	}
	return "", "", nil, errors.New("bad")
}

func TestAPIKey(t *testing.T) {
	var gotOrg string
	var can bool
	h := middleware.APIKey(fakeKeys{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotOrg = rbac.OrgOf(r.Context())
		can = rbac.Can(r.Context(), rbac.BillingRead)
		w.WriteHeader(200)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-API-Key", "good")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if gotOrg != "org-9" || !can {
		t.Fatalf("org=%q can=%v", gotOrg, can)
	}
	// billing:write must NOT be granted by billing:read scope
	req2 := httptest.NewRequest("GET", "/", nil)
	req2.Header.Set("X-API-Key", "good")
	rec2 := httptest.NewRecorder()
	gated := middleware.APIKey(fakeKeys{})(middleware.Require(rbac.BillingWrite)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})))
	gated.ServeHTTP(rec2, req2)
	if rec2.Code != 403 {
		t.Fatalf("scope must not escalate: %d", rec2.Code)
	}
	// bad key passes through anonymously
	req3 := httptest.NewRequest("GET", "/", nil)
	req3.Header.Set("X-API-Key", "bad")
	rec3 := httptest.NewRecorder()
	h.ServeHTTP(rec3, req3)
	if rec3.Code != 200 {
		t.Fatalf("bad key must pass through, got %d", rec3.Code)
	}
}
