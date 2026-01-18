package transport

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// JSON-RPC 2.0 error codes.
const (
	ErrParseCode      = -32700
	ErrInvalidReq     = -32600
	ErrMethodNotFound = -32601
	ErrInvalidParams  = -32602
	ErrInternal       = -32603
)

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
	ID      any    `json:"id,omitempty"`
}

// Error represents a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// ParseRequest parses and validates a JSON-RPC request payload.
func ParseRequest(body io.Reader) (Request, error) {
	var req Request
	dec := json.NewDecoder(body)
	if err := dec.Decode(&req); err != nil {
		return Request{}, fmt.Errorf("parse error: %w", err)
	}
	if req.JSONRPC != "2.0" || req.Method == "" {
		return Request{}, fmt.Errorf("invalid request")
	}
	return req, nil
}

// WriteResult writes a JSON-RPC success response.
func WriteResult(w http.ResponseWriter, id any, result any) {
	writeJSON(w, http.StatusOK, Response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	})
}

// WriteError writes a JSON-RPC error response.
func WriteError(w http.ResponseWriter, id any, code int, message string, data any) {
	writeJSON(w, http.StatusOK, Response{
		JSONRPC: "2.0",
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload Response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
