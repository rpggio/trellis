package transport

import (
	"context"
	"net/http"
)

type sessionKey struct{}

// SessionIDFromContext returns the session ID from context, if present.
func SessionIDFromContext(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(sessionKey{}).(string)
	return sessionID, ok
}

// SessionMiddleware extracts Mcp-Session-Id and stores it in context.
func SessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.Header.Get("Mcp-Session-Id")
		if sessionID != "" {
			ctx := context.WithValue(r.Context(), sessionKey{}, sessionID)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}
		next.ServeHTTP(w, r)
	})
}
