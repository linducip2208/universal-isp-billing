// RadSec: RADIUS over TLS (RFC 6614). Same packet codec and server logic
// as UDP, transported over a TLS stream (packets self-delimit via Length).
// Status: PROTOCOL_IMPLEMENTED + loopback-tested; production needs proper
// PKI (server certs, client-cert policy) — documented in SECURITY.md.
package radius

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"
)

// ServeTLS runs auth+acct over one TLS port until ctx ends.
func (s *Server) ServeTLS(ctx context.Context, addr string, cfg *tls.Config) error {
	ln, err := tls.Listen("tcp", addr, cfg)
	if err != nil {
		return fmt.Errorf("radsec listen %s: %w", addr, err)
	}
	defer ln.Close()
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()
	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				continue
			}
		}
		go s.serveTLSConn(ctx, conn)
	}
}

func (s *Server) serveTLSConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	ip, _, _ := net.SplitHostPort(conn.RemoteAddr().String())
	for {
		_ = conn.SetDeadline(time.Now().Add(60 * time.Second))
		hdr := make([]byte, 20)
		if _, err := io.ReadFull(conn, hdr); err != nil {
			return
		}
		length := int(hdr[2])<<8 | int(hdr[3])
		if length < 20 || length > 4096 {
			return
		}
		raw := hdr
		if length > 20 {
			rest := make([]byte, length-20)
			if _, err := io.ReadFull(conn, rest); err != nil {
				return
			}
			raw = append(hdr, rest...)
		}
		pkt, err := Decode(raw)
		if err != nil {
			return
		}
		if s.NAS != nil {
			if _, ok := s.NAS.Authorized(ip); !ok {
				continue
			}
		}
		var resp *Packet
		switch pkt.Code {
		case CodeAccessRequest:
			resp = s.handleAuth(pkt)
		case CodeAccountingReq:
			resp = s.handleAccounting(pkt, ip)
		default:
			continue
		}
		if _, err := conn.Write(Encode(resp, s.Secret, pkt.Authenticator)); err != nil {
			return
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}
}

// SendTLS transmits one packet over RadSec and returns the decoded response.
func SendTLS(addr string, cfg *tls.Config, pkt *Packet, timeout time.Duration) (*Packet, error) {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	d := &net.Dialer{Timeout: timeout}
	conn, err := tls.DialWithDialer(d, "tcp", addr, cfg)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if _, err := conn.Write(EncodeRequest(pkt)); err != nil {
		return nil, err
	}
	hdr := make([]byte, 20)
	if _, err := io.ReadFull(conn, hdr); err != nil {
		return nil, err
	}
	length := int(hdr[2])<<8 | int(hdr[3])
	raw := hdr
	if length > 20 {
		rest := make([]byte, length-20)
		if _, err := io.ReadFull(conn, rest); err != nil {
			return nil, err
		}
		raw = append(hdr, rest...)
	}
	return Decode(raw)
}
