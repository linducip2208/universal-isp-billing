package radius

import (
	"crypto/md5"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

// SendCoA sends a CoA/Disconnect request and waits for ACK (RFC 5176).
func SendCoA(nasAddr, secret string, code byte, identifier byte, attrs []Attr, timeout time.Duration) error {
	ua, err := net.ResolveUDPAddr("udp", nasAddr)
	if err != nil {
		return err
	}
	conn, err := net.DialUDP("udp", nil, ua)
	if err != nil {
		return err
	}
	defer conn.Close()
	// Request authenticator = MD5(code+id+len+16zero+attrs+secret)
	body := []byte{}
	for _, a := range attrs {
		body = append(body, a.Type, byte(len(a.Value)+2))
		body = append(body, a.Value...)
	}
	pkt := make([]byte, 20+len(body))
	pkt[0] = code
	pkt[1] = identifier
	binary.BigEndian.PutUint16(pkt[2:4], uint16(len(pkt)))
	copy(pkt[20:], body)
	h := md5.New()
	h.Write(pkt)
	h.Write([]byte(secret))
	copy(pkt[4:20], h.Sum(nil))
	_ = conn.SetDeadline(time.Now().Add(timeout))
	if _, err := conn.Write(pkt); err != nil {
		return err
	}
	resp := make([]byte, 4096)
	n, err := conn.Read(resp)
	if err != nil {
		return fmt.Errorf("no CoA response: %w", err)
	}
	if n < 2 {
		return fmt.Errorf("short CoA response")
	}
	switch resp[0] {
	case CodeCoAAck, CodeDisconnectAck:
		return nil
	default:
		return fmt.Errorf("CoA rejected (code=%d)", resp[0])
	}
}
