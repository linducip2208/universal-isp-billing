// Package cache: Cache interface with in-memory implementation and a
// minimal stdlib-only Redis client (RESP2 SETEX/GET/DEL/PING). Production
// uses Redis; memory is for dev/test.
package cache

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

type Memory struct {
	mu   sync.RWMutex
	vals map[string]string
	exp  map[string]time.Time
}

func NewMemory() *Memory { return &Memory{vals: map[string]string{}, exp: map[string]time.Time{}} }

func (m *Memory) Get(_ context.Context, key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.vals[key]
	if !ok {
		return "", false
	}
	if exp, has := m.exp[key]; has && time.Now().After(exp) {
		return "", false
	}
	return v, true
}

func (m *Memory) Set(_ context.Context, key, value string, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.vals[key] = value
	if ttl > 0 {
		m.exp[key] = time.Now().Add(ttl)
	} else {
		delete(m.exp, key)
	}
	return nil
}

func (m *Memory) Del(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.vals, key)
	delete(m.exp, key)
	return nil
}

// Redis is a tiny RESP2 client (no external deps).
type Redis struct {
	Addr string
	pool chan net.Conn
}

func NewRedis(addr string) *Redis { return &Redis{Addr: addr, pool: make(chan net.Conn, 8)} }

func (r *Redis) conn(ctx context.Context) (net.Conn, error) {
	select {
	case c := <-r.pool:
		return c, nil
	default:
		d := net.Dialer{}
		return d.DialContext(ctx, "tcp", r.Addr)
	}
}

func (r *Redis) do(ctx context.Context, args ...string) (string, error) {
	c, err := r.conn(ctx)
	if err != nil {
		return "", err
	}
	defer func() {
		select {
		case r.pool <- c:
		default:
			_ = c.Close()
		}
	}()
	var sb strings.Builder
	fmt.Fprintf(&sb, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(&sb, "$%d\r\n%s\r\n", len(a), a)
	}
	_ = c.SetDeadline(time.Now().Add(3 * time.Second))
	if _, err := c.Write([]byte(sb.String())); err != nil {
		_ = c.Close()
		return "", err
	}
	rd := bufio.NewReader(c)
	line, err := rd.ReadString('\n')
	if err != nil {
		return "", err
	}
	switch {
	case strings.HasPrefix(line, "+"):
		return strings.TrimSpace(line[1:]), nil
	case strings.HasPrefix(line, "$"):
		var n int
		fmt.Sscanf(line, "$%d", &n)
		if n < 0 {
			return "", nil // nil bulk
		}
		buf := make([]byte, n+2)
		for i := 0; i < n+2; i++ {
			buf[i], _ = rd.ReadByte()
		}
		return string(buf[:n]), nil
	case strings.HasPrefix(line, "-"):
		return "", fmt.Errorf("redis: %s", strings.TrimSpace(line[1:]))
	default:
		return strings.TrimSpace(line), nil
	}
}

func (r *Redis) Get(ctx context.Context, key string) (string, bool) {
	v, err := r.do(ctx, "GET", key)
	if err != nil || v == "" {
		return "", false
	}
	return v, true
}

func (r *Redis) Set(ctx context.Context, key, value string, ttl time.Duration) error {
	if ttl > 0 {
		_, err := r.do(ctx, "SETEX", key, fmt.Sprint(int(ttl.Seconds())), value)
		return err
	}
	_, err := r.do(ctx, "SET", key, value)
	return err
}

func (r *Redis) Del(ctx context.Context, key string) error {
	_, err := r.do(ctx, "DEL", key)
	return err
}
