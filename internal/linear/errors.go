package linear

import "errors"

// ErrorKind classifies Linear API errors.
type ErrorKind int

const (
	ErrNotFound ErrorKind = iota + 1
	ErrRateLimited
	ErrAPI
	ErrNetwork
)

// Error represents a Linear API error with classification.
type Error struct {
	Kind    ErrorKind
	Message string
	Err     error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }

func NewNotFoundError(msg string, cause error) *Error {
	return &Error{Kind: ErrNotFound, Message: msg, Err: cause}
}

func NewRateLimitedError(msg string, cause error) *Error {
	return &Error{Kind: ErrRateLimited, Message: msg, Err: cause}
}

func NewAPIError(msg string, cause error) *Error {
	return &Error{Kind: ErrAPI, Message: msg, Err: cause}
}

func NewNetworkError(msg string, cause error) *Error {
	return &Error{Kind: ErrNetwork, Message: msg, Err: cause}
}

func IsNotFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrNotFound
}

func IsRateLimited(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrRateLimited
}

func IsAPIError(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrAPI
}

func IsNetworkError(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrNetwork
}
