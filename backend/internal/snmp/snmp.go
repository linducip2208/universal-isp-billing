// Package snmp implements a real SNMP v2c client (BER codec, GET, GETNEXT
// walk) plus vendor/model-aware polling profiles. SNMPv3 USM is roadmap
// (see Profile.Auth). Tested against an in-process fake agent; live agents
// via SNMP_E2E.
package snmp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Well-known OIDs.
const (
	SysDescr    = ".1.3.6.1.2.1.1.1.0"
	SysUpTime   = ".1.3.6.1.2.1.1.3.0"
	IfNumber    = ".1.3.6.1.2.1.2.1.0"
	IfDescr     = ".1.3.6.1.2.1.2.2.1.2"
	IfOper      = ".1.3.6.1.2.1.2.2.1.8"
	IfInOctets  = ".1.3.6.1.2.1.2.2.1.10"
	IfOutOctets = ".1.3.6.1.2.1.2.2.1.16"
	IfSpeed     = ".1.3.6.1.2.1.2.2.1.5"
	HrProcLoad  = ".1.3.6.1.2.1.25.3.3.1.2" // + index
	HrMemSize   = ".1.3.6.1.2.1.25.2.2.0"
	HrMemUsed   = ".1.3.6.1.2.1.25.2.3.1.6" // + index
	// MikroTik HEALTH-MIB (enterprise 14988.1.1.3.x)
	MtVoltage = ".1.3.6.1.4.1.14988.1.1.3.8.0"
	MtTemp    = ".1.3.6.1.4.1.14988.1.1.3.10.0"
)

// Profile maps a vendor/model to its CPU/memory OID strategy.
type Profile struct {
	Vendor      string `json:"vendor"`
	ModelPrefix string `json:"model_prefix,omitempty"`
	CPUOid      string `json:"cpu_oid,omitempty"` // walked; average
	MemUsedOid  string `json:"mem_used_oid,omitempty"`
	MemSizeOid  string `json:"mem_size_oid,omitempty"`
}

var profiles = []Profile{
	{Vendor: "MikroTik", CPUOid: HrProcLoad, MemUsedOid: HrMemUsed, MemSizeOid: HrMemSize},
	{Vendor: "Cisco", CPUOid: ".1.3.6.1.4.1.9.9.109.1.1.1.1.8", MemUsedOid: ".1.3.6.1.4.1.9.9.48.1.1.1.5", MemSizeOid: ".1.3.6.1.4.1.9.9.48.1.1.1.6"},
	{Vendor: "Generic", CPUOid: HrProcLoad, MemUsedOid: HrMemUsed, MemSizeOid: HrMemSize},
}

// ProfileFor returns the best-match profile (prefix match on sysDescr).
func ProfileFor(sysDescr string) Profile {
	lower := strings.ToLower(sysDescr)
	for _, p := range profiles {
		if p.Vendor != "Generic" && strings.Contains(lower, strings.ToLower(p.Vendor)) {
			return p
		}
	}
	return profiles[len(profiles)-1]
}

// --- BER codec (subset needed for v2c GET/GETNEXT) ---

func encLen(n int) []byte {
	if n < 0x80 {
		return []byte{byte(n)}
	}
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(n))
	i := 0
	for i < 7 && b[i] == 0 {
		i++
	}
	out := []byte{byte(0x80 | (8 - i))}
	return append(out, b[i:]...)
}

func encInt(v int64) []byte {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(v))
	i := 0
	for i < 7 && ((b[i] == 0 && b[i+1]&0x80 == 0) || (b[i] == 0xFF && b[i+1]&0x80 != 0)) {
		i++
	}
	return append([]byte{0x02}, append(encLen(8-i), b[i:]...)...)
}

func encOID(s string) []byte {
	parts := strings.Split(strings.Trim(s, "."), ".")
	if len(parts) < 2 {
		return []byte{0x06, 0x00}
	}
	nums := make([]int, len(parts))
	for i, p := range parts {
		n, _ := strconv.Atoi(p)
		nums[i] = n
	}
	body := []byte{byte(nums[0]*40 + nums[1])}
	for _, n := range nums[2:] {
		var stack []byte
		stack = append([]byte{byte(n & 0x7F)}, stack...)
		n >>= 7
		for n > 0 {
			stack = append([]byte{byte(n&0x7F | 0x80)}, stack...)
			n >>= 7
		}
		for i := 0; i < len(stack)-1; i++ {
			stack[i] |= 0x80
		}
		body = append(body, stack...)
	}
	return append([]byte{0x06}, append(encLen(len(body)), body...)...)
}

func encOctets(b []byte) []byte { return append([]byte{0x04}, append(encLen(len(b)), b...)...) }
func encNull() []byte           { return []byte{0x05, 0x00} }

func seq(payload []byte) []byte {
	return append([]byte{0x30}, append(encLen(len(payload)), payload...)...)
}

type varbind struct {
	oid   string
	typ   byte
	value []byte
}

func encVarbind(vb varbind) []byte {
	inner := encOID(vb.oid)
	switch vb.typ {
	case 0x02:
		inner = append(inner, append([]byte{0x02}, append(encLen(len(vb.value)), vb.value...)...)...)
	case 0x04:
		inner = append(inner, encOctets(vb.value)...)
	case 0x06:
		inner = append(inner, encOID(string(vb.value))...)
	case 0x43:
		inner = append(inner, append([]byte{0x43}, append(encLen(len(vb.value)), vb.value...)...)...)
	default:
		inner = append(inner, encNull()...)
	}
	return seq(inner)
}

func encPDU(pduType byte, reqID int64, vbs []varbind) []byte {
	body := encInt(reqID)
	body = append(body, encInt(0)...) // error-status
	body = append(body, encInt(0)...) // error-index
	var list []byte
	for _, vb := range vbs {
		list = append(list, encVarbind(vb)...)
	}
	body = append(body, seq(list)...)
	return append([]byte{pduType}, append(encLen(len(body)), body...)...)
}

func encMessage(community string, pdu []byte) []byte {
	body := encInt(1) // version 1 = v2c
	body = append(body, encOctets([]byte(community))...)
	body = append(body, pdu...)
	return seq(body)
}

type decoder struct {
	b   []byte
	pos int
	err error
}

func (d *decoder) read(n int) []byte {
	if d.err != nil || d.pos+n > len(d.b) {
		d.err = errors.New("ber overrun")
		return nil
	}
	out := d.b[d.pos : d.pos+n]
	d.pos += n
	return out
}

func (d *decoder) tag() byte { b := d.read(1); return b[0] }
func (d *decoder) length() int {
	b := d.tag()
	if b < 0x80 {
		return int(b)
	}
	n := int(b & 0x7F)
	v := 0
	for i := 0; i < n; i++ {
		v = v<<8 | int(d.tag())
	}
	return v
}

func (d *decoder) integer() int64 {
	if d.tag() != 0x02 {
		d.err = errors.New("want integer")
		return 0
	}
	n := d.length()
	raw := d.read(n)
	var v int64
	for _, b := range raw {
		v = v<<8 | int64(b)
	}
	if len(raw) > 0 && raw[0]&0x80 != 0 {
		v -= 1 << (8 * len(raw))
	}
	return v
}

func (d *decoder) octets() []byte {
	if d.tag() != 0x04 {
		d.err = errors.New("want octets")
		return nil
	}
	return d.read(d.length())
}

func (d *decoder) oid() string {
	if d.tag() != 0x06 {
		d.err = errors.New("want oid")
		return ""
	}
	raw := d.read(d.length())
	if len(raw) == 0 {
		return ""
	}
	out := []int{int(raw[0] / 40), int(raw[0] % 40)}
	v := 0
	for _, b := range raw[1:] {
		v = v<<7 | int(b&0x7F)
		if b&0x80 == 0 {
			out = append(out, v)
			v = 0
		}
	}
	var sb strings.Builder
	for i, n := range out {
		if i > 0 {
			sb.WriteByte('.')
		}
		fmt.Fprint(&sb, n)
	}
	return "." + sb.String()
}

type response struct {
	reqID int64
	err   int64
	vbs   []respVB
}

type respVB struct {
	oid   string
	typ   byte
	value []byte
}

func decResponse(msg []byte) (*response, error) {
	d := &decoder{b: msg}
	if d.tag() != 0x30 {
		return nil, errors.New("want sequence")
	}
	d.length()
	d.integer() // version
	d.octets()  // community
	pduType := d.tag()
	if pduType != 0xA2 {
		return nil, fmt.Errorf("want GetResponse, got 0x%X", pduType)
	}
	d.length()
	res := &response{}
	res.reqID = d.integer()
	res.err = d.integer()
	d.integer() // error-index
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
		switch typ {
		case 0x05:
			d.length()
		case 0x06:
			l := d.length()
			val = d.read(l)
		default:
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

// --- Client ---

type Client struct {
	Target    string
	Community string
	Timeout   time.Duration
	Retries   int
}

func (c Client) withDefaults() Client {
	if c.Timeout == 0 {
		c.Timeout = 3 * time.Second
	}
	if c.Retries < 0 {
		c.Retries = 0
	}
	if c.Retries == 0 {
		c.Retries = 2
	}
	return c
}

var reqSeq int64 = 1

func (c Client) exchange(pduType byte, vbs []varbind) (*response, error) {
	c = c.withDefaults()
	var last error
	for attempt := 0; attempt <= c.Retries; attempt++ {
		reqSeq++
		msg := encMessage(c.Community, encPDU(pduType, reqSeq, vbs))
		conn, err := net.DialTimeout("udp", c.Target, c.Timeout)
		if err != nil {
			last = err
			continue
		}
		_ = conn.SetDeadline(time.Now().Add(c.Timeout))
		if _, err := conn.Write(msg); err != nil {
			_ = conn.Close()
			last = err
			continue
		}
		buf := make([]byte, 65535)
		n, err := conn.Read(buf)
		_ = conn.Close()
		if err != nil {
			last = err
			continue
		}
		res, err := decResponse(buf[:n])
		if err != nil {
			last = err
			continue
		}
		if res.err != 0 {
			return res, fmt.Errorf("snmp error-status=%d", res.err)
		}
		return res, nil
	}
	return nil, fmt.Errorf("snmp %s: %w", c.Target, last)
}

// Get fetches one OID; returns raw type + bytes.
func (c Client) Get(oid string) (byte, []byte, error) {
	res, err := c.exchange(0xA0, []varbind{{oid: oid}})
	if err != nil {
		return 0, nil, err
	}
	if len(res.vbs) == 0 {
		return 0, nil, errors.New("empty response")
	}
	return res.vbs[0].typ, res.vbs[0].value, nil
}

// GetString fetches an OCTET STRING / DisplayString OID.
func (c Client) GetString(oid string) (string, error) {
	_, v, err := c.Get(oid)
	return string(v), err
}

// Walk performs GETNEXT iteration under root (max 200 vars, loop-guarded).
func (c Client) Walk(root string) ([]respVB, error) {
	var out []respVB
	next := root
	for i := 0; i < 200; i++ {
		res, err := c.exchange(0xA1, []varbind{{oid: next}})
		if err != nil {
			return out, err
		}
		if len(res.vbs) == 0 {
			break
		}
		vb := res.vbs[0]
		if vb.typ == 0x82 || !strings.HasPrefix(vb.oid, strings.TrimRight(root, ".")+".") {
			break // endOfMibView or out of subtree
		}
		out = append(out, vb)
		next = vb.oid
	}
	return out, nil
}

// IfaceRow is one discovered interface row.
type IfaceRow struct {
	Index     int    `json:"index"`
	Descr     string `json:"descr"`
	OperUp    bool   `json:"oper_up"`
	InOctets  uint64 `json:"in_octets"`
	OutOctets uint64 `json:"out_octets"`
	SpeedBps  uint64 `json:"speed_bps"`
}

func u64(b []byte) uint64 {
	var v uint64
	for _, c := range b {
		v = v<<8 | uint64(c)
	}
	return v
}

// Interfaces walks the standard ifTable.
func (c Client) Interfaces() ([]IfaceRow, error) {
	byIdx := map[int]*IfaceRow{}
	collect := func(root string, fn func(r *IfaceRow, raw []byte)) error {
		vbs, err := c.Walk(root)
		if err != nil {
			return err
		}
		for _, vb := range vbs {
			idxStr := strings.TrimPrefix(vb.oid, strings.TrimRight(root, ".")+".")
			idx, err := strconv.Atoi(strings.Split(idxStr, ".")[0])
			if err != nil {
				continue
			}
			r := byIdx[idx]
			if r == nil {
				r = &IfaceRow{Index: idx}
				byIdx[idx] = r
			}
			fn(r, vb.value)
		}
		return nil
	}
	if err := collect(IfDescr, func(r *IfaceRow, raw []byte) { r.Descr = string(raw) }); err != nil {
		return nil, err
	}
	_ = collect(IfOper, func(r *IfaceRow, raw []byte) { r.OperUp = u64(raw) == 1 })
	_ = collect(IfInOctets, func(r *IfaceRow, raw []byte) { r.InOctets = u64(raw) })
	_ = collect(IfOutOctets, func(r *IfaceRow, raw []byte) { r.OutOctets = u64(raw) })
	_ = collect(IfSpeed, func(r *IfaceRow, raw []byte) { r.SpeedBps = u64(raw) })
	out := make([]IfaceRow, 0, len(byIdx))
	for _, r := range byIdx {
		out = append(out, *r)
	}
	return out, nil
}
