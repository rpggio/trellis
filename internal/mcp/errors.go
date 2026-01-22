package mcp

import (
	"errors"
	"fmt"

	"github.com/rpggio/trellis/internal/domain/record"
	"github.com/rpggio/trellis/internal/domain/session"
)

// mapError converts domain errors to SDK-compatible errors.
// The SDK automatically wraps these in CallToolResult with IsError=true.
func mapError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, record.ErrRecordNotFound):
		return fmt.Errorf("RECORD_NOT_FOUND: record not found (hint: check ID spelling)")
	case errors.Is(err, record.ErrNotActivated):
		return fmt.Errorf("NOT_ACTIVATED: record not activated (hint: call activate first)")
	case errors.Is(err, record.ErrParentNotActivated):
		return fmt.Errorf("PARENT_NOT_ACTIVATED: parent not activated (hint: activate parent first)")
	case errors.Is(err, record.ErrInvalidTransition):
		return fmt.Errorf("INVALID_TRANSITION: invalid state transition (hint: check valid transitions)")
	case errors.Is(err, record.ErrConflict):
		return fmt.Errorf("CONFLICT: record modified by another session (hint: sync and resolve)")
	case errors.Is(err, session.ErrSessionNotFound):
		return fmt.Errorf("SESSION_NOT_FOUND: session not found (hint: start a new session)")
	default:
		return err
	}
}
