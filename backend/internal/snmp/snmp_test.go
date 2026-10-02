package snmp_test

import (
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/snmp"
)

type obj struct {
	oid string
	typ byte // 0x02 int, 0x04 string, 0x43 timeticks
	val []byte
	str string
	num uint64
}

func oidParts(s string) []int {
	var out []int
	for _, p := range strings.Split(strings.Trim(s, "."), ".") {
		n, _ := strconv.Atoi(p)
		out = append(out, n)
	}
	return out
}

func oidLess(a, b string) bool {
	pa, pb := oidParts(a), oidParts(b)
	for i := 0; i < len(pa) && i < len(pb); i++ {
		if pa[i] != pb[i] {
			return pa[i] < pb[i]
		}
	}
	return len(pa) < len(pb)
}

// Minimal BER request parser + responder for tests.
func serveFake(t *testing.T, conn *net.UDPConn, community string, db []obj) {
	t.Helper()
	sort.Slice(db, func(i, j int) bool { return oidLess(db[i].oid, db[j].oid) })
	buf := make([]byte, 65535)
	for {
		n, addr, err := conn.ReadFromUDP(buf)
		if err != nil {
			return
		}
		req := append([]byte{}, buf[:n]...)
		resp := handleReq(req, community, db)
		if resp != nil {
			_, _ = conn.WriteToUDP(resp, addr)
		}
	}
}

func rdLen(b []byte, p *int) int {
	n := int(b[*p])
	*p++
	if n < 0x80 {
		return n
	}
	c := int(n & 0x7F)
	v := 0
	for i := 0; i < c; i++ {
		v = v<<8 | int(b[*p])
		*p++
	}
	return v
}

func rdOID(b []byte, p *int) string {
	_ = b[*p]
	*p++
	l := rdLen(b, p)
	raw := b[*p : *p+l]
	*p += l
	out := []int{int(raw[0] / 40), int(raw[0] % 40)}
	v := 0
	for _, c := range raw[1:] {
		v = v<<7 | int(c&0x7F)
		if c&0x80 == 0 {
			out = append(out, v)
			v = 0
		}
	}
	var sb strings.Builder
	for i, x := range out {
		if i > 0 {
			sb.WriteByte('.')
		}
		sb.WriteString(strconv.Itoa(x))
	}
	return "." + sb.String()
}

func encVal(o obj) []byte {
	switch o.typ {
	case 0x04:
		b := []byte(o.str)
		return append([]byte{0x04, byte(len(b))}, b...)
	case 0x43:
		v := o.num
		raw := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
		i := 0
		for i < 3 && raw[i] == 0 {
			i++
		}
		return append([]byte{0x43, byte(4 - i)}, raw[i:]...)
	default:
		v := o.num
		raw := []byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}
		i := 0
		for i < 3 && raw[i] == 0 {
			i++
		}
		return append([]byte{0x02, byte(4 - i)}, raw[i:]...)
	}
}

func encRespOID(s string) []byte {
	nums := oidParts(s)
	body := []byte{byte(nums[0]*40 + nums[1])}
	for _, n := range nums[2:] {
		var st []byte
		st = append([]byte{byte(n & 0x7F)}, st...)
		n >>= 7
		for n > 0 {
			st = append([]byte{byte(n&0x7F | 0x80)}, st...)
			n >>= 7
		}
		for i := 0; i < len(st)-1; i++ {
			st[i] |= 0x80
		}
		body = append(body, st...)
	}
	return append([]byte{0x06, byte(len(body))}, body...)
}

func handleReq(req []byte, community string, db []obj) []byte {
	p := 0
	if req[p] != 0x30 {
		return nil
	}
	p++
	_ = rdLen(req, &p)
	if req[p] != 0x02 {
		return nil
	}
	p++
	l := rdLen(req, &p)
	p += l // version
	if req[p] != 0x04 {
		return nil
	}
	p++
	l = rdLen(req, &p)
	if string(req[p:p+l]) != community {
		return nil
	}
	p += l
	pdu := req[p]
	p++
	_ = rdLen(req, &p)
	if req[p] != 0x02 {
		return nil
	}
	p++
	l = rdLen(req, &p)
	reqID := append([]byte{}, req[p:p+l]...)
	p += l
	p += 2 + 1 + 2 + 1 // skip error-status, error-index (02 01 00 each)
	if req[p] != 0x30 {
		return nil
	}
	p++
	_ = rdLen(req, &p)
	if req[p] != 0x30 {
		return nil
	}
	p++
	_ = rdLen(req, &p)
	oid := rdOID(req, &p)

	mkResp := func(vbs []byte) []byte {
		body := append([]byte{0x02, byte(len(reqID))}, reqID...)
		body = append(body, 0x02, 0x01, 0x00, 0x02, 0x01, 0x00)
		body = append(body, 0x30, byte(len(vbs)))
		body = append(body, vbs...)
		pduBytes := append([]byte{0xA2, byte(len(body))}, body...)
		inner := append([]byte{0x02, 0x01, 0x01}, append([]byte{0x04, byte(len(community))}, []byte(community)...)...)
		inner = append(inner, pduBytes...)
		return append([]byte{0x30, byte(len(inner))}, inner...)
	}
	vb := func(o string, val []byte) []byte {
		inner := append(encRespOID(o), val...)
		return append([]byte{0x30, byte(len(inner))}, inner...)
	}

	switch pdu {
	case 0xA0: // GET
		for _, o := range db {
			if o.oid == oid {
				return mkResp(vb(o.oid, encVal(o)))
			}
		}
		// noSuchName
		body := append([]byte{0x02, byte(len(reqID))}, reqID...)
		body = append(body, 0x02, 0x01, 0x02, 0x02, 0x01, 0x00, 0x30, 0x00)
		pduBytes := append([]byte{0xA2, byte(len(body))}, body...)
		inner := append([]byte{0x02, 0x01, 0x01}, append([]byte{0x04, byte(len(community))}, []byte(community)...)...)
		inner = append(inner, pduBytes...)
		return append([]byte{0x30, byte(len(inner))}, inner...)
	case 0xA1: // GETNEXT
		for _, o := range db {
			if oidLess(oid, o.oid) {
				return mkResp(vb(o.oid, encVal(o)))
			}
		}
		return mkResp(vb(oid, []byte{0x82, 0x00})) // endOfMibView
	}
	return nil
}

func fakeAgent(t *testing.T) snmp.Client {
	t.Helper()
	db := []obj{
		{oid: ".1.3.6.1.2.1.1.1.0", typ: 0x04, str: "MikroTik RouterOS CHR"},
		{oid: ".1.3.6.1.2.1.1.3.0", typ: 0x43, num: 99999},
		{oid: ".1.3.6.1.2.1.2.2.1.2.1", typ: 0x04, str: "ether1"},
		{oid: ".1.3.6.1.2.1.2.2.1.2.2", typ: 0x04, str: "ether2"},
		{oid: ".1.3.6.1.2.1.2.2.1.8.1", typ: 0x02, num: 1},
		{oid: ".1.3.6.1.2.1.2.2.1.8.2", typ: 0x02, num: 2},
		{oid: ".1.3.6.1.2.1.2.2.1.10.1", typ: 0x02, num: 1000},
		{oid: ".1.3.6.1.2.1.2.2.1.16.1", typ: 0x02, num: 2000},
	}
	addr, _ := net.ResolveUDPAddr("udp", "127.0.0.1:0")
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		t.Skip("no udp")
	}
	t.Cleanup(func() { conn.Close() })
	go serveFake(t, conn, "public", db)
	return snmp.Client{Target: conn.LocalAddr().String(), Community: "public", Timeout: 2 * time.Second}
}

func TestGetAndWalk(t *testing.T) {
	c := fakeAgent(t)
	descr, err := c.GetString(snmp.SysDescr)
	if err != nil || descr != "MikroTik RouterOS CHR" {
		t.Fatalf("descr=%q err=%v", descr, err)
	}
	ifaces, err := c.Interfaces()
	if err != nil || len(ifaces) != 2 {
		t.Fatalf("ifaces=%v err=%v", ifaces, err)
	}
	byName := map[string]bool{}
	for _, r := range ifaces {
		byName[r.Descr] = r.OperUp
	}
	if !byName["ether1"] || byName["ether2"] {
		t.Fatalf("rows=%+v", ifaces)
	}
	for _, r := range ifaces {
		if r.Descr == "ether1" && r.InOctets != 1000 {
			t.Fatalf("row ether1=%+v", r)
		}
	}
	if p := snmp.ProfileFor(descr); p.Vendor != "MikroTik" {
		t.Fatalf("profile=%+v", p)
	}
}

// TestE2E polls a REAL agent when SNMP_E2E=1 (SNMP_TARGET, SNMP_COMMUNITY).
func TestE2E(t *testing.T) {
	if os.Getenv("SNMP_E2E") != "1" {
		t.Skip("SNMP_E2E not set")
	}
	c := snmp.Client{Target: os.Getenv("SNMP_TARGET"), Community: os.Getenv("SNMP_COMMUNITY"), Timeout: 5 * time.Second}
	d, err := c.GetString(snmp.SysDescr)
	if err != nil {
		t.Fatalf("E2E failed: %v", err)
	}
	t.Logf("sysDescr=%s profile=%+v", d, snmp.ProfileFor(d))
}
