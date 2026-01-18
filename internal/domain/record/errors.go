package record

import "errors"

var (
	// ErrRecordNotFound indicates the record doesn't exist.
	ErrRecordNotFound = errors.New("record not found")
	// ErrNotActivated indicates the record is not activated in the session.
	ErrNotActivated = errors.New("record not activated")
	// ErrParentNotActivated indicates the parent record is not activated.
	ErrParentNotActivated = errors.New("parent record not activated")
	// ErrInvalidTransition indicates an invalid workflow transition.
	ErrInvalidTransition = errors.New("invalid record state transition")
	// ErrMissingReason indicates a reason is required for the transition.
	ErrMissingReason = errors.New("reason required for state transition")
	// ErrMissingResolvedBy indicates resolved_by is required for transition.
	ErrMissingResolvedBy = errors.New("resolved_by required for state transition")
	// ErrConflict indicates a record was updated since activation.
	ErrConflict = errors.New("record modified since activation")
	// ErrInvalidInput indicates invalid input for record operations.
	ErrInvalidInput = errors.New("invalid record input")
)
