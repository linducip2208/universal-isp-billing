package timezone_test

import (
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/timezone"
)

func TestBusinessJakarta(t *testing.T) {
	utc := time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
	got := timezone.Business(utc, "Asia/Jakarta")
	if got != "2026-10-02 07:00:00 WIB" {
		t.Fatalf("got %q", got)
	}
	if got := timezone.Business(utc, "America/New_York"); got == "" {
		t.Fatal("empty")
	}
	loc, ok := timezone.Load("Mars/Olympus")
	if ok {
		t.Fatal("invalid zone must report fallback")
	}
	if loc.String() != "Asia/Jakarta" {
		t.Fatalf("fallback=%s", loc)
	}
}
