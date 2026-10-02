package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// SaveLabRun persists Connection Lab evidence (never credentials).
func (s *Store) SaveLabRun(ctx context.Context, orgID string, vendor, family, connType, host string,
	healthy bool, latencyMs int64, info, caps, probes any, testErr, testedBy string) (string, error) {
	if s == nil || s.db == nil {
		return "", ErrNoDatabase
	}
	marshal := func(v any) []byte {
		b, _ := json.Marshal(v)
		if b == nil {
			return []byte("null")
		}
		return b
	}
	var id string
	err := s.db.QueryRowContext(ctx, `INSERT INTO lab_runs
		(org_id, vendor, family, connection_type, host, healthy, latency_ms, device_info, capabilities, probes, error, tested_by)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`,
		orgID, vendor, family, connType, host, healthy, latencyMs,
		string(marshal(info)), string(marshal(caps)), string(marshal(probes)), testErr, testedBy).Scan(&id)
	return id, err
}

// IncidentAction applies ack/assign/resolve with a timeline-worthy update.
func (s *Store) IncidentAction(ctx context.Context, orgID, id, action, actor, extra string) error {
	if s == nil || s.db == nil {
		return ErrNoDatabase
	}
	now := time.Now().UTC()
	switch action {
	case "ack":
		_, err := s.db.ExecContext(ctx,
			`UPDATE incidents SET status='acknowledged', acked_at=$3 WHERE org_id=$1 AND id=$2 AND status='open'`,
			orgID, id, now)
		return err
	case "assign":
		if extra == "" {
			return fmt.Errorf("assignee required")
		}
		_, err := s.db.ExecContext(ctx,
			`UPDATE incidents SET assignee=$3 WHERE org_id=$1 AND id=$2`, orgID, id, extra)
		return err
	case "resolve":
		_, err := s.db.ExecContext(ctx,
			`UPDATE incidents SET status='resolved', resolved_at=$3, root_cause=COALESCE(NULLIF($4,''),root_cause) WHERE org_id=$1 AND id=$2 AND status!='resolved'`,
			orgID, id, now, extra)
		return err
	default:
		return fmt.Errorf("unknown incident action %q", action)
	}
}

// Customer360 composes customer, contacts, addresses, subscriptions,
// invoices, payments, tickets from tenant-scoped queries. Sessions and
// health resolve per subscription by the caller (see /service-health).
func (s *Store) Customer360(ctx context.Context, orgID, customerID string) (map[string]any, error) {
	if s == nil || s.db == nil {
		return nil, ErrNoDatabase
	}
	cust, err := s.GetByID(ctx, orgID, "customers", customerID)
	if err != nil {
		return nil, err
	}
	out := map[string]any{"customer": cust}
	collect := func(table, fk string) []map[string]any {
		rows, err := s.db.QueryContext(ctx,
			fmt.Sprintf(`SELECT * FROM %s WHERE org_id=$1 AND %s=$2 ORDER BY created_at DESC LIMIT 50`, table, fk),
			orgID, customerID)
		if err != nil {
			return nil
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		var list []map[string]any
		for rows.Next() {
			vals := make([]any, len(cols))
			ptrs := make([]any, len(cols))
			for i := range vals {
				ptrs[i] = &vals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				break
			}
			m := map[string]any{}
			for i, c := range cols {
				m[c] = vals[i]
			}
			list = append(list, m)
		}
		return list
	}
	out["contacts"] = collect("customer_contacts", "customer_id")
	out["addresses"] = collect("service_addresses", "customer_id")
	out["subscriptions"] = collect("subscriptions", "customer_id")
	out["invoices"] = collect("invoices", "customer_id")
	out["tickets"] = collect("tickets", "customer_id")
	out["payments"] = s.paymentsForCustomer(ctx, orgID, customerID)
	return out, nil
}

func (s *Store) paymentsForCustomer(ctx context.Context, orgID, customerID string) []map[string]any {
	rows, err := s.db.QueryContext(ctx,
		`SELECT p.* FROM payments p JOIN invoices i ON i.id = p.invoice_id
		 WHERE p.org_id=$1 AND i.customer_id=$2 ORDER BY p.created_at DESC LIMIT 50`,
		orgID, customerID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	cols, _ := rows.Columns()
	var list []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			break
		}
		m := map[string]any{}
		for i, c := range cols {
			m[c] = vals[i]
		}
		list = append(list, m)
	}
	return list
}
