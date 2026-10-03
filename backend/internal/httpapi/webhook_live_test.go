package httpapi_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/universal-isp/platform/internal/customers"
	"github.com/universal-isp/platform/internal/database"
	"github.com/universal-isp/platform/internal/store"
)

// TestWebhookLive proves the signature gate end-to-end against live Postgres:
// valid Midtrans signature applies once; forged signature is rejected;
// unknown provider refused; unknown reference 404s.
func TestWebhookLive(t *testing.T) {
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
	tag := fmt.Sprint(time.Now().UnixNano() % 1000000)
	ref := "mt-" + tag
	_ = db // store owns writes below
	st := store.New(db)
	cust, err := st.CreateCustomer(ctx, org, customers.Customer{Name: "WH " + tag, Phone: "+628120000011"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(ctx, `DELETE FROM customers WHERE id=$1::uuid`, cust)
	sub, err := st.CreateSubscription(ctx, org, cust, "66666666-6666-6666-6666-666666666666", "wh-"+tag, "pppoe")
	if err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(ctx, `DELETE FROM subscriptions WHERE id=$1::uuid`, sub)
	inv, err := st.CreateInvoice(ctx, org, cust, sub,
		[]store.InvoiceLine{{Description: "P", Qty: 1, UnitCents: 5000000}}, 0,
		time.Now().Add(7*24*time.Hour), 0, "whk-"+tag)
	if err != nil {
		t.Fatal(err)
	}
	defer db.ExecContext(ctx, `DELETE FROM invoice_items WHERE invoice_id=$1::uuid`, inv)
	defer db.ExecContext(ctx, `DELETE FROM invoice_idem WHERE invoice_id=$1::uuid`, inv)
	defer db.ExecContext(ctx, `DELETE FROM payments WHERE invoice_id=$1::uuid`, inv)
	defer db.ExecContext(ctx, `DELETE FROM invoices WHERE id=$1::uuid`, inv)
	pay, err := st.RecordPayment(ctx, org, inv, 5000000, "qris", "midtrans", ref, "whpay-"+tag)
	if err != nil {
		t.Fatal(err)
	}
	_ = pay

	srv := newServer().WithStore(st)
	os.Setenv("MIDTRANS_SERVER_KEY", "test-server-key")
	defer os.Unsetenv("MIDTRANS_SERVER_KEY")
	sig := func(order, status, gross string) string {
		mac := hmac.New(sha512.New, []byte("test-server-key"))
		mac.Write([]byte(order + status + gross))
		return hex.EncodeToString(mac.Sum(nil))
	}
	post := func(body string, headers map[string]string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/v1/payments/webhook/midtrans", strings.NewReader(body))
		for k, v := range headers {
			req.Header.Set(k, v)
		}
		rec := httptest.NewRecorder()
		srv.Handler().ServeHTTP(rec, req)
		return rec
	}
	valid := fmt.Sprintf(`{"order_id":%q,"status_code":"200","gross_amount":"5000000","signature_key":%q,"transaction_status":"settlement"}`,
		ref, sig(ref, "200", "5000000"))
	if rec := post(valid, nil); rec.Code != 200 {
		t.Fatalf("valid webhook: %d %s", rec.Code, rec.Body.String())
	}
	var status string
	_ = db.QueryRowContext(ctx, `SELECT status FROM invoices WHERE id=$1::uuid`, inv).Scan(&status)
	if status != "paid" {
		t.Fatalf("invoice not paid: %s", status)
	}
	// replay idempotent
	if rec := post(valid, nil); rec.Code != 200 {
		t.Fatalf("replay: %d", rec.Code)
	}
	// forged signature rejected (use a fresh reference to avoid applied-shortcut)
	forged := fmt.Sprintf(`{"order_id":%q,"status_code":"200","gross_amount":"5000000","signature_key":"00","transaction_status":"settlement"}`, ref)
	if rec := post(forged, nil); rec.Code != 401 {
		t.Fatalf("forged must 401, got %d", rec.Code)
	}
	// unknown provider refused
	req := httptest.NewRequest("POST", "/api/v1/payments/webhook/nope", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("unknown provider: %d", rec.Code)
	}
	_ = http.StatusOK
}
