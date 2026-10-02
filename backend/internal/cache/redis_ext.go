package cache

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

// Full RESP2 client: Do sends a command and parses any reply type
// (simple string, error, integer, bulk, array — recursive).

func (r *Redis) Do(ctx context.Context, args ...string) (any, error) {
	c, err := r.conn(ctx)
	if err != nil {
		return nil, err
	}
	release := true
	defer func() {
		if release {
			select {
			case r.pool <- c:
			default:
				_ = c.Close()
			}
		} else {
			_ = c.Close()
		}
	}()
	var sb strings.Builder
	fmt.Fprintf(&sb, "*%d\r\n", len(args))
	for _, a := range args {
		fmt.Fprintf(&sb, "$%d\r\n%s\r\n", len(a), a)
	}
	_ = c.SetDeadline(time.Now().Add(10 * time.Second))
	if _, err := c.Write([]byte(sb.String())); err != nil {
		release = false
		return nil, err
	}
	v, err := parseReply(bufio.NewReader(c))
	if err != nil {
		release = false
		return nil, err
	}
	if e, ok := v.(redisError); ok {
		return nil, errors.New("redis: " + string(e))
	}
	return v, nil
}

type redisError string

func parseReply(r *bufio.Reader) (any, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 3 {
		return nil, fmt.Errorf("short reply %q", line)
	}
	switch line[0] {
	case '+':
		return strings.TrimRight(line[1:], "\r\n"), nil
	case '-':
		return redisError(strings.TrimRight(line[1:], "\r\n")), nil
	case ':':
		return strconv.ParseInt(strings.TrimRight(line[1:], "\r\n"), 10, 64)
	case '$':
		n, _ := strconv.Atoi(strings.TrimRight(line[1:], "\r\n"))
		if n < 0 {
			return nil, nil
		}
		buf := make([]byte, n+2)
		for i := range buf {
			buf[i], _ = r.ReadByte()
		}
		return string(buf[:n]), nil
	case '*':
		n, _ := strconv.Atoi(strings.TrimRight(line[1:], "\r\n"))
		if n < 0 {
			return nil, nil
		}
		out := make([]any, 0, n)
		for i := 0; i < n; i++ {
			v, err := parseReply(r)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	}
	return nil, fmt.Errorf("unknown reply %q", line)
}

func dialRaw(ctx context.Context, addr string) (net.Conn, error) {
	d := net.Dialer{}
	return d.DialContext(ctx, "tcp", addr)
}

var _ = dialRaw

// --- Distributed lock (SET NX PX + Lua compare-del) ---

type Lock struct {
	c     *Redis
	key   string
	token string
	ttl   time.Duration
}

func newToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

// AcquireLock takes a distributed lock; ok=false when held elsewhere.
func (r *Redis) AcquireLock(ctx context.Context, key string, ttl time.Duration) (*Lock, bool, error) {
	token := newToken()
	ms := int(ttl.Milliseconds())
	v, err := r.Do(ctx, "SET", key, token, "NX", "PX", strconv.Itoa(ms))
	if err != nil {
		if strings.Contains(err.Error(), "nil") || strings.Contains(strings.ToLower(err.Error()), "busy") {
			return nil, false, nil
		}
		// SET NX miss returns nil bulk (no error) -> v == nil
		return nil, false, nil
	}
	if v == nil {
		return nil, false, nil
	}
	if s, ok := v.(string); !ok || s != "OK" {
		return nil, false, nil
	}
	return &Lock{c: r, key: key, token: token, ttl: ttl}, true, nil
}

var luaRelease = `if redis.call("get",KEYS[1]) == ARGV[1] then return redis.call("del",KEYS[1]) else return 0 end`
var luaRefresh = `if redis.call("get",KEYS[1]) == ARGV[1] then return redis.call("pexpire",KEYS[1],ARGV[2]) else return 0 end`

func (l *Lock) Release(ctx context.Context) error {
	v, err := l.c.Do(ctx, "EVAL", luaRelease, "1", l.key, l.token)
	if err != nil {
		return err
	}
	if n, ok := v.(int64); !ok || n != 1 {
		return errors.New("lock not owned")
	}
	return nil
}

func (l *Lock) Refresh(ctx context.Context) error {
	v, err := l.c.Do(ctx, "EVAL", luaRefresh, "1", l.key, l.token, strconv.Itoa(int(l.ttl.Milliseconds())))
	if err != nil {
		return err
	}
	if n, ok := v.(int64); !ok || n != 1 {
		return errors.New("lock not owned")
	}
	return nil
}

// --- Streams transport (multi-worker job fan-out) ---

type StreamMsg struct {
	ID     string
	Fields map[string]string
}

type Stream struct {
	c        *Redis
	Name     string
	Group    string
	Consumer string
}

func NewStream(c *Redis, name, group, consumer string) *Stream {
	return &Stream{c: c, Name: name, Group: group, Consumer: consumer}
}

// EnsureGroup creates the consumer group (idempotent).
func (s *Stream) EnsureGroup(ctx context.Context) error {
	_, err := s.c.Do(ctx, "XGROUP", "CREATE", s.Name, s.Group, "$", "MKSTREAM")
	if err != nil && strings.Contains(err.Error(), "BUSYGROUP") {
		return nil
	}
	return err
}

var ErrDuplicate = errors.New("duplicate stream message (idempotency key seen)")

// Publish appends fields; idemKey dedups via SET NX (returns ErrDuplicate).
func (s *Stream) Publish(ctx context.Context, idemKey string, fields map[string]string) (string, error) {
	if idemKey != "" {
		_, ok, err := s.c.AcquireLock(ctx, "stream:idem:"+s.Name+":"+idemKey, time.Hour)
		if err != nil {
			return "", err
		}
		if !ok {
			return "", ErrDuplicate
		}
	}
	args := []string{"XADD", s.Name, "MAXLEN", "~", "100000", "*"}
	for k, v := range fields {
		args = append(args, k, v)
	}
	v, err := s.c.Do(ctx, args...)
	if err != nil {
		return "", err
	}
	id, _ := v.(string)
	return id, nil
}

// Read fetches up to count pending/new messages (BLOCK up to block).
func (s *Stream) Read(ctx context.Context, count int, block time.Duration) ([]StreamMsg, error) {
	v, err := s.c.Do(ctx, "XREADGROUP", "GROUP", s.Group, s.Consumer,
		"COUNT", strconv.Itoa(count), "BLOCK", strconv.Itoa(int(block.Milliseconds())),
		"STREAMS", s.Name, ">")
	if err != nil {
		return nil, err
	}
	if v == nil {
		return nil, nil
	}
	var out []StreamMsg
	arr, _ := v.([]any)
	for _, stream := range arr {
		pair, _ := stream.([]any)
		if len(pair) < 2 {
			continue
		}
		msgs, _ := pair[1].([]any)
		for _, m := range msgs {
			mp, _ := m.([]any)
			if len(mp) < 2 {
				continue
			}
			id, _ := mp[0].(string)
			fv, _ := mp[1].([]any)
			fields := map[string]string{}
			for i := 0; i+1 < len(fv); i += 2 {
				ks, _ := fv[i].(string)
				vs, _ := fv[i+1].(string)
				fields[ks] = vs
			}
			out = append(out, StreamMsg{ID: id, Fields: fields})
		}
	}
	return out, nil
}

func (s *Stream) Ack(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := s.c.Do(ctx, append([]string{"XACK", s.Name, s.Group}, ids...)...)
	return err
}
