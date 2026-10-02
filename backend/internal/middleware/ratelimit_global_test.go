package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/middleware"
)

type fakeRedis struct {
	mu sync.Mutex
	m  map[string]int64
}

func (f *fakeRedis) Do(_ context.Context, args ...string) (any, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch args[0] {
	case "INCR":
		f.m[args[1]]++
		return f.m[args[1]], nil
	case "EXPIRE":
		return int64(1), nil
	}
	return nil, errors.New("unknown")
}

func TestGlobalLimit(t *testing.T) {
	fr := &fakeRedis{m: map[string]int64{}}
	h := middleware.GlobalLimit(fr.Do, 2, time.Minute, "t:")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	for i := 0; i < 2; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
		if rec.Code != 200 {
			t.Fatalf("req %d: %d", i, rec.Code)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 429 {
		t.Fatalf("want 429, got %d", rec.Code)
	}
}

func TestGlobalLimitFailOpen(t *testing.T) {
	broken := func(context.Context, ...string) (any, error) { return nil, errors.New("down") }
	h := middleware.GlobalLimit(broken, 1, time.Minute, "t:")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != 200 {
		t.Fatalf("backend down must fail open, got %d", rec.Code)
	}
	var nilDo middleware.DoFunc
	h2 := middleware.GlobalLimit(nilDo, 1, time.Minute, "t:")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	rec2 := httptest.NewRecorder()
	h2.ServeHTTP(rec2, httptest.NewRequest("GET", "/", nil))
	if rec2.Code != 200 {
		t.Fatalf("nil backend must pass, got %d", rec2.Code)
	}
}
