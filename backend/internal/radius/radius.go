// Package radius implements a native Go RADIUS server core:
// packet codec (RFC 2865/2866), authentication/authorization, accounting,
// session tracking, CoA/Disconnect (RFC 5176) client, and a flexible
// vendor attribute dictionary (NOT MikroTik-only).
package radius

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

const (
	CodeAccessRequest  = 1
	CodeAccessAccept   = 2
	CodeAccessReject   = 3
	CodeAccountingReq  = 4
	CodeAccountingResp = 5
	CodeCoARequest     = 43
	CodeCoAAck         = 44
	CodeDisconnectReq  = 40
	CodeDisconnectAck  = 41
)

type Attr struct {
	Type  byte
	Value []byte
}

type Packet struct {
	Code          byte
	Identifier    byte
	Authenticator [16]byte
	Attrs         []Attr
}

func (p *Packet) Get(t byte) []byte {
	for _, a := range p.Attrs {
		if a.Type == t {
			return a.Value
		}
	}
	return nil
}

func (p *Packet) GetString(t byte) string { return string(p.Get(t)) }

// Encode serializes with ResponseAuth computed from secret.
func Encode(p *Packet, secret string, reqAuth [16]byte) []byte {
	attrs := []byte{}
	for _, a := range p.Attrs {
		attrs = append(attrs, a.Type, byte(len(a.Value)+2))
		attrs = append(attrs, a.Value...)
	}
	buf := make([]byte, 20+len(attrs))
	buf[0] = p.Code
	buf[1] = p.Identifier
	binary.BigEndian.PutUint16(buf[2:4], uint16(20+len(attrs)))
	copy(buf[4:20], reqAuth[:])
	copy(buf[20:], attrs)
	h := md5.New()
	h.Write(buf)
	h.Write([]byte(secret))
	copy(buf[4:20], h.Sum(nil))
	return buf
}

// EncodeRequest serializes an Access-/Accounting-Request, preserving the
// random request authenticator (needed for PAP hiding). Encode (above) is
// for responses, where the authenticator becomes the ResponseAuth.
func EncodeRequest(p *Packet) []byte {
	attrs := []byte{}
	for _, a := range p.Attrs {
		attrs = append(attrs, a.Type, byte(len(a.Value)+2))
		attrs = append(attrs, a.Value...)
	}
	buf := make([]byte, 20+len(attrs))
	buf[0] = p.Code
	buf[1] = p.Identifier
	binary.BigEndian.PutUint16(buf[2:4], uint16(20+len(attrs)))
	copy(buf[4:20], p.Authenticator[:])
	copy(buf[20:], attrs)
	return buf
}

// Decode parses a datagram; secret is used only by callers for verification.
func Decode(b []byte) (*Packet, error) {
	if len(b) < 20 {
		return nil, errors.New("packet too short")
	}
	length := int(binary.BigEndian.Uint16(b[2:4]))
	if length > len(b) || length < 20 {
		return nil, errors.New("bad length")
	}
	p := &Packet{Code: b[0], Identifier: b[1]}
	copy(p.Authenticator[:], b[4:20])
	rest := b[20:length]
	for len(rest) >= 2 {
		t, l := rest[0], int(rest[1])
		if l < 2 || l > len(rest) {
			break
		}
		p.Attrs = append(p.Attrs, Attr{Type: t, Value: append([]byte{}, rest[2:l]...)})
		rest = rest[l:]
	}
	return p, nil
}

// --- Dictionary: flexible vendor attributes ---

type VendorAttr struct {
	VendorID uint32 `json:"vendor_id"`
	Type     byte   `json:"type"`
	Name     string `json:"name"`
}

var (
	dictMu sync.RWMutex
	dict   = map[string]VendorAttr{
		"Mikrotik-Rate-Limit":         {VendorID: 14988, Type: 8, Name: "Mikrotik-Rate-Limit"},
		"Mikrotik-Address-List":       {VendorID: 14988, Type: 19, Name: "Mikrotik-Address-List"},
		"Cisco-AVPair":                {VendorID: 9, Type: 1, Name: "Cisco-AVPair"},
		"Huawei-Input-Committed-Rate": {VendorID: 2011, Type: 2, Name: "Huawei-Input-Committed-Rate"},
		"ZTE-Rate":                    {VendorID: 3902, Type: 1, Name: "ZTE-Rate"},
	}
)

func RegisterVendorAttr(name string, va VendorAttr) {
	dictMu.Lock()
	defer dictMu.Unlock()
	dict[name] = va
}

func LookupVendorAttr(name string) (VendorAttr, bool) {
	dictMu.RLock()
	defer dictMu.RUnlock()
	va, ok := dict[name]
	return va, ok
}

// --- Sessions ---

type Session struct {
	Username      string
	NASIP         string
	AcctSessionID string
	FramedIP      string
	InputOctets   int64
	OutputOctets  int64
	StartedAt     time.Time
	LastInterim   time.Time
}

type SessionStore struct {
	mu sync.RWMutex
	m  map[string]*Session // acctSessionID -> session
}

func NewSessionStore() *SessionStore { return &SessionStore{m: map[string]*Session{}} }

func (s *SessionStore) Start(sess *Session) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess.StartedAt = time.Now()
	s.m[sess.AcctSessionID] = sess
}

func (s *SessionStore) Interim(id string, in, out int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if se, ok := s.m[id]; ok {
		se.InputOctets = in
		se.OutputOctets = out
		se.LastInterim = time.Now()
	}
}

func (s *SessionStore) Stop(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.m, id)
}

func (s *SessionStore) Active() []*Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Session, 0, len(s.m))
	for _, v := range s.m {
		out = append(out, v)
	}
	return out
}

// VerifyUser is the pluggable credential checker used by Server.
type VerifyUser func(username, password string) (attrs map[string]string, ok bool)

// Server is a UDP RADIUS server for auth+acct with NAS authorization,
// anti-replay dedup, session tracking, and usage aggregation. Optional
// subsystems (NAS, Dedup, Usage) are nil-safe: absent means allow-all,
// no dedup, no usage — production wires all three.
type Server struct {
	Secret   string
	Verify   VerifyUser
	Sessions *SessionStore
	NAS      *NASStore
	Dedup    *Dedup
	Usage    *UsageStore
}

func responsePacket(code, id byte, attrs []Attr) *Packet {
	return &Packet{Code: code, Identifier: id, Attrs: attrs}
}

func (s *Server) handleAuth(pkt *Packet) *Packet {
	user := pkt.GetString(1)
	// User-Password arrives MD5-hidden per RFC 2865; decrypt first. A bare
	// plaintext attribute (loopback/test clients) is accepted as fallback —
	// real NASes always send ciphertext.
	pass := ""
	if raw := pkt.Get(2); raw != nil {
		if dec, err := DecryptPAP(s.Secret, pkt.Authenticator, raw); err == nil {
			pass = dec
		} else {
			pass = string(raw)
		}
	}
	if s.Verify != nil {
		if attrs, ok := s.Verify(user, pass); ok {
			var out []Attr
			for k, v := range attrs {
				// Map common keys to standard attrs; vendor keys pass through as VSA text for now.
				switch k {
				case "Framed-IP-Address":
					ip := net.ParseIP(v).To4()
					if ip != nil {
						out = append(out, Attr{Type: 8, Value: []byte(ip)})
					}
				case "Session-Timeout":
					out = append(out, Attr{Type: 27, Value: []byte(v)})
				default:
					out = append(out, Attr{Type: 26, Value: []byte(k + "=" + v)})
				}
			}
			return responsePacket(CodeAccessAccept, pkt.Identifier, out)
		}
	}
	return responsePacket(CodeAccessReject, pkt.Identifier, nil)
}

func (s *Server) Serve(authAddr, acctAddr string) error {
	return s.ServeCtx(context.Background(), authAddr, acctAddr)
}

// ServeCtx runs auth + accounting listeners until ctx ends.
func (s *Server) ServeCtx(ctx context.Context, authAddr, acctAddr string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errCh := make(chan error, 2)
	go func() { errCh <- s.serveOne(ctx, authAddr, true) }()
	go func() { errCh <- s.serveOne(ctx, acctAddr, false) }()
	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) serveOne(ctx context.Context, addr string, isAuth bool) error {
	ua, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", ua)
	if err != nil {
		return err
	}
	defer conn.Close()
	go func() {
		<-ctx.Done()
		_ = conn.Close()
	}()
	buf := make([]byte, 4096)
	for {
		n, remote, err := conn.ReadFromUDP(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				continue
			}
		}
		ip := remote.IP.String()
		// NAS authorization: unknown/disabled sources get nothing.
		if s.NAS != nil {
			if _, ok := s.NAS.Authorized(ip); !ok {
				if isAuth {
					rej := responsePacket(CodeAccessReject, buf[1], nil)
					var auth [16]byte
					copy(auth[:], buf[4:20])
					_, _ = conn.WriteToUDP(Encode(rej, s.Secret, auth), remote)
				}
				continue
			}
		}
		pkt, err := Decode(bytes.Clone(buf[:n]))
		if err != nil {
			continue
		}
		// Anti-replay: retransmits get the cached response bytes.
		if s.Dedup != nil {
			if cached, ok := s.Dedup.Get(ip, pkt.Identifier, pkt.Authenticator); ok {
				_, _ = conn.WriteToUDP(cached, remote)
				continue
			}
		}
		var resp *Packet
		switch pkt.Code {
		case CodeAccessRequest:
			if !isAuth {
				continue
			}
			resp = s.handleAuth(pkt)
		case CodeAccountingReq:
			if isAuth {
				continue
			}
			resp = s.handleAccounting(pkt, ip)
		default:
			continue
		}
		raw := Encode(resp, s.Secret, pkt.Authenticator)
		if s.Dedup != nil {
			s.Dedup.Put(ip, pkt.Identifier, pkt.Authenticator, raw)
		}
		_, _ = conn.WriteToUDP(raw, remote)
	}
}

func u32(b []byte) uint32 {
	if len(b) < 4 {
		return 0
	}
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3])
}

// handleAccounting tracks Start/Interim/Stop (Acct-Status-Type 40) and feeds
// usage aggregation. Shared by UDP and RadSec transports.
func (s *Server) handleAccounting(pkt *Packet, ip string) *Packet {
	sid := pkt.GetString(44)
	if sid == "" {
		sid = fmt.Sprintf("%s-%d", ip, pkt.Identifier)
	}
	user := pkt.GetString(1)
	switch string(pkt.Get(40)) {
	case string([]byte{1}): // Start
		s.Sessions.Start(&Session{Username: user, NASIP: ip, AcctSessionID: sid})
	case string([]byte{3}): // Interim-Update
		in, out := u32(pkt.Get(42)), u32(pkt.Get(43))
		s.Sessions.Interim(sid, int64(in), int64(out))
		if s.Usage != nil {
			s.Usage.Interim(user, sid, int64(in), int64(out))
		}
	case string([]byte{2}): // Stop
		s.Sessions.Stop(sid)
		if s.Usage != nil {
			s.Usage.Stop(sid)
		}
	}
	return responsePacket(CodeAccountingResp, pkt.Identifier, nil)
}
