package transport

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// ErrUnauthorized indicates invalid or missing credentials.
var ErrUnauthorized = errors.New("unauthorized")

type tenantKey struct{}

// TenantResolver resolves a tenant ID from a bearer token.
type TenantResolver interface {
	ResolveTenant(ctx context.Context, token string) (string, error)
}

// TenantFromContext returns the tenant ID from context, if present.
func TenantFromContext(ctx context.Context) (string, bool) {
	tenantID, ok := ctx.Value(tenantKey{}).(string)
	return tenantID, ok
}

// AuthMiddleware enforces bearer token authentication.
func AuthMiddleware(resolver TenantResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			if token == "" {
				http.Error(w, "missing bearer token", http.StatusUnauthorized)
				return
			}

			tenantID, err := resolver.ResolveTenant(r.Context(), token)
			if err != nil || tenantID == "" {
				http.Error(w, "invalid bearer token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), tenantKey{}, tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
