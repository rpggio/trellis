package mcp

import (
	"context"
	"fmt"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

type contextKey int

const (
	tenantIDKey contextKey = iota
	sessionIDKey
)

// getTenantID extracts tenant ID from context.
func getTenantID(ctx context.Context) string {
	v, _ := ctx.Value(tenantIDKey).(string)
	return v
}

// getSessionID extracts session ID from context.
func getSessionID(ctx context.Context) string {
	v, _ := ctx.Value(sessionIDKey).(string)
	return v
}

// TenantResolver resolves a tenant ID from a bearer token.
type TenantResolver interface {
	ResolveTenant(ctx context.Context, token string) (string, error)
}

// authMiddleware implements bearer token authentication as MCP middleware.
func authMiddleware(resolver TenantResolver) sdkmcp.Middleware {
	return func(next sdkmcp.MethodHandler) sdkmcp.MethodHandler {
		return func(ctx context.Context, method string, req sdkmcp.Request) (sdkmcp.Result, error) {
			// Skip auth for protocol methods
			if method == "initialize" || method == "ping" {
				return next(ctx, method, req)
			}

			extra := req.GetExtra()
			if extra == nil || extra.Header == nil {
				return nil, fmt.Errorf("unauthorized: missing headers")
			}

			auth := extra.Header.Get("Authorization")
			token := strings.TrimSpace(strings.TrimPrefix(auth, "Bearer "))
			if token == "" {
				return nil, fmt.Errorf("unauthorized: missing bearer token")
			}

			tenantID, err := resolver.ResolveTenant(ctx, token)
			if err != nil {
				return nil, fmt.Errorf("unauthorized: %w", err)
			}
			if tenantID == "" {
				return nil, fmt.Errorf("unauthorized: invalid bearer token")
			}

			ctx = context.WithValue(ctx, tenantIDKey, tenantID)
			return next(ctx, method, req)
		}
	}
}

// noAuthMiddleware injects a default tenant when auth is disabled.
func noAuthMiddleware(defaultTenant string) sdkmcp.Middleware {
	return func(next sdkmcp.MethodHandler) sdkmcp.MethodHandler {
		return func(ctx context.Context, method string, req sdkmcp.Request) (sdkmcp.Result, error) {
			ctx = context.WithValue(ctx, tenantIDKey, defaultTenant)
			return next(ctx, method, req)
		}
	}
}

// sessionMiddleware extracts session ID from Mcp-Session-Id header (HTTP) or metadata (stdio).
func sessionMiddleware() sdkmcp.Middleware {
	return func(next sdkmcp.MethodHandler) sdkmcp.MethodHandler {
		return func(ctx context.Context, method string, req sdkmcp.Request) (sdkmcp.Result, error) {
			var sessionID string

			// Try HTTP header first (HTTP transport)
			extra := req.GetExtra()
			if extra != nil && extra.Header != nil {
				sessionID = extra.Header.Get("Mcp-Session-Id")
			}

			// If not in header, check metadata (stdio transport)
			// Note: Some notifications (like "initialized") have nil params,
			// so we must check carefully to avoid nil pointer dereference.
			if sessionID == "" {
				if params := req.GetParams(); params != nil {
					// Use defer/recover to safely handle cases where GetMeta
					// is called on a nil underlying value (SDK quirk)
					func() {
						defer func() { recover() }()
						if meta := params.GetMeta(); meta != nil {
							if sid, ok := meta["session_id"].(string); ok {
								sessionID = sid
							}
						}
					}()
				}
			}

			// Inject session ID into context if present
			if sessionID != "" {
				ctx = context.WithValue(ctx, sessionIDKey, sessionID)
			}

			return next(ctx, method, req)
		}
	}
}
