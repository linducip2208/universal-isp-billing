package store_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/customers"
	"github.com/universal-isp/platform/internal/database"
	"github.com/universal-isp/platform/internal/security"
	"github.com/universal-isp/platform/internal/store"
)

// TestLiveBillingChain proves the money path live: customer -> subscription
// -> numbered invoice (server totals) -> payment applies -> webhook replay
// idempotent -> radius user provisioned + revoked -> password rotated ->
// api key verified. Unique-per-run identifiers make reruns collision-free;
// one ordered cleanup removes everything FK-safely.
func TestLiveBillingChain(t *testing.T) {
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
	tag := fmt.Sprintf("%d", time.Now().UnixNano()%1000000)
	cname, uname, opname := "Chain "+tag, "chain-"+tag, "chain-op-"+tag
	key1, paykey, ref := "ck-"+tag, "pk-"+tag, "ref-"+tag

	cleanup := func() {
		db.ExecContext(ctx, `DELETE FROM invoice_items WHERE invoice_id IN (SELECT id FROM invoices WHERE customer_id IN (SELECT id FROM customers WHERE name=$1))`, cname)
		db.ExecContext(ctx, `DELETE FROM invoice_idem WHERE invoice_id IN (SELECT id FROM invoices WHERE customer_id IN (SELECT id FROM customers WHERE name=$1))`, cname)
		db.ExecContext(ctx, `DELETE FROM payments WHERE invoice_id IN (SELECT id FROM invoices WHERE customer_id IN (SELECT id FROM customers WHERE name=$1))`, cname)
		db.ExecContext(ctx, `DELETE FROM invoices WHERE customer_id IN (SELECT id FROM customers WHERE name=$1)`, cname)
		db.ExecContext(ctx, `DELETE FROM subscriptions WHERE username=$1`, uname)
		db.ExecContext(ctx, `DELETE FROM radius_users WHERE username=$1`, uname)
		db.ExecContext(ctx, `DELETE FROM users WHERE username=$1`, opname)
		db.ExecContext(ctx, `DELETE FROM totp_backup_codes WHERE user_id IN (SELECT id FROM users WHERE username=$1)`, opname)
		db.ExecContext(ctx, `DELETE FROM api_keys WHERE name=$1`, "t-"+tag)
		db.ExecContext(ctx, `DELETE FROM customers WHERE name=$1`, cname)
	}
	defer cleanup()

	custID, err := s.CreateCustomer(ctx, org, customers.Customer{Name: cname, Email: "chain@example.id", Phone: "+628120000009"})
	if err != nil {
		t.Fatalf("customer: %v", err)
	}
	if _, err := s.CreateCustomer(ctx, org, customers.Customer{}); err == nil {
		t.Fatal("nameless customer must fail validation")
	}
	subID, err := s.CreateSubscription(ctx, org, custID, "66666666-6666-6666-6666-666666666666", uname, "pppoe")
	if err != nil {
		t.Fatalf("subscription: %v", err)
	}
	invID, err := s.CreateInvoice(ctx, org, custID, subID,
		[]store.InvoiceLine{{Description: "Home 50M Oct", Qty: 1, UnitCents: 15000000}}, 1100,
		time.Now().Add(7*24*time.Hour), 3, key1)
	if err != nil {
		t.Fatalf("invoice: %v", err)
	}
	// duplicate period key returns original
	if _, err := s.CreateInvoice(ctx, org, custID, subID,
		[]store.InvoiceLine{{Description: "dup", Qty: 1, UnitCents: 1}}, 0,
		time.Now().Add(7*24*time.Hour), 0, key1); err == nil {
		t.Fatal("duplicate invoice key must fail")
	}
	var number string
	var total, sub int64
	_ = db.QueryRowContext(ctx, `SELECT number,total_cents,subtotal_cents FROM invoices WHERE id=$1::uuid`, invID).Scan(&number, &total, &sub)
	if number == "" || total != 16650000 || sub != 15000000 {
		t.Fatalf("server totals wrong: number=%q total=%d sub=%d", number, total, sub)
	}
	payID, err := s.RecordPayment(ctx, org, invID, 16650000, "va", "manual", ref, paykey)
	if err != nil {
		t.Fatalf("payment: %v", err)
	}
	// idempotent replay returns same payment
	if dup, err := s.RecordPayment(ctx, org, invID, 16650000, "va", "manual", ref, paykey); err != nil || dup != payID {
		t.Fatalf("payment replay: %v %q", err, dup)
	}
	var status string
	var paid int64
	_ = db.QueryRowContext(ctx, `SELECT status,paid_cents FROM invoices WHERE id=$1::uuid`, invID).Scan(&status, &paid)
	if status != "paid" || paid != 16650000 {
		t.Fatalf("invoice not settled: %s %d", status, paid)
	}
	// webhook replay: already-paid returns same id, no double-apply
	if got, err := s.ApplyWebhook(ctx, org, "manual", ref, true); err != nil || got != payID {
		t.Fatalf("webhook replay: %v %q", err, got)
	}
	_ = db.QueryRowContext(ctx, `SELECT paid_cents FROM invoices WHERE id=$1::uuid`, invID).Scan(&paid)
	if paid != 16650000 {
		t.Fatalf("double-apply detected: %d", paid)
	}
	// radius user lifecycle
	ruID, err := s.CreateRadiusUser(ctx, org, uname, "radius-pass-1", "", 1)
	if err != nil {
		t.Fatalf("radius user: %v", err)
	}
	_ = ruID
	if err := s.DisableRadiusUser(ctx, org, uname); err != nil {
		t.Fatalf("disable: %v", err)
	}
	// password rotation
	pwHash, _ := security.HashPassword("oldpass-123")
	var uID string
	_ = db.QueryRowContext(ctx, `INSERT INTO users(id,org_id,username,password_hash) VALUES(gen_random_uuid(),$1,$2,$3) RETURNING id`, org, opname, pwHash).Scan(&uID)
	if err := s.ChangePassword(ctx, uID, "wrong", "newpass-123"); err == nil {
		t.Fatal("wrong old password must fail")
	}
	if err := s.ChangePassword(ctx, uID, "oldpass-123", "newpass-123"); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	// api key verify
	raw := "isp_testkey_" + tag
	sum := sha256.Sum256([]byte(raw))
	_, _ = db.ExecContext(ctx, `INSERT INTO api_keys(org_id,name,key_hash,scopes) VALUES($1,$2,$3,'{billing:read}')`, org, "t-"+tag, hex.EncodeToString(sum[:]))
	_, gotOrg, scopes, err := s.VerifyAPIKey(ctx, raw)
	if err != nil || gotOrg != org || len(scopes) != 1 {
		t.Fatalf("apikey: %v %q %v", err, gotOrg, scopes)
	}
	if _, _, _, err := s.VerifyAPIKey(ctx, "isp_nope"); err == nil {
		t.Fatal("bad key must fail")
	}
}
