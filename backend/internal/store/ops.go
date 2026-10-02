package store

import (
	"context"
	"fmt"

	"github.com/universal-isp/platform/internal/economics"
	"github.com/universal-isp/platform/internal/subscriptions"
)

// EconomicsSummary computes MRR/ARPU/churn inputs live: MRR = sum of active
// subscription package prices; churn base = non-new subs this period.
func (s *Store) EconomicsSummary(ctx context.Context, orgID string) (economics.Summary, error) {
	if s == nil || s.db == nil {
		return economics.Summary{}, ErrNoDatabase
	}
	var mrr int64
	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(p.price_cents),0) FROM subscriptions s
		JOIN packages p ON p.id = s.package_id
		WHERE s.org_id=$1 AND s.status='active' AND s.deleted_at IS NULL`, orgID).Scan(&mrr)
	if err != nil {
		return economics.Summary{}, err
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT p.name, COUNT(*), COALESCE(SUM(p.price_cents),0) FROM subscriptions s
		JOIN packages p ON p.id = s.package_id
		WHERE s.org_id=$1 AND s.status='active' AND s.deleted_at IS NULL
		GROUP BY p.name`, orgID)
	if err != nil {
		return economics.Summary{}, err
	}
	defer rows.Close()
	byPack := map[string]int64{}
	for rows.Next() {
		var name string
		var n int
		var sum int64
		if err := rows.Scan(&name, &n, &sum); err != nil {
			return economics.Summary{}, err
		}
		_ = n
		byPack[name] = sum
	}
	if err := rows.Err(); err != nil {
		return economics.Summary{}, err
	}
	var active, susp, term int
	err = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FILTER (WHERE status='active'),
		       COUNT(*) FILTER (WHERE status='suspended'),
		       COUNT(*) FILTER (WHERE status='terminated')
		FROM subscriptions WHERE org_id=$1 AND deleted_at IS NULL`, orgID).Scan(&active, &susp, &term)
	if err != nil {
		return economics.Summary{}, err
	}
	subs := make([]economics.SubRow, 0, active+susp+term)
	for i := 0; i < active; i++ {
		subs = append(subs, economics.SubRow{Status: "active"})
	}
	for i := 0; i < susp; i++ {
		subs = append(subs, economics.SubRow{Status: "suspended"})
	}
	for i := 0; i < term; i++ {
		subs = append(subs, economics.SubRow{Status: "terminated", Churned: true})
	}
	mrrRows := []economics.MoneyRow{{Amount: mrr}}
	sum := economics.Summarize(mrrRows, subs)
	sum.RevenueByPack = byPack
	return sum, nil
}

// BulkItem is one per-item bulk result (partial success is explicit).
type BulkItem struct {
	ID     string `json:"id"`
	OK     bool   `json:"ok"`
	Error  string `json:"error,omitempty"`
	Status string `json:"status,omitempty"`
}

// BulkSubscriptionStatus applies suspend/activate/terminate to many
// subscriptions with per-row lifecycle validation. Each row is independent;
// failures never roll back siblings (caller retries by ID).
func (s *Store) BulkSubscriptionStatus(ctx context.Context, orgID string, ids []string, action string) ([]BulkItem, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	var to subscriptions.Status
	switch action {
	case "suspend":
		to = subscriptions.Suspended
	case "activate":
		to = subscriptions.Active
	case "terminate":
		to = subscriptions.Terminated
	default:
		return nil, fmt.Errorf("unknown bulk action %q", action)
	}
	out := make([]BulkItem, 0, len(ids))
	for _, id := range ids {
		var cur string
		err := s.db.QueryRowContext(ctx,
			`SELECT status FROM subscriptions WHERE org_id=$1 AND id=$2 AND deleted_at IS NULL`, orgID, id).Scan(&cur)
		if err != nil {
			out = append(out, BulkItem{ID: id, Error: "not found"})
			continue
		}
		sub := &subscriptions.Subscription{Status: subscriptions.Status(cur)}
		if err := sub.Transition(to); err != nil {
			out = append(out, BulkItem{ID: id, Error: err.Error(), Status: cur})
			continue
		}
		if _, err := s.db.ExecContext(ctx,
			`UPDATE subscriptions SET status=$3, updated_at=now() WHERE org_id=$1 AND id=$2`, orgID, id, string(to)); err != nil {
			out = append(out, BulkItem{ID: id, Error: err.Error(), Status: cur})
			continue
		}
		out = append(out, BulkItem{ID: id, OK: true, Status: string(to)})
	}
	return out, nil
}

// RecordTrafficSample persists one device traffic reading (telemetry writer
// for the polling loop; partition-ready table). The device must belong to
// the org (tenant guard on write path too).
func (s *Store) RecordTrafficSample(ctx context.Context, orgID, deviceID, target string, rxBps, txBps int64) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	var ok bool
	err := s.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM devices WHERE org_id=$1 AND id=$2 AND deleted_at IS NULL)`, orgID, deviceID).Scan(&ok)
	if err != nil || !ok {
		return fmt.Errorf("device not in org")
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO traffic_samples(device_id, target, rx_bps, tx_bps) VALUES($1,$2,$3,$4)`,
		deviceID, target, rxBps, txBps)
	return err
}

// RecordInterfaceSample persists one interface reading.
func (s *Store) RecordInterfaceSample(ctx context.Context, deviceID, ifname string, rxBps, txBps, errors int64) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO interface_samples(device_id, ifname, rx_bps, tx_bps, errors) VALUES($1,$2,$3,$4,$5)`,
		deviceID, ifname, rxBps, txBps, errors)
	return err
}
