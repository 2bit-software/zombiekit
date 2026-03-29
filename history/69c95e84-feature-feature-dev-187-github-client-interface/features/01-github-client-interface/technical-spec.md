# Technical Specification: GitHubClient Interface and Test Stub

## Package

`internal/github/`

## File: `client.go`

```go
package github

import (
	"context"
	"time"
)

// CommentKind distinguishes between issue comments and review comments.
type CommentKind string

const (
	// CommentKindIssue represents top-level PR conversation comments.
	CommentKindIssue CommentKind = "issue"

	// CommentKindReview represents inline diff review comments.
	CommentKindReview CommentKind = "review"
)

// CreatePRInput holds parameters for creating a pull request.
type CreatePRInput struct {
	Title string
	Body  string
	Head  string
	Base  string
}

// PRComment represents a comment on a pull request.
type PRComment struct {
	ID          int64
	Author      string
	Body        string
	CreatedAt   time.Time
	Path        string // Review comments only; empty for issue comments.
	DiffHunk    string // Review comments only; empty for issue comments.
	InReplyToID int64  // Review comments only; 0 if top-level.
}

// PRSummary represents a pull request in list results.
type PRSummary struct {
	Number int
	Title  string
	Head   string
	Base   string
	Labels []string
}

// Client defines the interface for GitHub API operations.
type Client interface {
	// CreatePR creates a pull request and returns its number.
	CreatePR(ctx context.Context, input CreatePRInput) (int, error)

	// UpdatePRBody updates the description of an existing pull request.
	UpdatePRBody(ctx context.Context, prNumber int, body string) error

	// GetCommentsSince returns comments with IDs greater than afterID,
	// in chronological order. Pass afterID=0 to fetch all comments.
	GetCommentsSince(ctx context.Context, prNumber int, kind CommentKind, afterID int64) ([]PRComment, error)

	// PostCommentReply posts a comment on a PR.
	//
	// For CommentKindIssue: posts a new top-level issue comment.
	// The commentID is recorded for caller context but not used in the API call.
	//
	// For CommentKindReview: posts a threaded reply to the specified review
	// comment. commentID must be non-zero; returns ErrNotFound if zero.
	PostCommentReply(ctx context.Context, prNumber int, kind CommentKind, commentID int64, body string) (int64, error)

	// ApplyLabel adds a label to a pull request. Idempotent.
	ApplyLabel(ctx context.Context, prNumber int, label string) error

	// IsMerged returns true if the pull request has been merged.
	IsMerged(ctx context.Context, prNumber int) (bool, error)

	// IsClosed returns true if the pull request is closed without being merged.
	// Returns false for open PRs and merged PRs.
	IsClosed(ctx context.Context, prNumber int) (bool, error)

	// ListOpenPRs returns open pull requests carrying the specified label.
	ListOpenPRs(ctx context.Context, label string) ([]PRSummary, error)
}
```

## File: `errors.go`

```go
package github

import "errors"

// ErrorKind classifies GitHub API errors.
type ErrorKind int

const (
	ErrNotFound    ErrorKind = iota + 1
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
```

## File: `mock.go`

```go
package github

import (
	"context"
	"fmt"
)

var _ Client = (*MockClient)(nil)

// Call records a single method invocation on MockClient.
type Call struct {
	Method string
	Args   []any
}

// MockClient is a configurable test stub for Client.
type MockClient struct {
	CreatePRFn         func(ctx context.Context, input CreatePRInput) (int, error)
	UpdatePRBodyFn     func(ctx context.Context, prNumber int, body string) error
	GetCommentsSinceFn func(ctx context.Context, prNumber int, kind CommentKind, afterID int64) ([]PRComment, error)
	PostCommentReplyFn func(ctx context.Context, prNumber int, kind CommentKind, commentID int64, body string) (int64, error)
	ApplyLabelFn       func(ctx context.Context, prNumber int, label string) error
	IsMergedFn         func(ctx context.Context, prNumber int) (bool, error)
	IsClosedFn         func(ctx context.Context, prNumber int) (bool, error)
	ListOpenPRsFn      func(ctx context.Context, label string) ([]PRSummary, error)

	Calls []Call
}

func (m *MockClient) CreatePR(ctx context.Context, input CreatePRInput) (int, error) {
	m.Calls = append(m.Calls, Call{Method: "CreatePR", Args: []any{input}})
	if m.CreatePRFn != nil {
		return m.CreatePRFn(ctx, input)
	}
	return 0, fmt.Errorf("MockClient.CreatePR not configured")
}

func (m *MockClient) UpdatePRBody(ctx context.Context, prNumber int, body string) error {
	m.Calls = append(m.Calls, Call{Method: "UpdatePRBody", Args: []any{prNumber, body}})
	if m.UpdatePRBodyFn != nil {
		return m.UpdatePRBodyFn(ctx, prNumber, body)
	}
	return fmt.Errorf("MockClient.UpdatePRBody not configured")
}

func (m *MockClient) GetCommentsSince(ctx context.Context, prNumber int, kind CommentKind, afterID int64) ([]PRComment, error) {
	m.Calls = append(m.Calls, Call{Method: "GetCommentsSince", Args: []any{prNumber, kind, afterID}})
	if m.GetCommentsSinceFn != nil {
		return m.GetCommentsSinceFn(ctx, prNumber, kind, afterID)
	}
	return nil, fmt.Errorf("MockClient.GetCommentsSince not configured")
}

func (m *MockClient) PostCommentReply(ctx context.Context, prNumber int, kind CommentKind, commentID int64, body string) (int64, error) {
	m.Calls = append(m.Calls, Call{Method: "PostCommentReply", Args: []any{prNumber, kind, commentID, body}})
	if m.PostCommentReplyFn != nil {
		return m.PostCommentReplyFn(ctx, prNumber, kind, commentID, body)
	}
	return 0, fmt.Errorf("MockClient.PostCommentReply not configured")
}

func (m *MockClient) ApplyLabel(ctx context.Context, prNumber int, label string) error {
	m.Calls = append(m.Calls, Call{Method: "ApplyLabel", Args: []any{prNumber, label}})
	if m.ApplyLabelFn != nil {
		return m.ApplyLabelFn(ctx, prNumber, label)
	}
	return fmt.Errorf("MockClient.ApplyLabel not configured")
}

func (m *MockClient) IsMerged(ctx context.Context, prNumber int) (bool, error) {
	m.Calls = append(m.Calls, Call{Method: "IsMerged", Args: []any{prNumber}})
	if m.IsMergedFn != nil {
		return m.IsMergedFn(ctx, prNumber)
	}
	return false, fmt.Errorf("MockClient.IsMerged not configured")
}

func (m *MockClient) IsClosed(ctx context.Context, prNumber int) (bool, error) {
	m.Calls = append(m.Calls, Call{Method: "IsClosed", Args: []any{prNumber}})
	if m.IsClosedFn != nil {
		return m.IsClosedFn(ctx, prNumber)
	}
	return false, fmt.Errorf("MockClient.IsClosed not configured")
}

func (m *MockClient) ListOpenPRs(ctx context.Context, label string) ([]PRSummary, error) {
	m.Calls = append(m.Calls, Call{Method: "ListOpenPRs", Args: []any{label}})
	if m.ListOpenPRsFn != nil {
		return m.ListOpenPRsFn(ctx, label)
	}
	return nil, fmt.Errorf("MockClient.ListOpenPRs not configured")
}
```

## File: `mock_test.go`

Test structure (following `internal/linear/mock_test.go` pattern):

```go
package github

// Tests to implement:

// TestMockClient_InterfaceCompliance
//   Verify compile-time assertion + construct MockClient as Client.

// TestMockClient_ConfiguredResponse_CreatePR
//   Configure CreatePRFn to return PR number 42.
//   Call CreatePR, assert returned number == 42.
//   (Covers AC 2)

// TestMockClient_ConfiguredResponse_GetCommentsSince
//   Configure GetCommentsSinceFn to return 2 PRComment objects in order.
//   Call GetCommentsSince, assert both returned in order.
//   (Covers AC 3)

// TestMockClient_UnconfiguredMethod
//   Call GetCommentsSince on unconfigured mock.
//   Assert error contains "MockClient.GetCommentsSince not configured".

// TestMockClient_CallRecording_AllMethods
//   Configure all 8 methods with no-op functions.
//   Call each method with distinct arguments.
//   Assert Calls has 8 entries with correct Method and Args.
//   Assert context is NOT recorded in Args.
//   (Covers AC 5)

// TestMockClient_ErrorPredicates
//   Create each error kind, assert its predicate returns true
//   and all other predicates return false.

// TestMockClient_ErrorPredicates_NilAndForeign
//   Assert all predicates return false for nil and foreign errors.

// TestMockClient_ErrorUnwrap
//   Create error with a cause, assert Unwrap returns the cause,
//   assert errors.Is finds the cause in the chain.

// TestMockClient_ConfiguredError
//   Configure CreatePRFn to return NewNotFoundError.
//   Call CreatePR, assert IsNotFound(err) == true.
//   (Covers AC 4)

// TestMockClient_ConsumerWiring
//   Create a function accepting Client interface.
//   Pass MockClient, assert it compiles and works.
//   (Covers AC 1)

// TestMockClient_CallAccumulation
//   Call same method 3 times with different args.
//   Assert Calls has 3 entries with correct per-call args.
```

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Package name | `github` (not `ghclient`) | Matches `linear` package naming; yields `github.Client` |
| CommentKind type | `string` | Matches `callback.EventKind` pattern; more debuggable than `iota` |
| PR number type | `int` | Natural Go type; callers convert to `int64` at state store boundary |
| Error pattern | Typed `ErrorKind` + predicates | Exact match with `internal/linear/errors.go` |
| Mock pattern | Function fields + call recording | Exact match with `internal/linear/mock.go` |
| No doc.go | Omitted | LinearClient has no doc.go; only cmux/worktree/callback do. Can be added later. |
| No external deps | stdlib + testify only | Keeps DEV-187 focused; real HTTP client (DEV-188) may add deps |
