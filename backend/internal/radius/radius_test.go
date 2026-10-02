package radius_test

import (
	"testing"

	"github.com/universal-isp/platform/internal/radius"
)

func TestCodecRoundTrip(t *testing.T) {
	p := &radius.Packet{Code: radius.CodeAccessRequest, Identifier: 7, Attrs: []radius.Attr{{Type: 1, Value: []byte("alice")}}}
	raw := radius.Encode(p, "secret", p.Authenticator)
	got, err := radius.Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.GetString(1) != "alice" {
		t.Fatalf("user=%q", got.GetString(1))
	}
}

func TestSessionStore(t *testing.T) {
	s := radius.NewSessionStore()
	s.Start(&radius.Session{Username: "u", AcctSessionID: "s1"})
	s.Interim("s1", 10, 20)
	if len(s.Active()) != 1 {
		t.Fatal("want 1 active")
	}
	s.Stop("s1")
	if len(s.Active()) != 0 {
		t.Fatal("want 0 active")
	}
}
