package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type testResolver struct {
	tokenToTenant map[string]string
	err           error
}

func (r *testResolver) ResolveTenant(_ context.Context, token string) (string, error) {
	if r.err != nil {
		return "", r.err
	}
	tenant, ok := r.tokenToTenant[token]
	if !ok {
		return "", ErrUnauthorized
	}
	return tenant, nil
}

func TestAuthMiddleware(t *testing.T) {
	resolver := &testResolver{tokenToTenant: map[string]string{"token": "tenant1"}}

	handler := AuthMiddleware(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantFromContext(r.Context())
		require.True(t, ok)
		require.Equal(t, "tenant1", tenantID)
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestAuthMiddleware_Invalid(t *testing.T) {
	resolver := &testResolver{err: errors.New("invalid")}

	handler := AuthMiddleware(resolver)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
}
