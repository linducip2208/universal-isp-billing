package middleware

import (
	"context"
	"net/http"

	"github.com/universal-isp/platform/internal/rbac"
)

// KeyVerifier resolves X-API-Key into org + scopes (store-backed in prod).
type KeyVerifier interface {
	VerifyAPIKey(ctx context.Context, raw string) (keyID, orgID string, scopes []string, err error)
}

// APIKey authenticates service accounts via X-API-Key alongside/instead of
// Bearer JWT. On success it injects org + scopes; otherwise the request
// continues anonymously (JWT layer decides 401).
func APIKey(v KeyVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw := r.Header.Get("X-API-Key")
			if raw == "" || v == nil {
				next.ServeHTTP(w, r)
				return
			}
			_, org, scopes, err := v.VerifyAPIKey(r.Context(), raw)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			ctx := rbac.WithOrg(r.Context(), org)
			ctx = rbac.WithScopes(ctx, scopes)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
