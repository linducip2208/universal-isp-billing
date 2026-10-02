package store

import (
	"context"

	"github.com/universal-isp/platform/internal/economics"
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
