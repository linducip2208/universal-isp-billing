package health

import (
	"database/sql"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/universal-isp/platform/internal/metrics"
)

type Checker struct {
	DB        *sql.DB
	RedisAddr string
	Metrics   *metrics.Registry
}

func (c *Checker) liveness(w http.ResponseWriter, _ *http.Request) {
	write(w, map[string]any{"status": "alive", "time": time.Now().UTC()})
}

func (c *Checker) readiness(w http.ResponseWriter, r *http.Request) {
	checks := map[string]string{"postgres": "unknown", "redis": "unknown"}
	if c.DB != nil {
		if err := c.DB.PingContext(r.Context()); err != nil {
			checks["postgres"] = "down: " + err.Error()
		} else {
			checks["postgres"] = "up"
		}
	} else {
		checks["postgres"] = "not-configured"
	}
	conn, err := net.DialTimeout("tcp", c.RedisAddr, 800*time.Millisecond)
	if err != nil {
		checks["redis"] = "down"
	} else {
		_ = conn.Close()
		checks["redis"] = "up"
	}
	ready := checks["postgres"] == "up" || checks["postgres"] == "not-configured"
	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	w.WriteHeader(status)
	write(w, map[string]any{"ready": ready, "checks": checks})
}

func (c *Checker) metrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	if c.Metrics == nil {
		_, _ = w.Write([]byte("# no metrics registry wired\n"))
		return
	}
	_, _ = w.Write([]byte(c.Metrics.Exposition()))
}

func (c *Checker) Register(mux *http.ServeMux) {
	mux.HandleFunc("/live", c.liveness)
	mux.HandleFunc("/health", c.liveness)
	mux.HandleFunc("/ready", c.readiness)
	mux.HandleFunc("/metrics", c.metrics)
}

func write(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
