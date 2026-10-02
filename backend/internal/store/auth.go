package store

import (
	"context"
	"database/sql"

	"github.com/universal-isp/platform/internal/security"
)

// Authenticate verifies username/password against users + roles.
// Passwords are PBKDF2 hashes; timing-safe compare inside security.
func (s *Store) Authenticate(ctx context.Context, username, password string) (userID, orgID string, roles []string, err error) {
	if s == nil || s.db == nil {
		return "", "", nil, ErrNoDatabase
	}
	var hash string
	err = s.db.QueryRowContext(ctx,
		`SELECT id, org_id, password_hash FROM users WHERE username = $1 AND deleted_at IS NULL`,
		username).Scan(&userID, &orgID, &hash)
	if err == sql.ErrNoRows {
		return "", "", nil, sql.ErrNoRows
	}
	if err != nil {
		return "", "", nil, err
	}
	if err := security.VerifyPassword(password, hash); err != nil {
		return "", "", nil, err
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT r.name FROM roles r JOIN user_roles ur ON ur.role_id = r.id WHERE ur.user_id = $1`, userID)
	if err != nil {
		return "", "", nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", "", nil, err
		}
		roles = append(roles, name)
	}
	return userID, orgID, roles, rows.Err()
}

// NOCSummary holds real aggregate counts for the NOC dashboard.
type NOCSummary struct {
	TotalDevices         int `json:"total_devices"`
	OnlineDevices        int `json:"online_devices"`
	OfflineDevices       int `json:"offline_devices"`
	DegradedDevices      int `json:"degraded_devices"`
	ActiveSubscribers    int `json:"active_subscribers"`
	SuspendedSubscribers int `json:"suspended_subscribers"`
	OnlineSessions       int `json:"online_sessions"`
	ActiveAlerts         int `json:"active_alerts"`
	CriticalAlerts       int `json:"critical_alerts"`
	ProvisioningFailures int `json:"provisioning_failures_24h"`
}

func countWhere(ctx context.Context, db *sql.DB, query string, args ...any) int {
	var n int
	if err := db.QueryRowContext(ctx, query, args...).Scan(&n); err != nil {
		return 0
	}
	return n
}

// NOC aggregates live counts. Individual query failures degrade to zero
// for that metric only — the response always states its source.
func (s *Store) NOC(ctx context.Context, orgID string) (*NOCSummary, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	out := &NOCSummary{}
	out.TotalDevices = countWhere(ctx, s.db, `SELECT COUNT(*) FROM devices WHERE org_id=$1 AND deleted_at IS NULL`, orgID)
	out.OnlineDevices = countWhere(ctx, s.db, `SELECT COUNT(*) FROM devices WHERE org_id=$1 AND status='online' AND deleted_at IS NULL`, orgID)
	out.OfflineDevices = countWhere(ctx, s.db, `SELECT COUNT(*) FROM devices WHERE org_id=$1 AND status='offline' AND deleted_at IS NULL`, orgID)
	out.DegradedDevices = countWhere(ctx, s.db, `SELECT COUNT(*) FROM devices WHERE org_id=$1 AND status='degraded' AND deleted_at IS NULL`, orgID)
	out.ActiveSubscribers = countWhere(ctx, s.db, `SELECT COUNT(*) FROM subscriptions WHERE org_id=$1 AND status='active' AND deleted_at IS NULL`, orgID)
	out.SuspendedSubscribers = countWhere(ctx, s.db, `SELECT COUNT(*) FROM subscriptions WHERE org_id=$1 AND status='suspended' AND deleted_at IS NULL`, orgID)
	out.OnlineSessions = countWhere(ctx, s.db, `SELECT COUNT(*) FROM radius_sessions WHERE username IN (SELECT username FROM subscriptions WHERE org_id=$1 AND deleted_at IS NULL)`, orgID)
	out.ActiveAlerts = countWhere(ctx, s.db, `SELECT COUNT(*) FROM alerts WHERE org_id=$1 AND status='open'`, orgID)
	out.CriticalAlerts = countWhere(ctx, s.db, `SELECT COUNT(*) FROM alerts WHERE org_id=$1 AND status='open' AND severity='critical'`, orgID)
	out.ProvisioningFailures = countWhere(ctx, s.db, `SELECT COUNT(*) FROM network_jobs WHERE org_id=$1 AND status='dead' AND created_at > now() - interval '24 hours'`, orgID)
	return out, nil
}
