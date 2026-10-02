package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/universal-isp/platform/internal/database"
	"github.com/universal-isp/platform/internal/store"
)

// TestLiveTelemetry writes + reads back samples; cross-org write refused.
func TestLiveTelemetry(t *testing.T) {
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
	dev := "99999999-9999-9999-9999-999999999999" // seeded simulated device
	s := store.New(db)
	if err := s.RecordTrafficSample(ctx, org, dev, "ether1", 1000, 2000); err != nil {
		t.Fatalf("record: %v", err)
	}
	if err := s.RecordTrafficSample(ctx, "00000000-0000-0000-0000-000000000000", dev, "ether1", 1, 1); err == nil {
		t.Fatal("cross-org write must fail")
	}
	if err := s.RecordInterfaceSample(ctx, dev, "ether1", 100, 200, 0); err != nil {
		t.Fatalf("iface: %v", err)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM traffic_samples WHERE device_id=$1`, dev).Scan(&n); err != nil || n == 0 {
		t.Fatalf("samples missing: %v n=%d", err, n)
	}
	db.ExecContext(ctx, `DELETE FROM traffic_samples WHERE device_id=$1`, dev)
	db.ExecContext(ctx, `DELETE FROM interface_samples WHERE device_id=$1`, dev)
}
