package mcp

import (
	"errors"
	"fmt"

	"github.com/ganot/threds-mcp/internal/domain/record"
	"github.com/ganot/threds-mcp/internal/domain/session"
)

// APIError represents an MCP error response.
type APIError struct {
	Code         string `json:"code"`
	Message      string `json:"message"`
	Details      any    `json:"details,omitempty"`
	RecoveryHint string `json:"recovery_hint,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *APIError) CodeValue() string {
	return e.Code
}

func (e *APIError) MessageValue() string {
	return e.Message
}

func (e *APIError) DetailsValue() any {
	return e.Details
}

func (e *APIError) RecoveryHintValue() string {
	return e.RecoveryHint
}

// MapError maps domain errors to MCP error codes.
func MapError(err error) *APIError {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, record.ErrRecordNotFound):
		return &APIError{Code: "RECORD_NOT_FOUND", Message: "record not found", RecoveryHint: "Check ID spelling"}
	case errors.Is(err, record.ErrNotActivated):
		return &APIError{Code: "NOT_ACTIVATED", Message: "record not activated", RecoveryHint: "Call activate() first"}
	case errors.Is(err, record.ErrParentNotActivated):
		return &APIError{Code: "PARENT_NOT_ACTIVATED", Message: "parent not activated", RecoveryHint: "Activate parent first"}
	case errors.Is(err, record.ErrInvalidTransition):
		return &APIError{Code: "INVALID_TRANSITION", Message: "invalid state transition", RecoveryHint: "Check valid transitions"}
	case errors.Is(err, record.ErrConflict):
		return &APIError{Code: "CONFLICT", Message: "record modified by another session", RecoveryHint: "Sync and resolve"}
	case errors.Is(err, session.ErrSessionNotFound):
		return &APIError{Code: "SESSION_NOT_FOUND", Message: "session not found", RecoveryHint: "Start a new session"}
	default:
		return nil
	}
}
