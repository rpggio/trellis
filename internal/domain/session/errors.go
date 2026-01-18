package session

import "errors"

var (
	// ErrSessionNotFound indicates the session doesn't exist.
	ErrSessionNotFound = errors.New("session not found")
	// ErrRecordNotFound indicates the target record doesn't exist.
	ErrRecordNotFound = errors.New("record not found")
	// ErrInvalidInput indicates invalid session input.
	ErrInvalidInput = errors.New("invalid session input")
)
