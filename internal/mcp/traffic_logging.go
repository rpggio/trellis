package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

func trafficLoggingMiddleware(logger *slog.Logger, direction string) sdkmcp.Middleware {
	return func(next sdkmcp.MethodHandler) sdkmcp.MethodHandler {
		return func(ctx context.Context, method string, req sdkmcp.Request) (sdkmcp.Result, error) {
			if logger == nil || !logger.Enabled(ctx, slog.LevelDebug) {
				return next(ctx, method, req)
			}

			sessionID := safeSessionID(req)
			tenantID := getTenantID(ctx)
			params := formatPayload(safeParams(req))
			logger.Debug("mcp traffic", "direction", direction, "stage", "request", "method", method, "session_id", sessionID, "tenant_id", tenantID, "params", params)

			result, err := next(ctx, method, req)
			if !strings.HasPrefix(method, "notifications/") {
				if err != nil {
					logger.Debug("mcp traffic", "direction", direction, "stage", "response", "method", method, "session_id", sessionID, "tenant_id", tenantID, "result", formatPayload(result), "error", err)
				} else {
					logger.Debug("mcp traffic", "direction", direction, "stage", "response", "method", method, "session_id", sessionID, "tenant_id", tenantID, "result", formatPayload(result))
				}
			}

			return result, err
		}
	}
}

func safeSessionID(req sdkmcp.Request) string {
	if req == nil {
		return ""
	}
	defer func() { recover() }()
	session := req.GetSession()
	if session == nil {
		return ""
	}
	defer func() { recover() }()
	return session.ID()
}

func safeParams(req sdkmcp.Request) any {
	if req == nil {
		return nil
	}
	defer func() { recover() }()
	return req.GetParams()
}

func formatPayload(payload any) string {
	if payload == nil {
		return "<nil>"
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("%T", payload)
	}
	return string(data)
}
