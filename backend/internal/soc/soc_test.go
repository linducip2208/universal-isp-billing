package soc_test

import (
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/soc"
)

func TestBruteForce(t *testing.T) {
	now := time.Now()
	var ev []soc.Event
	for i := 0; i < 6; i++ {
		ev = append(ev, soc.Event{At: now.Add(time.Duration(i) * time.Second), Actor: "root", Action: "login.failed"})
	}
	ev = append(ev, soc.Event{At: now, Actor: "ops", Action: "login.failed"})
	f := soc.Analyze(ev)
	if len(f) != 1 || f[0].Rule != "brute-force" || f[0].Actor != "root" {
		t.Fatalf("findings=%+v", f)
	}
}

func TestOffHours(t *testing.T) {
	at := time.Date(2026, 10, 2, 3, 0, 0, 0, time.UTC)
	f := soc.Analyze([]soc.Event{{At: at, Actor: "tech", Action: "device.login"}})
	if len(f) != 1 || f[0].Rule != "off-hours-device-login" {
		t.Fatalf("findings=%+v", f)
	}
	day := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	if len(soc.Analyze([]soc.Event{{At: day, Actor: "tech", Action: "device.login"}})) != 0 {
		t.Fatal("business hours must not flag")
	}
}
