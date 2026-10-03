package notify

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// SMTPMailer sends real email via SMTP (stdlib only). Tested against an
// in-process fake SMTP server (see smtp_test.go); production needs
// SMTP_HOST/PORT/USER/PASS + TLS.
type SMTPMailer struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	UseTLS   bool
	Timeout  time.Duration
}

func (m SMTPMailer) addr() string { return fmt.Sprintf("%s:%d", m.Host, m.Port) }

// Build composes a minimal RFC 5322 message (unit-testable, no network).
func (m SMTPMailer) Build(to, subject, body string) string {
	var sb strings.Builder
	sb.WriteString("From: " + m.From + "\r\n")
	sb.WriteString("To: " + to + "\r\n")
	sb.WriteString("Subject: " + subject + "\r\n")
	sb.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
	sb.WriteString(body)
	return sb.String()
}

func (m SMTPMailer) Send(ctx context.Context, to, subject, body string) error {
	if m.Host == "" {
		return fmt.Errorf("smtp: SMTP_HOST not configured")
	}
	timeout := m.Timeout
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	var c *smtp.Client
	done := make(chan error, 1)
	go func() {
		var conn net.Conn
		var err error
		d := net.Dialer{Timeout: timeout}
		if m.UseTLS {
			conn, err = tls.DialWithDialer(&d, "tcp", m.addr(), &tls.Config{ServerName: m.Host})
		} else {
			conn, err = d.DialContext(ctx, "tcp", m.addr())
		}
		if err != nil {
			done <- err
			return
		}
		c, err = smtp.NewClient(conn, m.Host)
		if err != nil {
			done <- err
			return
		}
		defer c.Close()
		if m.Username != "" {
			if ok, _ := c.Extension("AUTH"); ok {
				if err := c.Auth(smtp.PlainAuth("", m.Username, m.Password, m.Host)); err != nil {
					done <- err
					return
				}
			}
		}
		if err := c.Mail(m.From); err != nil {
			done <- err
			return
		}
		if err := c.Rcpt(to); err != nil {
			done <- err
			return
		}
		w, err := c.Data()
		if err != nil {
			done <- err
			return
		}
		if _, err := w.Write([]byte(m.Build(to, subject, body))); err != nil {
			done <- err
			return
		}
		done <- w.Close()
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		_ = c
		return err
	}
}
