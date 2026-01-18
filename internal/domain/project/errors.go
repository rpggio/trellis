package project

import "errors"

var (
	// ErrProjectNotFound indicates the project doesn't exist.
	ErrProjectNotFound = errors.New("project not found")
	// ErrInvalidInput indicates invalid project input.
	ErrInvalidInput = errors.New("invalid project input")
)
