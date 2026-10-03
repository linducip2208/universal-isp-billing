package httpapi

import (
	"encoding/csv"
	"fmt"
	"net/http"

	"github.com/universal-isp/platform/internal/middleware"
	"github.com/universal-isp/platform/internal/rbac"
	"github.com/universal-isp/platform/internal/store"
)

// CSV exports stream directly (bounded 5000 rows) for finance/BI tooling.

func (s *Server) handleCSVRevenue(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		middleware.Error(w, http.StatusServiceUnavailable, "database not configured", r.Header.Get("X-Request-ID"))
		return
	}
	sum, err := s.store.EconomicsSummary(r.Context(), rbac.OrgOf(r.Context()))
	if err != nil {
		middleware.Error(w, http.StatusBadGateway, "query failed", r.Header.Get("X-Request-ID"))
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=revenue.csv")
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"metric", "value_cents"})
	_ = cw.Write([]string{"mrr", itoa(sum.MRRcents)})
	_ = cw.Write([]string{"arpu", itoa(sum.ARPUcents)})
	for pack, v := range sum.RevenueByPack {
		_ = cw.Write([]string{"package:" + pack, itoa(v)})
	}
	cw.Flush()
}

func (s *Server) handleCSVSubs(w http.ResponseWriter, r *http.Request) {
	if s.store == nil {
		middleware.Error(w, http.StatusServiceUnavailable, "database not configured", r.Header.Get("X-Request-ID"))
		return
	}
	pg, err := s.store.List(r.Context(), rbac.OrgOf(r.Context()), "subscriptions", store.Query{PerPage: 100})
	if err != nil {
		middleware.Error(w, http.StatusBadGateway, "query failed", r.Header.Get("X-Request-ID"))
		return
	}
	_ = pg
	// Full export path streams in chunks; page through server-side.
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=subscribers.csv")
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"id", "username", "service", "status"})
	page := 1
	for {
		chunk, err := s.store.List(r.Context(), rbac.OrgOf(r.Context()), "subscriptions",
			store.Query{Page: page, PerPage: 500})
		if err != nil || len(chunk.Data) == 0 {
			break
		}
		for _, row := range chunk.Data {
			_ = cw.Write([]string{sstr(row["id"]), sstr(row["username"]), sstr(row["service"]), sstr(row["status"])})
		}
		if len(chunk.Data) < 500 {
			break
		}
		page++
		if page > 20 { // 10k cap per export; larger needs async report jobs
			break
		}
	}
	cw.Flush()
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [32]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func sstr(v any) string {
	if v == nil {
		return ""
	}
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return fmt.Sprint(v)
}
