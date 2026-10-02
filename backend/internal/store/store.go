// Package store is the tenant-scoped PostgreSQL read/write seam.
// Every query is filtered by org_id server-side — tenant isolation never
// relies on frontend filtering. Pagination/filtering/sorting are enforced
// here with per-resource column whitelists (SQL-injection safe).
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/universal-isp/platform/internal/security"
)

var ErrNoDatabase = errors.New("database not configured")

type Store struct {
	db      *sql.DB
	Secrets *security.SecretsBox // nullable; required for TOTP verify (decrypts totp_secret_enc)
}

func New(db *sql.DB) *Store { return &Store{db: db} }

// WithSecrets attaches the credential vault (AES-GCM) for secret decryption.
func (s *Store) WithSecrets(box *security.SecretsBox) *Store {
	s.Secrets = box
	return s
}

type Query struct {
	Search  string
	Status  string
	Page    int
	PerPage int
	Sort    string
	Dir     string // asc | desc
}

func (q Query) norm() Query {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PerPage < 1 || q.PerPage > 100 {
		q.PerPage = 25
	}
	if q.Dir != "asc" {
		q.Dir = "desc"
	}
	return q
}

type resource struct {
	table      string
	cols       []string
	searchCols []string
	sortCols   []string
	hasStatus  bool
}

var resources = map[string]resource{
	"customers":        {"customers", []string{"id", "name", "email", "phone", "status", "created_at"}, []string{"name", "email", "phone"}, []string{"created_at", "name"}, true},
	"subscriptions":    {"subscriptions", []string{"id", "customer_id", "package_id", "username", "service", "status", "created_at"}, []string{"username"}, []string{"created_at"}, true},
	"packages":         {"packages", []string{"id", "name", "price_cents", "service", "created_at"}, []string{"name"}, []string{"created_at", "name"}, false},
	"invoices":         {"invoices", []string{"id", "customer_id", "total_cents", "paid_cents", "status", "due_at", "created_at"}, nil, []string{"created_at", "due_at"}, true},
	"payments":         {"payments", []string{"id", "invoice_id", "amount_cents", "method", "provider", "status", "created_at"}, nil, []string{"created_at"}, true},
	"devices":          {"devices", []string{"id", "vendor", "family", "model", "host", "connection_type", "status", "created_at"}, []string{"vendor", "model", "host"}, []string{"created_at"}, true},
	"sites":            {"sites", []string{"id", "name", "region", "created_at"}, []string{"name", "region"}, []string{"created_at", "name"}, false},
	"alerts":           {"alerts", []string{"id", "severity", "title", "status", "created_at"}, []string{"title"}, []string{"created_at"}, true},
	"events":           {"events", []string{"id", "type", "actor", "resource", "created_at"}, []string{"type", "actor"}, []string{"created_at"}, false},
	"network_jobs":     {"network_jobs", []string{"id", "kind", "status", "attempts", "created_at"}, []string{"kind"}, []string{"created_at"}, true},
	"audit_logs":       {"audit_logs", []string{"id", "actor", "action", "resource", "created_at"}, []string{"actor", "action"}, []string{"created_at"}, false},
	"incidents":        {"incidents", []string{"id", "title", "severity", "status", "scope_key", "opened_at"}, []string{"title", "scope_key"}, []string{"opened_at"}, true},
	"tickets":          {"tickets", []string{"id", "subject", "status", "priority", "created_at"}, []string{"subject"}, []string{"created_at"}, true},
	"work_orders":      {"work_orders", []string{"id", "kind", "status", "scheduled_for", "created_at"}, []string{"kind"}, []string{"created_at"}, true},
	"contracts":        {"contracts", []string{"id", "kind", "status", "mrc_cents", "created_at"}, []string{"kind"}, []string{"created_at"}, true},
	"config_snapshots": {"config_snapshots", []string{"id", "device_id", "version", "taken_by", "taken_at"}, nil, []string{"taken_at"}, false},
	"changes":          {"changes", []string{"id", "device_id", "summary", "status", "created_at"}, []string{"summary"}, []string{"created_at"}, true},
	"radius_sessions":  {"radius_sessions", []string{"id", "username", "nas_ip", "framed_ip", "started_at"}, []string{"username", "nas_ip"}, []string{"started_at"}, false},
	"lab_runs":         {"lab_runs", []string{"id", "vendor", "family", "connection_type", "host", "healthy", "latency_ms", "error", "tested_at"}, []string{"vendor", "host"}, []string{"tested_at"}, false},
}

type Page struct {
	Data    []map[string]any `json:"data"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	PerPage int              `json:"per_page"`
}

func (s *Store) List(ctx context.Context, orgID, name string, q Query) (*Page, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	r, ok := resources[name]
	if !ok {
		return nil, fmt.Errorf("unknown resource %q", name)
	}
	q = q.norm()
	sort := "created_at"
	for _, c := range r.sortCols {
		if q.Sort == c {
			sort = c
		}
	}
	where := "org_id = $1"
	args := []any{orgID}
	if r.hasStatus && q.Status != "" {
		args = append(args, q.Status)
		where += fmt.Sprintf(" AND status = $%d", len(args))
	}
	if q.Search != "" && len(r.searchCols) > 0 {
		var ors []string
		for _, c := range r.searchCols {
			args = append(args, "%"+q.Search+"%")
			ors = append(ors, fmt.Sprintf("%s ILIKE $%d", c, len(args)))
		}
		where += " AND (" + strings.Join(ors, " OR ") + ")"
	}
	var total int
	if err := s.db.QueryRowContext(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", r.table, where), args...).Scan(&total); err != nil {
		return nil, err
	}
	offset := (q.Page - 1) * q.PerPage
	args = append(args, q.PerPage, offset)
	sel := fmt.Sprintf("SELECT %s FROM %s WHERE %s ORDER BY %s %s LIMIT $%d OFFSET $%d",
		strings.Join(r.cols, ", "), r.table, where, sort, strings.ToUpper(q.Dir), len(args)-1, len(args))
	rows, err := s.db.QueryContext(ctx, sel, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		vals := make([]any, len(r.cols))
		ptrs := make([]any, len(r.cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		m := map[string]any{}
		for i, c := range r.cols {
			m[c] = vals[i]
		}
		out = append(out, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &Page{Data: out, Total: total, Page: q.Page, PerPage: q.PerPage}, nil
}

// GetByID fetches one row scoped to org.
func (s *Store) GetByID(ctx context.Context, orgID, name, id string) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	r, ok := resources[name]
	if !ok {
		return nil, fmt.Errorf("unknown resource %q", name)
	}
	vals := make([]any, len(r.cols))
	ptrs := make([]any, len(r.cols))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	err := s.db.QueryRowContext(ctx,
		fmt.Sprintf("SELECT %s FROM %s WHERE org_id = $1 AND id = $2", strings.Join(r.cols, ", "), r.table),
		orgID, id).Scan(ptrs...)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("not found")
	}
	if err != nil {
		return nil, err
	}
	m := map[string]any{}
	for i, c := range r.cols {
		m[c] = vals[i]
	}
	return m, nil
}
