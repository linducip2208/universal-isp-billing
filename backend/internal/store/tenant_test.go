package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/universal-isp/platform/internal/database"
	"github.com/universal-isp/platform/internal/store"
)

// TestTenantIsolation proves org scoping live: rows in org B are invisible
// to org A through List AND GetByID. Requires TEST_DATABASE_URL.
func TestTenantIsolation(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := database.Open(url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := context.Background()
	orgA, orgB := "00000000-0000-0000-0000-000000000a01", "00000000-0000-0000-0000-000000000b02"
	for _, org := range []string{orgA, orgB} {
		_, _ = db.ExecContext(ctx, `INSERT INTO organizations(id,name) VALUES($1,$2) ON CONFLICT DO NOTHING`, org, "tiso")
	}
	defer db.ExecContext(ctx, `DELETE FROM customers WHERE org_id IN ($1,$2)`, orgA, orgB)
	defer db.ExecContext(ctx, `DELETE FROM organizations WHERE id IN ($1,$2)`, orgA, orgB)
	if _, err := db.ExecContext(ctx, `INSERT INTO customers(id,org_id,name) VALUES(gen_random_uuid(),$1,'A-cust')`, orgA); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO customers(id,org_id,name) VALUES(gen_random_uuid(),$1,'B-cust')`, orgB); err != nil {
		t.Fatal(err)
	}
	s := store.New(db)
	pgA, err := s.List(ctx, orgA, "customers", store.Query{PerPage: 50})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range pgA.Data {
		if r["name"] == "B-cust" {
			t.Fatal("cross-tenant row leaked into list")
		}
	}
	foundA := false
	for _, r := range pgA.Data {
		if r["name"] == "A-cust" {
			foundA = true
		}
	}
	if !foundA {
		t.Fatal("own row missing")
	}
	var bID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM customers WHERE org_id=$1`, orgB).Scan(&bID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetByID(ctx, orgA, "customers", bID); err == nil {
		t.Fatal("cross-tenant GetByID must fail")
	}
}
