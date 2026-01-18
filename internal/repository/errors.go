package repository

import "errors"

var (
	// ErrNotFound is returned when a requested entity doesn't exist
	ErrNotFound = errors.New("not found")

	// ErrConflict is returned when an optimistic concurrency check fails
	ErrConflict = errors.New("conflict: entity was modified by another session")

	// ErrForeignKeyViolation is returned when a foreign key constraint fails
	ErrForeignKeyViolation = errors.New("foreign key violation")

	// ErrInvalidInput is returned when input validation fails
	ErrInvalidInput = errors.New("invalid input")
)
