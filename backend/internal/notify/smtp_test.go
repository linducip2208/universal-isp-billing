package notify_test

import (
	"bufio"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/notify"
)

// fakeSMTP speaks minimal SMTP: greeting, EHLO, MAIL, RCPT, DATA, capture.
func fakeSMTP(t *testing.T) (addr string, got *string) {
	t.Helper()
	var body string
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skip("no tcp")
	}
	t.Cleanup(func() { ln.Close() })
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		r := bufio.NewReader(conn)
		w := bufio.NewWriter(conn)
		say := func(s string) { w.WriteString(s + "\r\n"); w.Flush() }
		say("220 fake ESMTP")
		inData := false
		var data strings.Builder
		for {
			line, err := r.ReadString('\n')
			if err != nil {
				return
			}
			line = strings.TrimRight(line, "\r\n")
			if inData {
				if line == "." {
					inData = false
					body = data.String()
					say("250 ok")
					continue
				}
				data.WriteString(line + "\n")
				continue
			}
			up := strings.ToUpper(line)
			switch {
			case strings.HasPrefix(up, "EHLO"), strings.HasPrefix(up, "HELO"):
				say("250-fake")
				say("250 AUTH PLAIN")
			case strings.HasPrefix(up, "AUTH"):
				say("235 ok")
			case strings.HasPrefix(up, "MAIL"), strings.HasPrefix(up, "RCPT"):
				say("250 ok")
			case strings.HasPrefix(up, "DATA"):
				say("354 end")
				inData = true
			case strings.HasPrefix(up, "QUIT"):
				say("221 bye")
				return
			default:
				say("250 ok")
			}
		}
	}()
	return ln.Addr().String(), &body
}

func TestSMTPSend(t *testing.T) {
	addr, got := fakeSMTP(t)
	host, port, _ := strings.Cut(addr, ":")
	var p int
	for _, c := range port {
		p = p*10 + int(c-'0')
	}
	m := notify.SMTPMailer{Host: host, Port: p, From: "noreply@isp.test"}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := m.Send(ctx, "user@example.id", "Invoice due", "Pay soon"); err != nil {
		t.Fatalf("send: %v", err)
	}
	if !strings.Contains(*got, "Subject: Invoice due") || !strings.Contains(*got, "Pay soon") {
		t.Fatalf("captured=%q", *got)
	}
	if err := (notify.SMTPMailer{}).Send(ctx, "a@b", "s", "b"); err == nil {
		t.Fatal("unconfigured must fail")
	}
}
