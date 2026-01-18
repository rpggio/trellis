package transport

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// MCPHandler handles MCP method dispatch.
type MCPHandler interface {
	Handle(ctx context.Context, tenantID, sessionID, method string, params json.RawMessage) (any, error)
}

// Server wires HTTP handlers.
type Server struct {
	handler MCPHandler
}

// NewServer creates an HTTP server router with middleware.
func NewServer(handler MCPHandler, authMiddleware func(http.Handler) http.Handler) *chi.Mux {
	r := chi.NewRouter()

	if authMiddleware != nil {
		r.Use(authMiddleware)
	}
	r.Use(SessionMiddleware)

	srv := &Server{handler: handler}

	r.Post("/mcp", srv.handleMCP)
	r.Get("/health", srv.handleHealth)

	return r
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	req, err := ParseRequest(r.Body)
	if err != nil {
		WriteError(w, nil, ErrInvalidReq, "invalid request", nil)
		return
	}

	tenantID, ok := TenantFromContext(r.Context())
	if !ok || tenantID == "" {
		http.Error(w, "missing tenant", http.StatusUnauthorized)
		return
	}

	sessionID, _ := SessionIDFromContext(r.Context())

	result, err := s.handler.Handle(r.Context(), tenantID, sessionID, req.Method, req.Params)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		WriteError(w, req.ID, ErrInternal, err.Error(), nil)
		return
	}

	WriteResult(w, req.ID, result)
}
