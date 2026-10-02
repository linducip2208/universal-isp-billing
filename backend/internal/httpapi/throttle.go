package httpapi

import (
	"sync"
	"time"
)

// loginThrottle: max 5 attempts per IP per minute, then 429.
type loginThrottle struct {
	mu   sync.Mutex
	hits map[string][]time.Time
}

func newLoginThrottle() *loginThrottle { return &loginThrottle{hits: map[string][]time.Time{}} }

func (l *loginThrottle) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	var keep []time.Time
	for _, t := range l.hits[ip] {
		if now.Sub(t) < time.Minute {
			keep = append(keep, t)
		}
	}
	if len(keep) >= 5 {
		l.hits[ip] = keep
		return false
	}
	l.hits[ip] = append(keep, now)
	return true
}
