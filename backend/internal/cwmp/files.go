package cwmp

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// FileStore stages firmware/config blobs for Download RPCs.
// Production backs this with object storage; the interface is stable.
type FileStore struct {
	mu    sync.RWMutex
	blobs map[string][]byte
}

func NewFileStore() *FileStore { return &FileStore{blobs: map[string][]byte{}} }

func (f *FileStore) Put(id string, data []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.blobs[id] = append([]byte{}, data...)
}

func (f *FileStore) Get(id string) ([]byte, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	b, ok := f.blobs[id]
	return b, ok
}

// FileHandler serves GET /files/{id} (CPE download) and PUT /files/{id}
// (operator upload, authenticated). Mount where the Download URLs point.
func (s *Server) FileHandler(store *FileStore) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/files/")
		if id == "" || strings.Contains(id, "/") {
			http.Error(w, "bad file id", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodGet:
			b, ok := store.Get(id)
			if !ok {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(b)
		case http.MethodPut, http.MethodPost:
			if !CheckAuth(s.Users, r) {
				Challenge(w, "acs-files")
				return
			}
			data, err := io.ReadAll(io.LimitReader(r.Body, 256<<20))
			if err != nil {
				http.Error(w, "bad body", http.StatusBadRequest)
				return
			}
			store.Put(id, data)
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(fmt.Sprintf(`{"id":%q,"size":%d}`, id, len(data))))
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// --- Retry storm guard ---
// A misconfigured CPE fleet can hammer Inform. Track per-serial arrival
// times; reporters (NOC) list suspects, and the server sheds load by ending
// sessions early (204) while storming.

const (
	stormWindow = time.Minute
	stormMax    = 20
)

func (s *Server) noteInform(serial string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	key := "storm:" + serial
	ts := s.storm[key]
	var keep []time.Time
	for _, t := range ts {
		if now.Sub(t) < stormWindow {
			keep = append(keep, t)
		}
	}
	keep = append(keep, now)
	s.storm[key] = keep
	return len(keep) > stormMax
}

// StormSuspects lists serials currently over the Inform rate threshold.
func (s *Server) StormSuspects() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now()
	var out []string
	for k, ts := range s.storm {
		var keep []time.Time
		for _, t := range ts {
			if now.Sub(t) < stormWindow {
				keep = append(keep, t)
			}
		}
		if len(keep) > stormMax {
			out = append(out, strings.TrimPrefix(k, "storm:"))
		}
	}
	return out
}

var _ = bytes.MinRead
