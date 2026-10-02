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

func newServer() *httpapi.Server {
	return httpapi.New(slog.New(slog.NewTextHandler(os.Stderr, nil)), "test-secret-12345678", &health.Checker{RedisAddr: "127.0.0.1:1"})
}

func TestDemoLoginGatedOffByDefault(t *testing.T) {
	s := newServer()
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 401 {
		t.Fatalf("demo login must be refused by default, got %d", rec.Code)
	}
}

func TestDemoLoginWhenEnabled(t *testing.T) {
	s := newServer().WithDemoLogin(true)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "token") {
		t.Fatalf("enabled demo login failed: %d %s", rec.Code, rec.Body.String())
	}
	// use the token against a protected route without DB -> 503 (honest, not fake)
	tok := strings.Split(strings.Split(rec.Body.String(), `"token":"`)[1], `"`)[0]
	req2 := httptest.NewRequest("GET", "/api/v1/customers", nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	rec2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503 without DB, got %d %s", rec2.Code, rec2.Body.String())
	}
	// unauthenticated -> 401
	req3 := httptest.NewRequest("GET", "/api/v1/customers", nil)
	rec3 := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusUnauthorized {
		t.Fatalf("want 401 got %d", rec3.Code)
	}
}

func TestLoginThrottle(t *testing.T) {
	s := newServer().WithDemoLogin(true)
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"wrong"}`))
		req.RemoteAddr = "9.9.9.9:1234"
		rec := httptest.NewRecorder()
		s.Handler().ServeHTTP(rec, req)
		if i >= 5 && rec.Code != http.StatusTooManyRequests {
			t.Fatalf("attempt %d: want 429 got %d", i, rec.Code)
		}
	}
}

func TestRequestIDPropagated(t *testing.T) {
	s := newServer()
	req := httptest.NewRequest("GET", "/live", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("missing X-Request-ID")
	}
}

func TestLogoutRevokes(t *testing.T) {
	s := newServer().WithDemoLogin(true)
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"secret"}`))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("login: %d", rec.Code)
	}
	tok := strings.Split(strings.Split(rec.Body.String(), `"token":"`)[1], `"`)[0]
	// logout
	reqL := httptest.NewRequest("POST", "/api/v1/auth/logout", nil)
	reqL.Header.Set("Authorization", "Bearer "+tok)
	recL := httptest.NewRecorder()
	s.Handler().ServeHTTP(recL, reqL)
	if recL.Code != 200 {
		t.Fatalf("logout: %d %s", recL.Code, recL.Body.String())
	}
	// token must now be rejected
	req2 := httptest.NewRequest("GET", "/api/v1/customers", nil)
	req2.Header.Set("Authorization", "Bearer "+tok)
	rec2 := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("revoked token must 401, got %d", rec2.Code)
	}
}
