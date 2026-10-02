package mikrotik_test

import (
	"bufio"
	"context"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/connectors/mikrotik"
	"github.com/universal-isp/platform/internal/connectors/sdk"
)

func writeWord(w *bufio.Writer, s string) {
	b := []byte(s)
	_ = w.WriteByte(byte(len(b)))
	_, _ = w.Write(b)
	_ = w.Flush()
}

func readWord(r *bufio.Reader) string {
	b, err := r.ReadByte()
	if err != nil {
		return ""
	}
	buf := make([]byte, int(b))
	for i := range buf {
		buf[i], _ = r.ReadByte()
	}
	return string(buf)
}

// scriptFake is a programmable RouterOS API stub for protocol tests.
// Each connection: login sentence -> !done; then repeated command sentences.
func scriptFake(t *testing.T, ln net.Listener, scripts map[string][]map[string]string) {
	t.Helper()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				r := bufio.NewReader(c)
				w := bufio.NewWriter(c)
				for i := 0; i < 4; i++ {
					readWord(r)
				}
				writeWord(w, "!done")
				writeWord(w, "")
				for {
					var words []string
					for {
						s := readWord(r)
						if s == "" {
							break
						}
						words = append(words, s)
					}
					if len(words) == 0 {
						return
					}
					rows := scripts[words[0]]
					for _, row := range rows {
						writeWord(w, "!re")
						for k, v := range row {
							writeWord(w, "="+k+"="+v)
						}
					}
					writeWord(w, "!done")
					writeWord(w, "")
				}
			}(conn)
		}
	}()
}

func dialTest(t *testing.T, scripts map[string][]map[string]string) *mikrotik.Connector {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("no loopback")
	}
	t.Cleanup(func() { ln.Close() })
	scriptFake(t, ln, scripts)
	_, port, _ := net.SplitHostPort(ln.Addr().String())
	p, _ := strconv.Atoi(port)
	return mikrotik.New(mikrotik.Config{Host: "127.0.0.1", Port: p, Username: "admin", Password: "x", Timeout: 3 * time.Second})
}

func TestProtocolDeviceInfo(t *testing.T) {
	c := dialTest(t, map[string][]map[string]string{
		"/system/resource/print": {{"board-name": "CRS326", "version": "7.12", "uptime": "1d2h3m"}},
	})
	ctx := context.Background()
	if err := c.TestConnection(ctx); err != nil {
		t.Fatal(err)
	}
	info, err := c.GetDeviceInfo(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if info.Model != "CRS326" || info.Firmware != "7.12" {
		t.Fatalf("info=%+v", info)
	}
	if info.UptimeSecs != 86400+7200+180 {
		t.Fatalf("uptime=%d", info.UptimeSecs)
	}
}

func TestProtocolSubscribers(t *testing.T) {
	c := dialTest(t, map[string][]map[string]string{
		"/ppp/secret/print":           {{"name": "alice", "profile": "default"}},
		"/ppp/active/print":           {{"name": "alice", "address": "10.0.0.2"}},
		"/ip/dhcp-server/lease/print": {{"mac-address": "AA:BB", "address": "10.0.0.3"}},
		"/queue/simple/print":         {},
		"/queue/simple/add":           {},
		"/ppp/profile/print":          {},
		"/ppp/profile/add":            {},
	})
	ctx := context.Background()
	secrets, err := c.GetPPPSecrets(ctx)
	if err != nil || len(secrets) != 1 || secrets[0].Name != "alice" {
		t.Fatalf("secrets=%v err=%v", secrets, err)
	}
	active, err := c.GetPPPActive(ctx)
	if err != nil || len(active) != 1 || active[0].Address != "10.0.0.2" {
		t.Fatalf("active=%v err=%v", active, err)
	}
	leases, err := c.GetDHCPLeases(ctx)
	if err != nil || len(leases) != 1 {
		t.Fatalf("leases=%v err=%v", leases, err)
	}
	clients, err := c.GetClients(ctx)
	if err != nil || len(clients) != 1 || clients[0].MAC != "AA:BB" {
		t.Fatalf("clients=%v err=%v", clients, err)
	}
	if err := c.EnsurePPPProfile(ctx, "TIER-50M", 50, 20); err != nil {
		t.Fatal(err)
	}
	if err := c.ProvisionSubscriber(ctx, sdk.ProvisionRequest{SubscriberID: "s1", Username: "bob", ServiceType: "pppoe", DownloadMbps: 50, UploadMbps: 20, IdempotencyKey: "k1"}); err != nil {
		t.Fatal(err)
	}
}

func TestCapabilitiesHonest(t *testing.T) {
	c := mikrotik.New(mikrotik.Config{Host: "127.0.0.1"})
	caps, err := c.GetCapabilities(context.Background())
	if err != nil || caps == nil {
		t.Fatal("caps")
	}
	if !caps.Has(sdk.CapPPPoE) {
		t.Fatal("mikrotik must advertise pppoe")
	}
	if st, _ := caps.StatusOf(sdk.CapPPPoE); st != sdk.Partial {
		t.Fatalf("pppoe must be PARTIAL until hardware E2E, got %s", st)
	}
}

// TestE2E runs against REAL hardware only when MIKROTIK_E2E=true.
// Required env: MIKROTIK_HOST, MIKROTIK_USERNAME, MIKROTIK_PASSWORD.
// Optional: MIKROTIK_PORT, MIKROTIK_TLS=1. Never commit credentials.
func TestE2E(t *testing.T) {
	if strings.ToUpper(env("MIKROTIK_E2E", "")) != "TRUE" {
		t.Skip("MIKROTIK_E2E not set")
	}
	host := env("MIKROTIK_HOST", "")
	if host == "" {
		t.Fatal("MIKROTIK_HOST required")
	}
	port := 8728
	if p := env("MIKROTIK_PORT", ""); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}
	c := mikrotik.New(mikrotik.Config{Host: host, Port: port, Username: env("MIKROTIK_USERNAME", ""), Password: env("MIKROTIK_PASSWORD", ""), UseTLS: env("MIKROTIK_TLS", "") == "1", Timeout: 10 * time.Second})
	ctx := context.Background()
	if err := c.TestConnection(ctx); err != nil {
		t.Fatalf("E2E connection failed: %v", err)
	}
	if _, err := c.GetDeviceInfo(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := c.GetInterfaces(ctx); err != nil {
		t.Fatal(err)
	}
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
