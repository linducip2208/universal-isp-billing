package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/universal-isp/platform/internal/auth"
	"github.com/universal-isp/platform/internal/bruteforce"
	"github.com/universal-isp/platform/internal/cache"
	"github.com/universal-isp/platform/internal/connectors/lab"
	"github.com/universal-isp/platform/internal/connectors/registry"
	"github.com/universal-isp/platform/internal/connectors/sdk"
	"github.com/universal-isp/platform/internal/health"
	"github.com/universal-isp/platform/internal/metrics"
	"github.com/universal-isp/platform/internal/middleware"
	"github.com/universal-isp/platform/internal/rbac"
	"github.com/universal-isp/platform/internal/store"
	"log/slog"
	"time"
)

type Server struct {
	mux         *http.ServeMux
	log         *slog.Logger
	jwtSecret   string
	health      *health.Checker
	store       *store.Store
	revoker     auth.Revoker
	bf          bruteforce.Tracker
	glimit      middleware.DoFunc
	meter       *metrics.Registry
	demoLogin   bool
	corsOrigins []string
	throttle    *loginThrottle
}

func New(log *slog.Logger, jwtSecret string, h *health.Checker) *Server {
	s := &Server{mux: http.NewServeMux(), log: log, jwtSecret: jwtSecret, health: h, throttle: newLoginThrottle(), revoker: auth.NewMemoryRevoker(), bf: bruteforce.NewMemory(10, 15*time.Minute), meter: metrics.New()}
	s.meter.Help("http_requests", "API requests by method/route/status")
	s.meter.Help("http_latency_ms", "API latency milliseconds by method/route")
	h.Metrics = s.meter
	h.Register(s.mux)
	s.routes()
	return s
}

// Metrics exposes the registry for /metrics (wired into health.Checker).
func (s *Server) Metrics() *metrics.Registry { return s.meter }

func (s *Server) WithStore(st *store.Store) *Server { s.store = st; return s }
func (s *Server) WithDemoLogin(on bool) *Server     { s.demoLogin = on; return s }
func (s *Server) WithCORS(origins []string) *Server { s.corsOrigins = origins; return s }

// WithRedis upgrades revocation + brute-force tracking to cross-instance
// Redis state (falls back gracefully per-call if Redis is unreachable).
func (s *Server) WithRedis(r *cache.Redis) *Server {
	s.revoker = &auth.RedisRevoker{Do: r.Do}
	s.bf = &bruteforce.Redis{Do: r.Do, Max: 10, Window: 15 * time.Minute, Prefix: "isp:loginfail:"}
	s.glimit = r.Do
	return s
}

func (s *Server) Handler() http.Handler {
	return middleware.Chain(s.mux,
		middleware.RequestID,
		middleware.Metrics(s.meter),
		middleware.CORS(s.corsOrigins),
		middleware.Logging(s.log),
		middleware.RateLimit(300, time.Minute),
		middleware.GlobalLimit(s.glimit, 1200, time.Minute, "isp:rl:"),
		middleware.PerOrgRateLimit(600, time.Minute),
		middleware.JWT(s.jwtSecret, s.revoker),
	)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/v1/auth/login", s.handleLogin)
	s.mux.HandleFunc("/api/v1/auth/mfa/verify", s.handleMFAVerify)
	s.mux.HandleFunc("/api/v1/auth/logout", s.handleLogout)
	s.mux.HandleFunc("/api/v1/connectors", s.handleConnectors)
	s.mux.HandleFunc("/api/v1/noc/summary", s.handleNOC)
	s.mux.HandleFunc("/api/v1/vendor-matrix", s.handleMatrix)
	// Resource lists are DB-backed and tenant-scoped. Without a database the
	// API answers 503 explicitly — it never returns fabricated rows.
	s.mux.Handle("/api/v1/customers", middleware.Require(rbac.CustomersRead)(http.HandlerFunc(s.handleResource("customers"))))
	s.mux.Handle("/api/v1/subscriptions", middleware.Require(rbac.CustomersRead)(http.HandlerFunc(s.handleResource("subscriptions"))))
	s.mux.Handle("/api/v1/packages", middleware.Require(rbac.CustomersRead)(http.HandlerFunc(s.handleResource("packages"))))
	s.mux.Handle("/api/v1/invoices", middleware.Require(rbac.BillingRead)(http.HandlerFunc(s.handleResource("invoices"))))
	s.mux.Handle("/api/v1/payments", middleware.Require(rbac.BillingRead)(http.HandlerFunc(s.handleResource("payments"))))
	s.mux.Handle("/api/v1/devices", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleResource("devices"))))
	s.mux.Handle("/api/v1/sites", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleResource("sites"))))
	s.mux.Handle("/api/v1/provisioning/jobs", middleware.Require(rbac.ProvisionExec)(http.HandlerFunc(s.handleResource("network_jobs"))))
	s.mux.Handle("/api/v1/alerts", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleResource("alerts"))))
	s.mux.Handle("/api/v1/events", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleResource("events"))))
	s.mux.Handle("/api/v1/audit", middleware.Require(rbac.SystemAdmin)(http.HandlerFunc(s.handleResource("audit_logs"))))
	s.mux.Handle("/api/v1/reports/summary", middleware.Require(rbac.BillingRead)(http.HandlerFunc(s.handleNOC)))
	s.mux.Handle("/api/v1/system/settings", middleware.Require(rbac.SystemAdmin)(http.HandlerFunc(s.handleSettings)))
	s.mux.Handle("/api/v1/lab/test", middleware.Require(rbac.NetworkWrite)(http.HandlerFunc(s.handleLabTest)))
	s.mux.Handle("/api/v1/topology", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleTopology)))
	s.mux.Handle("/api/v1/incidents", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleResource("incidents"))))
	s.mux.Handle("/api/v1/tickets", middleware.Require(rbac.CustomersRead)(http.HandlerFunc(s.handleResource("tickets"))))
	s.mux.Handle("/api/v1/work-orders", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleResource("work_orders"))))
	s.mux.Handle("/api/v1/contracts", middleware.Require(rbac.BillingRead)(http.HandlerFunc(s.handleResource("contracts"))))
	s.mux.Handle("/api/v1/config/snapshots", middleware.Require(rbac.NetworkWrite)(http.HandlerFunc(s.handleResource("config_snapshots"))))
	s.mux.Handle("/api/v1/config/changes", middleware.Require(rbac.NetworkWrite)(http.HandlerFunc(s.handleResource("changes"))))
	s.mux.Handle("/api/v1/radius/sessions", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleRadiusSessions)))
	s.mux.Handle("/api/v1/economics/summary", middleware.Require(rbac.BillingRead)(http.HandlerFunc(s.handleEconomics)))
	s.mux.Handle("/api/v1/service-health", middleware.Require(rbac.CustomersRead)(http.HandlerFunc(s.handleServiceHealth)))
	s.mux.Handle("/api/v1/copilot/ask", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleCopilot)))
	s.mux.Handle("/api/v1/customer-360", middleware.Require(rbac.CustomersRead)(http.HandlerFunc(s.handleCustomer360)))
	s.mux.Handle("/api/v1/incidents/action", middleware.Require(rbac.NetworkWrite)(http.HandlerFunc(s.handleIncidentAction)))
	s.mux.Handle("/api/v1/lab/runs", middleware.Require(rbac.NetworkRead)(http.HandlerFunc(s.handleResource("lab_runs"))))
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !s.throttle.allow(r.RemoteAddr) {
		middleware.Error(w, http.StatusTooManyRequests, "too many login attempts", r.Header.Get("X-Request-ID"))
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
	bfKey := "login:" + r.RemoteAddr + ":" + body.Username
	fail := func() {
		if _, blocked := s.bf.Fail(bfKey); blocked {
			middleware.Error(w, http.StatusTooManyRequests, "account temporarily locked (too many failures)", r.Header.Get("X-Request-ID"))
			return
		}
		writeJSON(w, 401, map[string]string{"error": "invalid credentials"})
	}
	// Production path: verify against users table (PBKDF2 hash + roles).
	if s.store != nil {
		_, org, roles, rotate, mfaRequired, err := s.store.Authenticate(r.Context(), body.Username, body.Password)
		if err != nil {
			fail()
			return
		}
		s.bf.Reset(bfKey)
		if len(roles) == 0 {
			roles = []string{"viewer"}
		}
		if mfaRequired {
			challenge, _ := auth.SignScoped(s.jwtSecret, body.Username, org, roles, "mfa-pending", 5*time.Minute)
			writeJSON(w, 202, map[string]any{"mfa_required": true, "challenge": challenge})
			return
		}
		tok, _ := auth.Sign(s.jwtSecret, body.Username, org, roles, 12*time.Hour)
		writeJSON(w, 200, map[string]any{"token": tok, "user": body.Username, "must_rotate_password": rotate})
		return
	}
	// Demo path: ONLY when explicitly enabled (ISP_DEMO_LOGIN=1). Refused otherwise.
	if !s.demoLogin || !(body.Username == "admin" && body.Password == "secret") {
		fail()
		return
	}
	s.bf.Reset(bfKey)
	s.log.Warn("demo login used — enable only for local development")
	tok, _ := auth.Sign(s.jwtSecret, body.Username, "demo-isp", []string{"superadmin"}, 12*time.Hour)
	writeJSON(w, 200, map[string]any{"token": tok, "user": body.Username})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cl := auth.ClaimsOf(r.Context())
	if cl == nil {
		writeJSON(w, 401, map[string]string{"error": "invalid token"})
		return
	}
	ttl := time.Until(time.Unix(cl.ExpiresAt, 0))
	if ttl < 0 {
		ttl = 0
	}
	s.revoker.Revoke(cl.ID, ttl)
	writeJSON(w, 200, map[string]string{"status": "logged out"})
}

// handleMFAVerify exchanges a 5-minute mfa-pending challenge + TOTP code
// for a full session token. The challenge authorizes ONLY this endpoint.
func (s *Server) handleMFAVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Challenge string `json:"challenge"`
		Code      string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Challenge == "" || body.Code == "" {
		writeJSON(w, 400, map[string]string{"error": "challenge and code required"})
		return
	}
	cl, err := auth.Verify(s.jwtSecret, body.Challenge)
	if err != nil || cl.Scope != "mfa-pending" {
		writeJSON(w, 401, map[string]string{"error": "invalid challenge"})
		return
	}
	if s.store == nil {
		writeJSON(w, 503, map[string]string{"error": "mfa requires database"})
		return
	}
	if err := s.store.VerifyTOTP(r.Context(), cl.Subject, body.Code); err != nil {
		if _, blocked := s.bf.Fail("mfa:" + r.RemoteAddr + ":" + cl.Subject); blocked {
			middleware.Error(w, http.StatusTooManyRequests, "too many mfa attempts", r.Header.Get("X-Request-ID"))
			return
		}
		writeJSON(w, 401, map[string]string{"error": "invalid totp code"})
		return
	}
	s.bf.Reset("mfa:" + r.RemoteAddr + ":" + cl.Subject)
	tok, _ := auth.Sign(s.jwtSecret, cl.Subject, cl.Organization, cl.Roles, 12*time.Hour)
	writeJSON(w, 200, map[string]any{"token": tok, "user": cl.Subject})
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
	// Persist evidence (result only — host saved, credentials never).
	host := ""
	if body.Config != nil {
		host = body.Config["host"]
		if host == "" {
			host = body.Config["base_url"]
		}
		if host == "" {
			host = body.Config["target"]
		}
	}
	tester := ""
	if cl := auth.ClaimsOf(r.Context()); cl != nil {
		tester = cl.Subject
	}
	if s.store != nil {
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		_, _ = s.store.SaveLabRun(r.Context(), rbac.OrgOf(r.Context()),
			body.Vendor, body.Family, body.ConnectionType, host,
			err == nil, res.LatencyMs, res, res.Capabilities, res.Probes, errStr, tester)
	}
	if err != nil {
		writeJSON(w, 200, map[string]any{"healthy": false, "error": err.Error(), "result": res})
		return
	}
	writeJSON(w, 200, map[string]any{"healthy": true, "result": res})
}

func (s *Server) handleNOC(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		middleware.Error(w, http.StatusServiceUnavailable, "database not configured", r.Header.Get("X-Request-ID"))
		return
	}
	sum, err := s.store.NOC(r.Context(), rbac.OrgOf(r.Context()))
	if err != nil {
		middleware.Error(w, http.StatusBadGateway, "noc query failed", r.Header.Get("X-Request-ID"))
		return
	}
	writeJSON(w, 200, sum)
}

// handleResource serves tenant-scoped paginated lists:
// ?search=&status=&page=&per_page=&sort=&dir= ; ?id= fetches one row.
func (s *Server) handleResource(name string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.store == nil {
			middleware.Error(w, http.StatusServiceUnavailable, "database not configured", r.Header.Get("X-Request-ID"))
			return
		}
		q := r.URL.Query()
		if id := q.Get("id"); id != "" {
			row, err := s.store.GetByID(r.Context(), rbac.OrgOf(r.Context()), name, id)
			if err != nil {
				middleware.Error(w, http.StatusNotFound, "not found", r.Header.Get("X-Request-ID"))
				return
			}
			writeJSON(w, 200, row)
			return
		}
		pg, err := s.store.List(r.Context(), rbac.OrgOf(r.Context()), name, store.Query{
			Search: q.Get("search"), Status: q.Get("status"),
			Page: atoi(q.Get("page")), PerPage: atoi(q.Get("per_page")),
			Sort: q.Get("sort"), Dir: q.Get("dir"),
		})
		if err != nil {
			middleware.Error(w, http.StatusBadGateway, "query failed", r.Header.Get("X-Request-ID"))
			return
		}
		writeJSON(w, 200, pg)
	}
}

func (s *Server) handleSettings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]any{"data": map[string]string{"default_lang": "en"}})
}

func (s *Server) handleTopology(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		middleware.Error(w, http.StatusServiceUnavailable, "database not configured", r.Header.Get("X-Request-ID"))
		return
	}
	ctx := r.Context()
	org := rbac.OrgOf(ctx)
	devs, err := s.store.List(ctx, org, "devices", store.Query{PerPage: 100})
	if err != nil {
		middleware.Error(w, http.StatusBadGateway, "query failed", r.Header.Get("X-Request-ID"))
		return
	}
	sites, _ := s.store.List(ctx, org, "sites", store.Query{PerPage: 100})
	g := map[string]any{"nodes": []any{}, "edges": []any{}}
	var nodes []any
	if sites != nil {
		for _, n := range sites.Data {
			nodes = append(nodes, map[string]any{"id": n["id"], "kind": "site", "label": n["name"]})
		}
	}
	var edges []any
	for _, d := range devs.Data {
		nodes = append(nodes, map[string]any{"id": d["id"], "kind": "device", "label": d["host"], "status": d["status"]})
		edges = append(edges, map[string]any{"from": "site", "to": d["id"], "kind": "logical"})
	}
	g["nodes"], g["edges"] = nodes, edges
	writeJSON(w, 200, g)
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	return n
}
