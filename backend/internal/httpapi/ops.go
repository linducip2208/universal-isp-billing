package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/universal-isp/platform/internal/auth"
	"github.com/universal-isp/platform/internal/copilot"
	"github.com/universal-isp/platform/internal/middleware"
	"github.com/universal-isp/platform/internal/rbac"
	"github.com/universal-isp/platform/internal/servicehealth"
	"github.com/universal-isp/platform/internal/store"
)

type radiusLookupTool struct{ st *store.Store }

func (t radiusLookupTool) Name() string { return "radius_lookup" }
func (t radiusLookupTool) Query(ctx context.Context, args map[string]string) ([]copilot.Evidence, error) {
	if t.st == nil {
		return nil, fmt.Errorf("no database")
	}
	pg, err := t.st.List(ctx, rbac.OrgOf(ctx), "radius_sessions", store.Query{Search: args["q"], PerPage: 5})
	if err != nil {
		return nil, err
	}
	var out []copilot.Evidence
	for _, r := range pg.Data {
		out = append(out, copilot.Evidence{Source: "radius_sessions",
			Ref:    fmt.Sprint(r["id"]),
			Detail: fmt.Sprintf("user=%v nas=%v ip=%v since=%v", r["username"], r["nas_ip"], r["framed_ip"], r["started_at"])})
	}
	return out, nil
}

type incidentSummaryTool struct{ st *store.Store }

func (t incidentSummaryTool) Name() string { return "incident_summary" }
func (t incidentSummaryTool) Query(ctx context.Context, args map[string]string) ([]copilot.Evidence, error) {
	if t.st == nil {
		return nil, fmt.Errorf("no database")
	}
	pg, err := t.st.List(ctx, rbac.OrgOf(ctx), "alerts", store.Query{PerPage: 5})
	if err != nil {
		return nil, err
	}
	var out []copilot.Evidence
	for _, r := range pg.Data {
		out = append(out, copilot.Evidence{Source: "alerts", Ref: fmt.Sprint(r["id"]),
			Detail: fmt.Sprintf("[%v] %v (%v)", r["severity"], r["title"], r["status"])})
	}
	return out, nil
}

func (s *Server) handleCopilot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Question string `json:"question"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Question == "" {
		writeJSON(w, 400, map[string]string{"error": "question required"})
		return
	}
	o := copilot.New(radiusLookupTool{s.store}, incidentSummaryTool{s.store})
	ans, err := o.Ask(r.Context(), rbac.RolesOf(r.Context()), body.Question)
	if err != nil {
		middleware.Error(w, http.StatusForbidden, err.Error(), r.Header.Get("X-Request-ID"))
		return
	}
	writeJSON(w, 200, ans)
}

func (s *Server) handleEconomics(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		middleware.Error(w, http.StatusServiceUnavailable, "database not configured", r.Header.Get("X-Request-ID"))
		return
	}
	sum, err := s.store.EconomicsSummary(r.Context(), rbac.OrgOf(r.Context()))
	if err != nil {
		middleware.Error(w, http.StatusBadGateway, "query failed", r.Header.Get("X-Request-ID"))
		return
	}
	writeJSON(w, 200, sum)
}

func (s *Server) handleServiceHealth(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		middleware.Error(w, http.StatusServiceUnavailable, "database not configured", r.Header.Get("X-Request-ID"))
		return
	}
	subID := r.URL.Query().Get("subscriber")
	if subID == "" {
		writeJSON(w, 400, map[string]string{"error": "?subscriber=<subscription-id> required"})
		return
	}
	ctx := r.Context()
	org := rbac.OrgOf(ctx)
	st := s.store
	sub, err := st.GetByID(ctx, org, "subscriptions", subID)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "subscription not found"})
		return
	}
	username := fmt.Sprint(sub["username"])
	fetchers := []servicehealth.Fetcher{
		func(ctx context.Context, id string) servicehealth.Section {
			status := "ok"
			if sub["status"] == "suspended" {
				status = "down"
			} else if sub["status"] != "active" {
				status = "degraded"
			}
			return servicehealth.Section{Source: "subscriptions", Status: status,
				Detail: fmt.Sprintf("service=%v status=%v", sub["service"], sub["status"])}
		},
		func(ctx context.Context, id string) servicehealth.Section {
			if username == "" {
				return servicehealth.Section{Source: "radius_sessions", Status: "unknown", Detail: "no username"}
			}
			pg, err := st.List(ctx, org, "radius_sessions", store.Query{Search: username, PerPage: 1})
			if err != nil || len(pg.Data) == 0 {
				return servicehealth.Section{Source: "radius_sessions", Status: "down", Detail: "no active session"}
			}
			sess := pg.Data[0]
			return servicehealth.Section{Source: "radius_sessions", Status: "ok",
				Detail: fmt.Sprintf("online via %v ip=%v", sess["nas_ip"], sess["framed_ip"])}
		},
	}
	writeJSON(w, 200, servicehealth.Compose(ctx, subID, fetchers))
}

func (s *Server) handleRadiusSessions(w http.ResponseWriter, r *http.Request) {
	s.handleResource("radius_sessions")(w, r)
}

func (s *Server) handleCustomer360(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		middleware.Error(w, http.StatusServiceUnavailable, "database not configured", r.Header.Get("X-Request-ID"))
		return
	}
	id := r.URL.Query().Get("customer")
	if id == "" {
		writeJSON(w, 400, map[string]string{"error": "?customer=<customer-id> required"})
		return
	}
	out, err := s.store.Customer360(r.Context(), rbac.OrgOf(r.Context()), id)
	if err != nil {
		middleware.Error(w, http.StatusNotFound, "customer not found", r.Header.Get("X-Request-ID"))
		return
	}
	writeJSON(w, 200, out)
}

func (s *Server) handleIncidentAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.store == nil {
		middleware.Error(w, http.StatusServiceUnavailable, "database not configured", r.Header.Get("X-Request-ID"))
		return
	}
	var body struct {
		ID     string `json:"id"`
		Action string `json:"action"` // ack | assign | resolve
		Actor  string `json:"actor"`
		Extra  string `json:"extra,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.ID == "" || body.Action == "" {
		writeJSON(w, 400, map[string]string{"error": "id and action required"})
		return
	}
	if body.Actor == "" {
		if cl := auth.ClaimsOf(r.Context()); cl != nil {
			body.Actor = cl.Subject
		}
	}
	if err := s.store.IncidentAction(r.Context(), rbac.OrgOf(r.Context()), body.ID, body.Action, body.Actor, body.Extra); err != nil {
		middleware.Error(w, http.StatusBadRequest, err.Error(), r.Header.Get("X-Request-ID"))
		return
	}
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
