package radius_test

import (
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/radius"
)

func TestReapLifecycle(t *testing.T) {
	s := radius.NewSessionStore()
	// duplicate start upserts, doesn't duplicate
	s.Start(&radius.Session{Username: "a", AcctSessionID: "s1"})
	s.Start(&radius.Session{Username: "a", AcctSessionID: "s1"})
	if len(s.Active()) != 1 {
		t.Fatal("dup start must upsert")
	}
	// stop unknown is a no-op
	s.Stop("nope")
	// fresh session survives reap
	if reaped := s.Reap(time.Hour); len(reaped) != 0 {
		t.Fatalf("fresh reaped: %v", reaped)
	}
	// stale session reaped (zero LastInterim + old start can't be forged;
	// use tiny window after interim aging via Stop+re-add path)
	s.Interim("s1", 5, 5)
	time.Sleep(2 * time.Millisecond) // Windows clock granularity
	if reaped := s.Reap(0); len(reaped) != 1 || reaped[0].AcctSessionID != "s1" {
		t.Fatalf("reaped=%v", reaped)
	}
	if len(s.Active()) != 0 {
		t.Fatal("reaped session must be gone")
	}
	// touch-or-start for out-of-order interim
	s.TouchOrStart(&radius.Session{Username: "b", AcctSessionID: "s2"})
	if len(s.Active()) != 1 {
		t.Fatal("touch must create")
	}
}
