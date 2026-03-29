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
