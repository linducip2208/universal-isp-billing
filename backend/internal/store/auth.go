package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/universal-isp/platform/internal/mfa"
	"github.com/universal-isp/platform/internal/security"
)

// Authenticate verifies username/password against users + roles.
// Returns rotate (90-day policy) and mfaRequired (TOTP enrolled).
func (s *Store) Authenticate(ctx context.Context, username, password string) (userID, orgID string, roles []string, rotate, mfaRequired bool, err error) {
	if s == nil || s.db == nil {
		return "", "", nil, false, false, ErrNoDatabase
	}
	var hash string
	var changedAt *time.Time
	var totpEnabled bool
	err = s.db.QueryRowContext(ctx,
		`SELECT id, org_id, password_hash, password_changed_at, COALESCE(totp_enabled,false) FROM users WHERE username = $1 AND deleted_at IS NULL`,
		username).Scan(&userID, &orgID, &hash, &changedAt, &totpEnabled)
	if err == sql.ErrNoRows {
		return "", "", nil, false, false, sql.ErrNoRows
	}
	if err != nil {
		return "", "", nil, false, false, err
	}
	if err := security.VerifyPassword(password, hash); err != nil {
		return "", "", nil, false, false, err
	}
	if changedAt == nil || time.Since(*changedAt) > 90*24*time.Hour {
		rotate = true
	}
	mfaRequired = totpEnabled
	_, _ = s.db.ExecContext(ctx, `UPDATE users SET last_login_at = now() WHERE id = $1`, userID)
	rows, err := s.db.QueryContext(ctx,
		`SELECT r.name FROM roles r JOIN user_roles ur ON ur.role_id = r.id WHERE ur.user_id = $1`, userID)
	if err != nil {
		return "", "", nil, rotate, mfaRequired, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return "", "", nil, rotate, mfaRequired, err
		}
		roles = append(roles, name)
	}
	return userID, orgID, roles, rotate, mfaRequired, rows.Err()
}

// EnrollMFA generates a TOTP secret for the user, stores it encrypted with
// totp_enabled=false, and returns the one-time secret + otpauth URL.
// The plaintext secret is shown exactly once (QR scan); only the ciphertext
// is persisted.
func (s *Store) EnrollMFA(ctx context.Context, username, issuer string) (secret, url string, err error) {
	if s == nil || s.db == nil {
		return "", "", ErrNoDatabase
	}
	if s.Secrets == nil {
		return "", "", errors.New("mfa vault unavailable")
	}
	secret, err = mfa.GenerateSecret()
	if err != nil {
		return "", "", err
	}
	enc, err := s.Secrets.Encrypt(secret)
	if err != nil {
		return "", "", err
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE users SET totp_secret_enc=$2, totp_enabled=false WHERE username=$1 AND deleted_at IS NULL`, username, enc)
	if err != nil {
		return "", "", err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return "", "", sql.ErrNoRows
	}
	return secret, mfa.OtpauthURL(issuer, username, secret), nil
}

// ConfirmMFA enables TOTP after the user proves possession with a valid code.
func (s *Store) ConfirmMFA(ctx context.Context, username, code string) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	if s.Secrets == nil {
		return errors.New("mfa vault unavailable")
	}
	var enc string
	err := s.db.QueryRowContext(ctx,
		`SELECT totp_secret_enc FROM users WHERE username=$1 AND deleted_at IS NULL`, username).Scan(&enc)
	if err != nil || enc == "" {
		return errors.New("no pending enrollment")
	}
	secret, err := s.Secrets.Decrypt(enc)
	if err != nil {
		return errors.New("totp vault failure")
	}
	if !mfa.Verify(secret, code, time.Now(), 1) {
		return errors.New("invalid totp code")
	}
	_, err = s.db.ExecContext(ctx, `UPDATE users SET totp_enabled=true WHERE username=$1`, username)
	return err
}

// TOTP-enrolled user. The secret is AES-GCM-encrypted at rest and decrypted
// only here, in memory, per attempt. A consumed backup code is marked used.
func (s *Store) VerifyTOTP(ctx context.Context, username, code string) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	if s.Secrets == nil {
		return errors.New("mfa vault unavailable")
	}
	var userID, enc string
	var enabled bool
	err := s.db.QueryRowContext(ctx,
		`SELECT id, totp_secret_enc, totp_enabled FROM users WHERE username=$1 AND deleted_at IS NULL`,
		username).Scan(&userID, &enc, &enabled)
	if err != nil {
		return err
	}
	if !enabled || enc == "" {
		return errors.New("totp not enrolled")
	}
	if secret, err := s.Secrets.Decrypt(enc); err == nil {
		if mfa.Verify(secret, code, time.Now(), 1) {
			return nil
		}
	} else {
		return errors.New("totp vault failure")
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, code_hash FROM totp_backup_codes WHERE user_id=$1 AND used_at IS NULL`, userID)
	if err != nil {
		return errors.New("invalid totp code")
	}
	defer rows.Close()
	for rows.Next() {
		var id, h string
		if err := rows.Scan(&id, &h); err != nil {
			continue
		}
		if mfa.VerifyBackupCode(code, []string{h}) {
			_, _ = s.db.ExecContext(ctx, `UPDATE totp_backup_codes SET used_at=now() WHERE id=$1`, id)
			return nil
		}
	}
	return errors.New("invalid totp code")
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
