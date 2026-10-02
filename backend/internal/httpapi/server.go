package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/universal-isp/platform/internal/auth"
	"github.com/universal-isp/platform/internal/connectors/lab"
	"github.com/universal-isp/platform/internal/connectors/registry"
	"github.com/universal-isp/platform/internal/connectors/sdk"
	"github.com/universal-isp/platform/internal/health"
	"github.com/universal-isp/platform/internal/middleware"
	"github.com/universal-isp/platform/internal/rbac"
	"log/slog"
	"time"
)

type Server struct {
	mux       *http.ServeMux
	log       *slog.Logger
	jwtSecret string
	health    *health.Checker
}

func New(log *slog.Logger, jwtSecret string, h *health.Checker) *Server {
	s := &Server{mux: http.NewServeMux(), log: log, jwtSecret: jwtSecret, health: h}
	h.Register(s.mux)
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return middleware.Chain(s.mux,
		middleware.Logging(s.log),
		middleware.RateLimit(300, time.Minute),
		middleware.JWT(s.jwtSecret),
	)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/v1/auth/login", s.handleLogin)
	s.mux.HandleFunc("/api/v1/connectors", s.handleConnectors)
	s.mux.HandleFunc("/api/v1/noc/summary", s.handleNOC)
	s.mux.HandleFunc("/api/v1/vendor-matrix", s.handleMatrix)
	// CRUD stubs with RBAC (backed by DB in production; in-memory shape here)
	s.mux.Handle("/api/v1/customers", middleware.Require(rbac.CustomersRead)(http.HandlerFunc(s.handleStub("customers"))))
	s.mux.Handle("/api/v1/subscriptions", middleware.Require(rbac.CustomersRead)(http.HandlerFunc(s.handleStub("subscriptions"))))
	s.mux.Handle("/api/v1/packages", middleware.Require(rbac.CustomersRead)(http.HandlerFunc(s.handleStub("packages"))))
	s.mux.Handle("/api/v1/invoices", middleware.Require(rbac.BillingRead)(http.HandlerFunc(s.handleStub("invoices"))))
	s.mux.Handle("/api/v1/payments", middleware.Require(rbac.BillingRead)(http.HandlerFunc(s.handleStub("payments"))))
	s.mux.Handle("/api/v1/devices", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleStub("devices"))))
	s.mux.Handle("/api/v1/provisioning/jobs", middleware.Require(rbac.ProvisionExec)(http.HandlerFunc(s.handleStub("provisioning-jobs"))))
	s.mux.Handle("/api/v1/monitoring/devices", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleStub("monitoring"))))
	s.mux.Handle("/api/v1/alerts", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleStub("alerts"))))
	s.mux.Handle("/api/v1/events", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleStub("events"))))
	s.mux.Handle("/api/v1/reports/summary", middleware.Require(rbac.BillingRead)(http.HandlerFunc(s.handleStub("reports"))))
	s.mux.Handle("/api/v1/system/settings", middleware.Require(rbac.SystemAdmin)(http.HandlerFunc(s.handleStub("settings"))))
	s.mux.Handle("/api/v1/lab/test", middleware.Require(rbac.NetworkWrite)(http.HandlerFunc(s.handleLabTest)))
	s.mux.Handle("/api/v1/topology", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleStub("topology"))))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body.Username == "" || body.Password == "" {
		writeJSON(w, 400, map[string]string{"error": "username and password required"})
		return
	}
	// Demo auth: admin/secret. Production verifies bcrypt hash in users table + login throttling.
	if !(body.Username == "admin" && body.Password == "secret") {
		writeJSON(w, 401, map[string]string{"error": "invalid credentials"})
		return
	}
	tok, _ := auth.Sign(s.jwtSecret, body.Username, "demo-isp", []string{"superadmin"}, 12*time.Hour)
	writeJSON(w, 200, map[string]any{"token": tok, "user": body.Username})
}

func (s *Server) handleConnectors(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"data": registry.List()})
}

func (s *Server) handleMatrix(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"data": registry.List()})
}

func (s *Server) handleLabTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Vendor         string            `json:"vendor"`
		Family         string            `json:"family"`
		ConnectionType string            `json:"connection_type"`
		Config         map[string]string `json:"config"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if body.Vendor == "" || body.Family == "" || body.ConnectionType == "" {
		writeJSON(w, 400, map[string]string{"error": "vendor, family, connection_type required"})
		return
	}
	c, err := registry.Create(body.Vendor, body.Family, sdk.ConnectionType(body.ConnectionType), body.Config)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": err.Error()})
		return
	}
	// NOTE: credentials in body are used for this test only, never logged/stored.
	res, err := lab.TestAndDiscover(r.Context(), c)
	if err != nil {
		writeJSON(w, 200, map[string]any{"healthy": false, "error": err.Error(), "result": res})
		return
	}
	writeJSON(w, 200, map[string]any{"healthy": true, "result": res})
}

func (s *Server) handleNOC(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{
		"total_devices": 12, "online_devices": 10, "offline_devices": 1, "degraded_devices": 1,
		"active_subscribers": 1240, "suspended_subscribers": 37, "online_sessions": 986,
		"active_alerts": 4, "critical_alerts": 1, "provisioning_failures_24h": 2,
		"updated_at": time.Now().UTC(),
	})
}

func (s *Server) handleStub(resource string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"data": []any{}, "resource": resource, "page": 1, "per_page": 25, "total": 0})
	}
}
