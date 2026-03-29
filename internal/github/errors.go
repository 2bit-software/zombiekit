package github

import "errors"

// ErrorKind classifies GitHub API errors.
type ErrorKind int

const (
	ErrNotFound ErrorKind = iota + 1
	ErrRateLimited
	ErrAPI
	ErrNetwork
)

// Error represents a GitHub API error with classification.
type Error struct {
	Kind    ErrorKind
	Message string
	Err     error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }

// NewNotFoundError creates an error indicating a resource was not found.
func NewNotFoundError(msg string, cause error) *Error {
	return &Error{Kind: ErrNotFound, Message: msg, Err: cause}
}

// NewRateLimitedError creates an error indicating rate limiting.
func NewRateLimitedError(msg string, cause error) *Error {
	return &Error{Kind: ErrRateLimited, Message: msg, Err: cause}
}

// NewAPIError creates an error for non-success API responses.
func NewAPIError(msg string, cause error) *Error {
	return &Error{Kind: ErrAPI, Message: msg, Err: cause}
}

// NewNetworkError creates an error for connection failures.
func NewNetworkError(msg string, cause error) *Error {
	return &Error{Kind: ErrNetwork, Message: msg, Err: cause}
}

// IsNotFound returns true if the error is a not-found error.
func IsNotFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrNotFound
}

// IsRateLimited returns true if the error is a rate-limit error.
func IsRateLimited(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrRateLimited
}

// IsAPIError returns true if the error is an API error.
func IsAPIError(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrAPI
}

// IsNetworkError returns true if the error is a network error.
func IsNetworkError(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrNetwork
}
