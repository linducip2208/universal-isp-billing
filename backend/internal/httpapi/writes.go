package httpapi

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/universal-isp/platform/internal/auth"
	"github.com/universal-isp/platform/internal/billing"
	"github.com/universal-isp/platform/internal/customers"
	"github.com/universal-isp/platform/internal/middleware"
	"github.com/universal-isp/platform/internal/rbac"
	"github.com/universal-isp/platform/internal/store"
	"github.com/universal-isp/platform/internal/vouchers"
)

func mustStore(s *Server, w http.ResponseWriter, r *http.Request) bool {
	if s.store == nil {
		middleware.Error(w, http.StatusServiceUnavailable, "database not configured", r.Header.Get("X-Request-ID"))
		return false
	}
	return true
}

func (s *Server) handleCreateCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !mustStore(s, w, r) {
		return
	}
	var body struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	id, err := s.store.CreateCustomer(r.Context(), rbac.OrgOf(r.Context()),
		customers.Customer{Name: body.Name, Email: body.Email, Phone: body.Phone})
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]string{"id": id})
}

func (s *Server) handleCreateSubscription(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !mustStore(s, w, r) {
		return
	}
	var body struct {
		CustomerID string `json:"customer_id"`
		PackageID  string `json:"package_id"`
		Username   string `json:"username"`
		Service    string `json:"service"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	id, err := s.store.CreateSubscription(r.Context(), rbac.OrgOf(r.Context()),
		body.CustomerID, body.PackageID, body.Username, body.Service)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]string{"id": id})
}

func (s *Server) handleCreateInvoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !mustStore(s, w, r) {
		return
	}
	var body struct {
		CustomerID     string              `json:"customer_id"`
		SubscriptionID string              `json:"subscription_id"`
		Lines          []store.InvoiceLine `json:"lines"`
		TaxBps         int64               `json:"tax_bps"`
		DueDays        int                 `json:"due_days"`
		GraceDays      int                 `json:"grace_days"`
		IdemKey        string              `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	due := time.Now().AddDate(0, 0, body.DueDays+1)
	id, err := s.store.CreateInvoice(r.Context(), rbac.OrgOf(r.Context()),
		body.CustomerID, body.SubscriptionID, body.Lines, body.TaxBps, due, body.GraceDays, body.IdemKey)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]string{"id": id})
}

func (s *Server) handleCreatePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !mustStore(s, w, r) {
		return
	}
	var body struct {
		InvoiceID string `json:"invoice_id"`
		Amount    int64  `json:"amount_cents"`
		Method    string `json:"method"`
		Provider  string `json:"provider"`
		Reference string `json:"reference"`
		IdemKey   string `json:"idempotency_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	id, err := s.store.RecordPayment(r.Context(), rbac.OrgOf(r.Context()),
		body.InvoiceID, body.Amount, body.Method, body.Provider, body.Reference, body.IdemKey)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 201, map[string]string{"id": id})
}

// handlePaymentWebhook settles PUBLIC provider callbacks. Each provider must
// verify its signature from server-side secrets; missing secrets or unknown
// providers are refused (never applied blindly). Resolved tenant comes from
// the payment reference itself.
func (s *Server) handlePaymentWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !mustStore(s, w, r) {
		return
	}
	provider := r.PathValue("provider")
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": "bad body"})
		return
	}
	headers := map[string]string{}
	for k := range r.Header {
		headers[k] = r.Header.Get(k)
	}
	lower := map[string]string{}
	for k, v := range headers {
		lower[strings.ToLower(k)] = v
	}
	var event *billing.WebhookEvent
	switch provider {
	case "xendit":
		secret := os.Getenv("XENDIT_CALLBACK_TOKEN")
		if secret == "" {
			writeJSON(w, 503, map[string]string{"error": "xendit not configured"})
			return
		}
		lower["x-callback-token"] = r.Header.Get("X-Callback-Token")
		xp := billing.XenditProvider{SecretKey: secret}
		if event, err = xp.VerifyWebhook(r.Context(), raw, lower); err != nil {
			writeJSON(w, 401, map[string]string{"error": "webhook verification failed"})
			return
		}
	case "midtrans":
		secret := os.Getenv("MIDTRANS_SERVER_KEY")
		if secret == "" {
			writeJSON(w, 503, map[string]string{"error": "midtrans not configured"})
			return
		}
		for _, h := range []string{"order_id", "status_code", "gross_amount", "signature_key", "transaction_status"} {
			lower[h] = r.Header.Get(h)
		}
		// Midtrans also posts form fields; merge JSON body values.
		var f map[string]string
		_ = json.Unmarshal(raw, &f)
		for k, v := range f {
			if _, ok := lower[k]; !ok || lower[k] == "" {
				lower[k] = v
			}
		}
		if event, err = (billing.MidtransProvider{ServerKey: secret}).VerifyWebhook(r.Context(), raw, lower); err != nil {
			writeJSON(w, 401, map[string]string{"error": "webhook verification failed"})
			return
		}
	default:
		writeJSON(w, 400, map[string]string{"error": "unknown provider (manual payments use authenticated POST /payments)"})
		return
	}
	ref := event.Reference
	if ref == "" {
		var f map[string]any
		_ = json.Unmarshal(raw, &f)
		ref, _ = f["order_id"].(string)
		if ref == "" {
			ref, _ = f["external_id"].(string)
		}
	}
	if ref == "" {
		writeJSON(w, 400, map[string]string{"error": "no payment reference in callback"})
		return
	}
	org, err := s.store.LookupPaymentOrg(r.Context(), provider, ref)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": "unknown payment reference"})
		return
	}
	id, err := s.store.ApplyWebhook(r.Context(), org, provider, ref, event.Paid)
	if err != nil {
		writeJSON(w, 404, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"payment_id": id})
}

func (s *Server) handleRadiusUsers(w http.ResponseWriter, r *http.Request) {
	if !mustStore(s, w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleResource("radius_users")(w, r)
	case http.MethodPost:
		var body struct {
			Username    string `json:"username"`
			Password    string `json:"password"`
			ProfileID   string `json:"profile_id"`
			MaxSessions int    `json:"max_sessions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		id, err := s.store.CreateRadiusUser(r.Context(), rbac.OrgOf(r.Context()),
			body.Username, body.Password, body.ProfileID, body.MaxSessions)
		if err != nil {
			writeJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 201, map[string]string{"id": id})
	case http.MethodDelete:
		username := r.URL.Query().Get("username")
		if username == "" {
			writeJSON(w, 400, map[string]string{"error": "?username= required"})
			return
		}
		if err := s.store.DisableRadiusUser(r.Context(), rbac.OrgOf(r.Context()), username); err != nil {
			writeJSON(w, 404, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]string{"status": "disabled"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handlePasswordChange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cl := auth.ClaimsOf(r.Context())
	if cl == nil || s.store == nil {
		writeJSON(w, 401, map[string]string{"error": "invalid token"})
		return
	}
	var body struct {
		Old string `json:"old_password"`
		New string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Old == "" || body.New == "" {
		writeJSON(w, 400, map[string]string{"error": "old_password and new_password required"})
		return
	}
	// Resolve user id from subject (username) within org.
	var userID string
	_ = userID
	if err := s.store.ChangePasswordByUsername(r.Context(), rbac.OrgOf(r.Context()), cl.Subject, body.Old, body.New); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "password changed"})
}

func (s *Server) handleVouchers(w http.ResponseWriter, r *http.Request) {
	if !mustStore(s, w, r) {
		return
	}
	switch r.Method {
	case http.MethodGet:
		s.handleResource("vouchers")(w, r)
	case http.MethodPost:
		var body struct {
			Count     int    `json:"count"`
			PackageID string `json:"package_id"`
			Hours     int    `json:"hours"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.Count < 1 || body.Count > 500 {
			writeJSON(w, 400, map[string]string{"error": "count 1..500"})
			return
		}
		if body.Hours <= 0 {
			body.Hours = 24
		}
		var codes []string
		for i := 0; i < body.Count; i++ {
			code, err := vouchers.GenerateCode(8)
			if err != nil {
				writeJSON(w, 500, map[string]string{"error": err.Error()})
				return
			}
			if _, err := s.store.CreateVoucher(r.Context(), rbac.OrgOf(r.Context()), code, body.PackageID, body.Hours, nil); err != nil {
				writeJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			codes = append(codes, code)
		}
		writeJSON(w, 201, map[string]any{"codes": codes})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleVoucherRedeem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !mustStore(s, w, r) {
		return
	}
	var body struct {
		Code   string `json:"code"`
		UsedBy string `json:"used_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Code == "" {
		writeJSON(w, 400, map[string]string{"error": "code required"})
		return
	}
	if err := s.store.RedeemVoucher(r.Context(), rbac.OrgOf(r.Context()), body.Code, body.UsedBy); err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, 200, map[string]string{"status": "redeemed"})
}
