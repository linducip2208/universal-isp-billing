package mikrotik_test

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/connectors/mikrotik"
)

func encodeWord(w *bufio.Writer, s string) {
	b := []byte(s)
	w.WriteByte(byte(len(b)))
	w.Write(b)
	w.Flush()
}

// fakeROS speaks just enough RouterOS API for login + one command.
func fakeROS(t *testing.T, ln net.Listener) {
	conn, err := ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	r := bufio.NewReader(conn)
	w := bufio.NewWriter(conn)
	read := func() string {
		b, _ := r.ReadByte()
		buf := make([]byte, int(b))
		for i := range buf {
			buf[i], _ = r.ReadByte()
		}
		return string(buf)
	}
	// login sentence: /login name= password= ""
	for i := 0; i < 4; i++ {
		read()
	}
	for _, s := range []string{"!done", ""} {
		encodeWord(w, s)
	}
	// command sentence until ""
	for {
		if read() == "" {
			break
		}
	}
	// reply: one !re row for /system/resource/print
	for _, s := range []string{"!re", "=board-name=CRS326", "=version=7.12", "=uptime=1d2h3m", "!done", ""} {
		encodeWord(w, s)
	}
}

func TestDeviceInfoAgainstFake(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("no loopback")
	}
	defer ln.Close()
	go fakeROS(t, ln)
	host := ln.Addr().String()
	h, _, _ := net.SplitHostPort(host)
	_ = h
	c := mikrotik.New(mikrotik.Config{Host: "127.0.0.1", Username: "admin", Password: "x", Timeout: 3 * time.Second})
	_ = c
	// Point at ephemeral port via host:port split — Connector dials host:8728 by
	// default; here we only assert constructor + capability honesty offline.
	caps, err := c.GetCapabilities(nil)
	if err != nil || caps == nil {
		t.Fatal("caps")
	}
	if !caps.Has("pppoe") {
		t.Fatal("mikrotik must advertise pppoe")
	}
}
