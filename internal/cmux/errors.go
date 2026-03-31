package cmux

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorKind classifies cmux session manager errors.
type ErrorKind int

const (
	ErrSessionExists   ErrorKind = iota + 1
	ErrSessionNotFound
	ErrCmuxUnavailable
	ErrBinaryNotFound
	ErrCommandFailed
	ErrInvalidEnvKey
)

// Error is a classified cmux session manager error.
type Error struct {
	Kind    ErrorKind
	Message string
	Err     error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }

func newError(kind ErrorKind, msg string, err error) *Error {
	return &Error{Kind: kind, Message: msg, Err: err}
}

func newErrorf(kind ErrorKind, err error, format string, args ...any) *Error {
	return &Error{Kind: kind, Message: fmt.Sprintf(format, args...), Err: err}
}

// classifyError maps cmux stderr output to an ErrorKind.
func classifyError(stderr string) ErrorKind {
	switch {
	case strings.Contains(stderr, "not_found"):
		return ErrSessionNotFound
	case strings.Contains(stderr, "connection refused"),
		strings.Contains(stderr, "No such file"),
		strings.Contains(stderr, "could not connect"):
		return ErrCmuxUnavailable
	default:
		return ErrCommandFailed
	}
}

// IsSessionExists reports whether err is a session-already-exists error.
func IsSessionExists(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrSessionExists
}

// IsSessionNotFound reports whether err is a session-not-found error.
func IsSessionNotFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrSessionNotFound
}

// IsInvalidEnvKey reports whether err is an invalid-env-key error.
func IsInvalidEnvKey(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrInvalidEnvKey
}
