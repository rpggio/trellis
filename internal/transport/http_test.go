package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type testHandler struct {
	method string
}

func (h *testHandler) Handle(_ context.Context, tenantID, sessionID, method string, params json.RawMessage) (any, error) {
	h.method = method
	return map[string]string{"tenant": tenantID, "session": sessionID}, nil
}

type staticResolver struct {
	tenant string
}

func (r *staticResolver) ResolveTenant(_ context.Context, token string) (string, error) {
	if token == "" {
		return "", ErrUnauthorized
	}
	return r.tenant, nil
}

func TestHTTPServer_MCP(t *testing.T) {
	handler := &testHandler{}
	resolver := &staticResolver{tenant: "tenant1"}
	server := httptest.NewServer(NewServer(handler, AuthMiddleware(resolver)))
	t.Cleanup(server.Close)

	body := bytes.NewBufferString(`{"jsonrpc":"2.0","method":"list_projects","id":1}`)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/mcp", body)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer token")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Mcp-Session-Id", "sess1")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "list_projects", handler.method)
}

func TestHTTPServer_Health(t *testing.T) {
	handler := &testHandler{}
	server := httptest.NewServer(NewServer(handler, nil))
	t.Cleanup(server.Close)

	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
}
