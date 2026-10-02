package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/universal-isp/platform/internal/database"
	"github.com/universal-isp/platform/internal/store"
)

// TestLiveBulk exercises per-row lifecycle validation live: active subs
// suspend, terminated ones report errors without touching siblings.
func TestLiveBulk(t *testing.T) {
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
	org := "11111111-1111-1111-1111-111111111111"
	var aID, tID string
	_ = db.QueryRowContext(ctx, `INSERT INTO subscriptions(id,org_id,customer_id,package_id,username,status) VALUES(gen_random_uuid(),$1,'77777777-7777-7777-7777-777777777777','66666666-6666-6666-6666-666666666666','bulk-a','active') RETURNING id`, org).Scan(&aID)
	_ = db.QueryRowContext(ctx, `INSERT INTO subscriptions(id,org_id,customer_id,package_id,username,status) VALUES(gen_random_uuid(),$1,'77777777-7777-7777-7777-777777777777','66666666-6666-6666-6666-666666666666','bulk-t','terminated') RETURNING id`, org).Scan(&tID)
	defer db.ExecContext(ctx, `DELETE FROM subscriptions WHERE username IN ('bulk-a','bulk-t')`)
	s := store.New(db)
	items, err := s.BulkSubscriptionStatus(ctx, org, []string{aID, tID, "00000000-0000-0000-0000-000000000000"}, "suspend")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 3 || !items[0].OK || items[1].OK || items[2].OK {
		t.Fatalf("items=%+v", items)
	}
	if items[2].Error != "not found" {
		t.Fatalf("missing must 404: %+v", items[2])
	}
}
