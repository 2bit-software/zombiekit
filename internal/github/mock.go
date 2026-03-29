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
