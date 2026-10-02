// Package syslog implements a UDP syslog receiver (RFC 3164) that parses
// priority/facility/severity and forwards messages to a handler (events bus).
package syslog

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"time"
)

var rfc3164 = regexp.MustCompile(`^<(\d+)>(\w{3}\s+\d+\s+\d+:\d+:\d+)\s+(\S+)\s+(\S+?:)?\s?(.*)$`)

type Message struct {
	Priority int       `json:"priority"`
	Facility int       `json:"facility"`
	Severity int       `json:"severity"`
	Host     string    `json:"host"`
	Tag      string    `json:"tag"`
	Body     string    `json:"body"`
	Received time.Time `json:"received_at"`
	RemoteIP string    `json:"remote_ip"`
}

func Parse(raw string) Message {
	m := Message{Body: raw, Received: time.Now().UTC()}
	if ms := rfc3164.FindStringSubmatch(raw); ms != nil {
		if p, err := strconv.Atoi(ms[1]); err == nil {
			m.Priority, m.Facility, m.Severity = p, p/8, p%8
		}
		m.Host, m.Tag, m.Body = ms[3], ms[4], ms[5]
	}
	return m
}

// Serve listens on addr and calls h for each datagram until ctx ends.
func Serve(ctx context.Context, addr string, h func(Message)) error {
	pc, err := net.ListenPacket("udp", addr)
	if err != nil {
		return fmt.Errorf("syslog listen %s: %w", addr, err)
	}
	defer pc.Close()
	go func() {
		<-ctx.Done()
		_ = pc.Close()
	}()
	buf := make([]byte, 8192)
	for {
		n, remote, err := pc.ReadFrom(buf)
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				continue
			}
		}
		m := Parse(string(buf[:n]))
		if host, _, err := net.SplitHostPort(remote.String()); err == nil {
			m.RemoteIP = host
		} else {
			m.RemoteIP = remote.String()
		}
		h(m)
	}
}
