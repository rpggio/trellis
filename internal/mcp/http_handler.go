package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
)

// NewHTTPHandler creates a simple HTTP handler for JSON-RPC requests.
// This provides compatibility with existing tests while using the SDK server.
func NewHTTPHandler(server *sdkmcp.Server, logger *slog.Logger) http.Handler {
	return &httpHandler{
		server: server,
		logger: logger,
	}
}

type httpHandler struct {
	server *sdkmcp.Server
	logger *slog.Logger
}

type jsonrpcRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  any             `json:"result,omitempty"`
	Error   *jsonrpcError   `json:"error,omitempty"`
	ID      any             `json:"id,omitempty"`
}

type jsonrpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.writeError(w, -32700, "Parse error", nil, nil)
		return
	}

	var req jsonrpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		h.writeError(w, -32700, "Parse error", nil, nil)
		return
	}

	// Use in-memory transport to call the SDK server
	t1, t2 := sdkmcp.NewInMemoryTransports()

	// Connect the server
	session, err := h.server.Connect(r.Context(), t1, nil)
	if err != nil {
		h.writeError(w, -32603, fmt.Sprintf("Internal error: %v", err), nil, req.ID)
		return
	}
	defer session.Close()

	// Send the request through the transport
	if err := t2.Send(r.Context(), sdkmcp.JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  req.Method,
		Params:  req.Params,
		ID:      req.ID,
	}); err != nil {
		h.writeError(w, -32603, fmt.Sprintf("Internal error: %v", err), nil, req.ID)
		return
	}

	// Receive the response
	msg, err := t2.Receive(r.Context())
	if err != nil {
		h.writeError(w, -32603, fmt.Sprintf("Internal error: %v", err), nil, req.ID)
		return
	}

	// Write the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonrpcResponse{
		JSONRPC: "2.0",
		Result:  msg.Result,
		Error:   convertSDKError(msg.Error),
		ID:      msg.ID,
	})
}

func (h *httpHandler) writeError(w http.ResponseWriter, code int, message string, data any, id any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // JSON-RPC errors are still 200 OK
	json.NewEncoder(w).Encode(jsonrpcResponse{
		JSONRPC: "2.0",
		Error: &jsonrpcError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	})
}

func convertSDKError(err *sdkmcp.JSONRPCError) *jsonrpcError {
	if err == nil {
		return nil
	}
	return &jsonrpcError{
		Code:    err.Code,
		Message: err.Message,
		Data:    err.Data,
	}
}
