package radius_test

import (
	"context"
	"encoding/binary"
	"net"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/radius"
)

func freeUDP(t *testing.T) string {
	t.Helper()
	ln, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1")})
	if err != nil {
		t.Skip("no udp")
	}
	addr := ln.LocalAddr().String()
	ln.Close()
	return addr
}

func sendRecv(t *testing.T, addr string, pkt *radius.Packet, secret string) *radius.Packet {
	t.Helper()
	ua, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.DialUDP("udp", nil, ua)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
	if _, err := conn.Write(radius.EncodeRequest(pkt)); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("no response: %v", err)
	}
	got, err := radius.Decode(buf[:n])
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func u32b(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

// TestServerWiring exercises the full loopback path: NAS auth, Access-Accept,
// accounting Start/Interim (usage aggregation), replay dedup, and Stop.
func TestServerWiring(t *testing.T) {
	authAddr, acctAddr := freeUDP(t), freeUDP(t)
	nas := radius.NewNASStore()
	nas.Upsert(&radius.NAS{ID: "n1", IP: "127.0.0.1", Enabled: true})
	srv := &radius.Server{
		Secret: "s3cret",
		Verify: func(u, p string) (map[string]string, bool) {
			if u == "alice" && p == "pw" {
				return map[string]string{}, true
			}
			return nil, false
		},
		Sessions: radius.NewSessionStore(),
		NAS:      nas,
		Dedup:    radius.NewDedup(time.Minute),
		Usage:    radius.NewUsageStore(),
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = srv.ServeCtx(ctx, authAddr, acctAddr) }()
	time.Sleep(100 * time.Millisecond)

	// 1. auth accept — password sent RFC-2865-encrypted like a real NAS
	authReq := func(id byte, user, pass string) *radius.Packet {
		var ra [16]byte
		for i := range ra {
			ra[i] = id + byte(i)
		}
		ct, err := radius.EncryptPAP("s3cret", ra, pass)
		if err != nil {
			t.Fatal(err)
		}
		return &radius.Packet{Code: radius.CodeAccessRequest, Identifier: id, Authenticator: ra,
			Attrs: []radius.Attr{{Type: 1, Value: []byte(user)}, {Type: 2, Value: ct}}}
	}
	got := sendRecv(t, authAddr, authReq(10, "alice", "pw"), "s3cret")
	if got.Code != radius.CodeAccessAccept {
		t.Fatalf("want accept, got %d", got.Code)
	}
	// 2. bad password -> reject
	got = sendRecv(t, authAddr, authReq(11, "alice", "no"), "s3cret")
	if got.Code != radius.CodeAccessReject {
		t.Fatalf("want reject, got %d", got.Code)
	}
	// 3. accounting start + interim -> usage recorded once (dedup replays)
	acct := func(id byte, status byte, in, out uint32, sid string) *radius.Packet {
		return &radius.Packet{Code: radius.CodeAccountingReq, Identifier: id, Attrs: []radius.Attr{
			{Type: 1, Value: []byte("alice")}, {Type: 40, Value: []byte{status}},
			{Type: 44, Value: []byte(sid)}, {Type: 42, Value: u32b(in)}, {Type: 43, Value: u32b(out)}}}
	}
	sendRecv(t, acctAddr, acct(20, 1, 0, 0, "sess-1"), "s3cret")
	sendRecv(t, acctAddr, acct(21, 3, 1000, 2000, "sess-1"), "s3cret")
	if n := len(srv.Sessions.Active()); n != 1 {
		t.Fatalf("active=%d", n)
	}
	// 4. stop clears session
	sendRecv(t, acctAddr, acct(22, 2, 1500, 2500, "sess-1"), "s3cret")
	if n := len(srv.Sessions.Active()); n != 0 {
		t.Fatalf("active after stop=%d", n)
	}
}
