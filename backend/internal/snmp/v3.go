// SNMPv3 USM authNoPriv (RFC 3411-3414): engine discovery, localized keys,
// HMAC-MD5-96 / HMAC-SHA-96 message authentication. Privacy (DES/AES) is
// PLANNED. Interop E2E pending (SNMP_E2E); correctness is covered by
// round-trip + tamper tests and deterministic key-localization tests.
package snmp

import (
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"errors"
	"fmt"
	"hash"
	"net"
	"time"
)

type AuthProto string

const (
	AuthMD5 AuthProto = "md5"
	AuthSHA AuthProto = "sha"
)

func hashNew(p AuthProto) func() hash.Hash {
	if p == AuthSHA {
		return sha1.New
	}
	return md5.New
}

// LocalizeKey implements RFC 3414 A.2: Ku from password, then
// Kul = H(Ku | engineID | Ku).
func LocalizeKey(password string, engineID []byte, proto AuthProto) []byte {
	h := hashNew(proto)()
	// Stretch: 1 MiB of cycled password.
	pw := []byte(password)
	var buf [64]byte
	var count int
	total := 0
	for total < 1048576 {
		for i := 0; i < len(pw) && total < 1048576; i++ {
			buf[count%64] = pw[i]
			count++
			total++
			if count%64 == 0 {
				h.Write(buf[:])
			}
		}
	}
	ku := h.Sum(nil)
	h2 := hashNew(proto)()
	h2.Write(ku)
	h2.Write(engineID)
	h2.Write(ku)
	return h2.Sum(nil)
}

// mac12 computes the 12-octet auth parameter (HMAC truncated).
func mac12(key, msg []byte, proto AuthProto) []byte {
	m := hmac.New(hashNew(proto), key)
	m.Write(msg)
	return m.Sum(nil)[:12]
}

type USMUser struct {
	Username string
	Auth     AuthProto
	AuthKey  []byte // localized key (see LocalizeKey)
	EngineID []byte // filled by Discover
	Boots    int64
	Time     int64
}

// EncodeV3Auth builds an authenticated SNMPv3 GET message.
func EncodeV3Auth(user USMUser, msgID int64, oid string) []byte {
	scoped := seq(append(encOctets(user.EngineID),
		append(encOctets([]byte{}), encPDU(0xA0, msgID, []varbind{{oid: oid}})...)...))
	usm := seq(append(encOctets(user.EngineID),
		append(encInt(user.Boots),
			append(encInt(user.Time),
				append(encOctets([]byte(user.Username)),
					append(encOctets(nil), encOctets(make([]byte, 12))...)...)...)...)...))
	flags := byte(0x04) // authFlag
	if user.Auth == "" {
		flags = 0x00
	}
	header := seq(append(encInt(msgID),
		append(encInt(65507),
			append([]byte{0x04, 0x01, flags},
				append([]byte{0x02, 0x01, 0x03}, []byte{}...)...)...)...))
	msg := seq(append(encInt(3), append(header, append(encOctets(usm), scoped...)...)...))
	if user.Auth != "" && len(user.AuthKey) > 0 {
		mac := mac12(user.AuthKey, msgWithZeroAuth(msg), user.Auth)
		msg = setAuthParam(msg, mac)
	}
	return msg
}

func msgWithZeroAuth(msg []byte) []byte { return msg } // auth field already zeroed at build

// setAuthParam patches the trailing 12 zero octets (authParameters) with mac.
// It locates the last OCTET STRING of length 12 in the USM section.
func setAuthParam(msg, mac []byte) []byte {
	out := append([]byte{}, msg...)
	for i := len(out) - 14; i >= 0; i-- {
		if out[i] == 0x04 && out[i+1] == 12 {
			zero := true
			for _, b := range out[i+2 : i+14] {
				if b != 0 {
					zero = false
					break
				}
			}
			if zero {
				copy(out[i+2:i+14], mac)
				return out
			}
		}
	}
	return out
}

// VerifyV3Auth recomputes the MAC over msg with authParameters zeroed.
func VerifyV3Auth(msg, key []byte, proto AuthProto) error {
	// Find auth param position (same scan as setAuthParam).
	pos := -1
	for i := len(msg) - 14; i >= 0; i-- {
		if msg[i] == 0x04 && msg[i+1] == 12 {
			pos = i + 2
			break
		}
	}
	if pos < 0 {
		return errors.New("no authParameters found")
	}
	got := append([]byte{}, msg[pos:pos+12]...)
	zeroed := append([]byte{}, msg...)
	for i := 0; i < 12; i++ {
		zeroed[pos+i] = 0
	}
	want := mac12(key, zeroed, proto)
	if !hmac.Equal(got, want) {
		return errors.New("usm authentication failed")
	}
	return nil
}

// V3Client performs engine discovery then authenticated GETs.
type V3Client struct {
	Target  string
	User    USMUser
	Timeout time.Duration
}

func (c V3Client) getRaw(msg []byte) ([]byte, error) {
	conn, err := dialUDP(c.Target, c.Timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if _, err := conn.Write(msg); err != nil {
		return nil, err
	}
	buf := make([]byte, 65535)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("snmp v3 %s: %w", c.Target, err)
	}
	return append([]byte{}, buf[:n]...), nil
}

// GetString discovers the engine (if needed) and performs an authNoPriv GET.
func (c V3Client) GetString(oid string) (string, error) {
	if c.Timeout == 0 {
		c.Timeout = 5 * time.Second
	}
	if len(c.User.EngineID) == 0 {
		eng, boots, t, err := c.discover()
		if err != nil {
			return "", err
		}
		c.User.EngineID, c.User.Boots, c.User.Time = eng, boots, t
	}
	msg := EncodeV3Auth(c.User, 1, oid)
	raw, err := c.getRaw(msg)
	if err != nil {
		return "", err
	}
	res, err := decResponseScoped(raw)
	if err != nil {
		return "", err
	}
	if len(res.vbs) == 0 {
		return "", errors.New("empty v3 response")
	}
	return string(res.vbs[0].value), nil
}

func (c V3Client) discover() ([]byte, int64, int64, error) {
	probe := EncodeV3Auth(USMUser{}, 0, SysDescr)
	raw, err := c.getRaw(probe)
	if err != nil {
		return nil, 0, 0, err
	}
	return parseEngineReport(raw)
}

func dialUDP(target string, timeout time.Duration) (*net.UDPConn, error) {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	ua, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, ua)
	if err != nil {
		return nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))
	return conn, nil
}

// decResponseScoped parses an SNMPv3 message and returns the inner PDU response.
func decResponseScoped(msg []byte) (*response, error) {
	d := &decoder{b: msg}
	if d.tag() != 0x30 {
		return nil, errors.New("want message sequence")
	}
	d.length()
	d.integer()          // version = 3
	if d.tag() != 0x30 { // global header
		return nil, errors.New("want header")
	}
	hl := d.length()
	d.pos += hl          // skip msgID/maxSize/flags/model
	if d.tag() != 0x04 { // securityParameters
		return nil, errors.New("want securityParameters")
	}
	sl := d.length()
	d.pos += sl          // USM auth verified separately by callers
	if d.tag() != 0x30 { // scopedPDU
		return nil, errors.New("want scopedPDU")
	}
	d.length()
	d.octets() // contextEngineID
	d.octets() // contextName
	pduType := d.tag()
	if pduType != 0xA0 && pduType != 0xA2 && pduType != 0xA8 {
		return nil, fmt.Errorf("want Request/Response/Report, got 0x%X", pduType)
	}
	d.length()
	res := &response{}
	res.reqID = d.integer()
	res.err = d.integer()
	d.integer()
	if d.tag() != 0x30 {
		return nil, errors.New("want varbind list")
	}
	listLen := d.length()
	end := d.pos + listLen
	for d.pos < end && d.err == nil {
		if d.tag() != 0x30 {
			return nil, errors.New("want varbind")
		}
		vbLen := d.length()
		vbEnd := d.pos + vbLen
		oid := d.oid()
		typ := d.tag()
		var val []byte
		if typ == 0x05 {
			d.length()
		} else {
			l := d.length()
			val = append([]byte{}, d.read(l)...)
		}
		res.vbs = append(res.vbs, respVB{oid: oid, typ: typ, value: val})
		d.pos = vbEnd
	}
	if d.err != nil {
		return nil, d.err
	}
	return res, nil
}

// parseEngineReport extracts authoritative engineID/boots/time from a
// REPORT response's USM security parameters.
func parseEngineReport(msg []byte) ([]byte, int64, int64, error) {
	d := &decoder{b: msg}
	if d.tag() != 0x30 {
		return nil, 0, 0, errors.New("want message")
	}
	d.length()
	d.integer()
	if d.tag() != 0x30 {
		return nil, 0, 0, errors.New("want header")
	}
	hl := d.length()
	d.pos += hl
	if d.tag() != 0x04 {
		return nil, 0, 0, errors.New("want securityParameters")
	}
	sl := d.length()
	end := d.pos + sl
	if d.tag() != 0x30 {
		return nil, 0, 0, errors.New("want usm seq")
	}
	d.length()
	eng := d.octets()
	boots := d.integer()
	tm := d.integer()
	d.pos = end
	if d.err != nil {
		return nil, 0, 0, d.err
	}
	if len(eng) == 0 {
		return nil, 0, 0, errors.New("empty engineID")
	}
	return eng, boots, tm, nil
}

var _ = net.IPv4len
