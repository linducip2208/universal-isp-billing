package httpapi_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/universal-isp/platform/internal/health"
	"github.com/universal-isp/platform/internal/httpapi"
)

func TestLoginAndRBAC(t *testing.T) {
	s := httpapi.New(slog.New(slog.NewTextHandler(os.Stderr, nil)), "test-secret-12345678", &health.Checker{RedisAddr: "127.0.0.1:1"})
	// login
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "token") {
		t.Fatalf("login failed: %d %s", rec.Code, rec.Body.String())
	}
	// unauthenticated customers -> 401
	req2 := httptest.NewRequest("GET", "/api/v1/customers", nil)
	rec2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rec2.Code)
	}
}
