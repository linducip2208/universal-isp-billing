package radius

import (
	"strings"
	"testing"

	"github.com/universal-isp/platform/internal/metrics"
)

func TestMeteredServer(t *testing.T) {
	reg := metrics.New()
	s := &Server{Secret: "x", Sessions: NewSessionStore(), Meter: reg,
		Verify: func(u, p string) (map[string]string, bool) { return nil, false }}
	pkt := &Packet{Code: CodeAccessRequest, Identifier: 1,
		Attrs: []Attr{{Type: 1, Value: []byte("bob")}}}
	if resp := s.handleAuth(pkt); resp.Code != CodeAccessReject {
		t.Fatalf("code=%d", resp.Code)
	}
	out := reg.Exposition()
	if !strings.Contains(out, "radius_auth") {
		t.Fatalf("no radius_auth in exposition:\n%s", out)
	}
	// nil meter must not panic
	plain := &Server{Secret: "x", Verify: func(u, p string) (map[string]string, bool) { return nil, false }}
	plain.handleAuth(pkt)
}
