package radius_test

import (
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/radius"
)

func TestNASAuth(t *testing.T) {
	ns := radius.NewNASStore()
	ns.Upsert(&radius.NAS{ID: "n1", IP: "10.0.0.1", Enabled: true})
	if _, ok := ns.Authorized("10.0.0.1"); !ok {
		t.Fatal("should authorize")
	}
	if _, ok := ns.Authorized("10.0.0.2"); ok {
		t.Fatal("unknown NAS must fail")
	}
	ns.Upsert(&radius.NAS{ID: "n1", IP: "10.0.0.1", Enabled: false})
	if _, ok := ns.Authorized("10.0.0.1"); ok {
		t.Fatal("disabled NAS must fail")
	}
}

func TestDedupReplay(t *testing.T) {
	d := radius.NewDedup(50 * time.Millisecond)
	var auth [16]byte
	auth[0] = 7
	if _, ok := d.Get("1.1.1.1", 3, auth); ok {
		t.Fatal("empty miss expected")
	}
	d.Put("1.1.1.1", 3, auth, []byte{1, 2, 3})
	got, ok := d.Get("1.1.1.1", 3, auth)
	if !ok || len(got) != 3 {
		t.Fatal("want hit")
	}
	time.Sleep(70 * time.Millisecond)
	if _, ok := d.Get("1.1.1.1", 3, auth); ok {
		t.Fatal("want expiry")
	}
}

func TestUsageNoDoubleCount(t *testing.T) {
	u := radius.NewUsageStore()
	u.Interim("alice", "s1", 1000, 2000)
	u.Interim("alice", "s1", 1000, 2000) // retransmit: no progress
	u.Interim("alice", "s1", 1500, 2500) // +500/+500
	got := u.Get("alice", time.Now())
	if got.InOctets != 1500 || got.OutOctets != 2500 {
		t.Fatalf("usage=%+v", got)
	}
	if got.Sessions != 1 {
		t.Fatalf("sessions=%d", got.Sessions)
	}
	u.Stop("s1")
}
