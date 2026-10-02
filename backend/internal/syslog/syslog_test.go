package syslog_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/syslog"
)

func TestParse(t *testing.T) {
	m := syslog.Parse("<34>Oct 11 22:14:15 mymachine su: 'su root' failed")
	if m.Facility != 4 || m.Severity != 2 {
		t.Fatalf("pri decode: %+v", m)
	}
	if m.Host != "mymachine" {
		t.Fatalf("host: %+v", m)
	}
}
