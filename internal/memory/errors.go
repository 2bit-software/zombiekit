// Package memory provides persistent memory storage functionality.
package memory

import "errors"

// Common errors for memory operations.
var (
	// ErrNotFound is returned when a memory item is not found.
	ErrNotFound = errors.New("memory not found")

	// ErrNameTooLong is returned when the name exceeds MaxNameLength.
	ErrNameTooLong = errors.New("name exceeds maximum length")

	// ErrContentTooLarge is returned when content exceeds MaxContentSize.
	ErrContentTooLarge = errors.New("content exceeds maximum size (1MB)")

	// ErrInvalidBackend is returned when an unknown backend is specified.
	ErrInvalidBackend = errors.New("invalid storage backend")
)
