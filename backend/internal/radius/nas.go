package radius

import (
	"sync"
	"time"
)

// NAS is one authorized network access server/client.
type NAS struct {
	ID        string `json:"id"`
	OrgID     string `json:"org_id"`
	Name      string `json:"name"`
	IP        string `json:"ip"`
	SecretRef string `json:"secret_ref"` // reference only — secret lives in the vault
	Vendor    string `json:"vendor,omitempty"`
	Enabled   bool   `json:"enabled"`
}

// NASStore is the in-memory NAS registry (DB-backed in production via store).
type NASStore struct {
	mu   sync.RWMutex
	byIP map[string]*NAS
}

func NewNASStore() *NASStore { return &NASStore{byIP: map[string]*NAS{}} }

func (n *NASStore) Upsert(nas *NAS) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.byIP[nas.IP] = nas
}

func (n *NASStore) Authorized(ip string) (*NAS, bool) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	nas, ok := n.byIP[ip]
	if !ok || !nas.Enabled {
		return nil, false
	}
	return nas, true
}

// --- Anti-replay / idempotency ---
// Retransmitted requests (same client IP + identifier + authenticator)
// return the cached response instead of executing twice. Entries expire.

type dedupKey struct {
	ip   string
	id   byte
	auth [16]byte
}

type dedupEntry struct {
	resp []byte
	exp  time.Time
}

// Dedup is safe for concurrent use and bounded by TTL expiry sweep.
type Dedup struct {
	mu  sync.Mutex
	m   map[dedupKey]dedupEntry
	ttl time.Duration
}

func NewDedup(ttl time.Duration) *Dedup {
	if ttl == 0 {
		ttl = 60 * time.Second
	}
	return &Dedup{m: map[dedupKey]dedupEntry{}, ttl: ttl}
}

func (d *Dedup) Get(ip string, id byte, auth [16]byte) ([]byte, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	e, ok := d.m[dedupKey{ip, id, auth}]
	if !ok || time.Now().After(e.exp) {
		delete(d.m, dedupKey{ip, id, auth})
		return nil, false
	}
	return e.resp, true
}

func (d *Dedup) Put(ip string, id byte, auth [16]byte, resp []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if len(d.m) > 10000 {
		now := time.Now()
		for k, e := range d.m {
			if now.After(e.exp) {
				delete(d.m, k)
			}
		}
	}
	d.m[dedupKey{ip, id, auth}] = dedupEntry{resp: append([]byte{}, resp...), exp: time.Now().Add(d.ttl)}
}

// --- Usage aggregation for billing ---
// Interim updates carry cumulative counters; on retransmit only forward
// progress counts (last-seen per session), so usage is never double-billed.

type Usage struct {
	Username  string    `json:"username"`
	Day       string    `json:"day"` // YYYY-MM-DD UTC
	InOctets  int64     `json:"in_octets"`
	OutOctets int64     `json:"out_octets"`
	Sessions  int       `json:"sessions"`
	UpdatedAt time.Time `json:"updated_at"`
}

type UsageStore struct {
	mu    sync.RWMutex
	usage map[string]*Usage   // username|day
	last  map[string][2]int64 // acctSessionID -> last (in,out)
	seen  map[string]bool     // acctSessionID counted
}

func NewUsageStore() *UsageStore {
	return &UsageStore{usage: map[string]*Usage{}, last: map[string][2]int64{}, seen: map[string]bool{}}
}

func dayKey(t time.Time) string { return t.UTC().Format("2006-01-02") }

// Interim records cumulative counters; only the forward delta is added.
func (u *UsageStore) Interim(username, sessionID string, inCum, outCum int64) {
	u.mu.Lock()
	defer u.mu.Unlock()
	prev := u.last[sessionID]
	u.last[sessionID] = [2]int64{inCum, outCum}
	di, do := inCum-prev[0], outCum-prev[1]
	if di < 0 {
		di = 0
	}
	if do < 0 {
		do = 0
	}
	k := username + "|" + dayKey(time.Now())
	us := u.usage[k]
	if us == nil {
		us = &Usage{Username: username, Day: dayKey(time.Now())}
		u.usage[k] = us
	}
	us.InOctets += di
	us.OutOctets += do
	if !u.seen[sessionID] {
		u.seen[sessionID] = true
		us.Sessions++
	}
	us.UpdatedAt = time.Now()
	delete(u.last, sessionID+"_stop")
}

func (u *UsageStore) Stop(sessionID string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	delete(u.last, sessionID)
	delete(u.seen, sessionID)
}

func (u *UsageStore) Get(username string, day time.Time) *Usage {
	u.mu.RLock()
	defer u.mu.RUnlock()
	if us, ok := u.usage[username+"|"+dayKey(day)]; ok {
		cp := *us
		return &cp
	}
	return &Usage{Username: username, Day: dayKey(day)}
}
