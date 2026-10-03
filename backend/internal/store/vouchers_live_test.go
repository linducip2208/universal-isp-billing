package store_test

import (
	"context"
	"os"
	"testing"

	"github.com/universal-isp/platform/internal/database"
	"github.com/universal-isp/platform/internal/store"
	"github.com/universal-isp/platform/internal/vouchers"
)

func TestLiveVouchers(t *testing.T) {
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
	s := store.New(db)
	code, err := vouchers.GenerateCode(8)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateVoucher(ctx, org, code, "", 24, nil); err != nil {
		t.Fatalf("create: %v", err)
	}
	defer db.ExecContext(ctx, `DELETE FROM vouchers WHERE org_id=$1 AND code=$2`, org, code)
	if err := s.RedeemVoucher(ctx, org, code, "mac-1"); err != nil {
		t.Fatalf("redeem: %v", err)
	}
	if err := s.RedeemVoucher(ctx, org, code, "mac-2"); err == nil {
		t.Fatal("double redeem must fail")
	}
	if err := s.RedeemVoucher(ctx, org, "NOPE1234", "x"); err == nil {
		t.Fatal("unknown code must fail")
	}
}
