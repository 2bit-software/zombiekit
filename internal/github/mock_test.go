package github

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockClient_InterfaceCompliance(t *testing.T) {
	var c Client = &MockClient{}
	assert.NotNil(t, c)
}

func TestMockClient_ConfiguredResponse_CreatePR(t *testing.T) {
	m := &MockClient{
		CreatePRFn: func(_ context.Context, _ CreatePRInput) (int, error) {
			return 42, nil
		},
	}

	got, err := m.CreatePR(context.Background(), CreatePRInput{
		Title: "feat: add thing",
		Body:  "Description",
		Head:  "feature-branch",
		Base:  "main",
	})
	require.NoError(t, err)
	assert.Equal(t, 42, got)
}

func TestMockClient_ConfiguredResponse_GetCommentsSince(t *testing.T) {
	now := time.Now()
	comments := []PRComment{
		{ID: 100, Author: "alice", Body: "First", CreatedAt: now},
		{ID: 101, Author: "bob", Body: "Second", CreatedAt: now.Add(time.Minute)},
	}
	m := &MockClient{
		GetCommentsSinceFn: func(_ context.Context, _ int, _ CommentKind, _ int64) ([]PRComment, error) {
			return comments, nil
		},
	}

	got, err := m.GetCommentsSince(context.Background(), 1, CommentKindIssue, 0)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, int64(100), got[0].ID)
	assert.Equal(t, "alice", got[0].Author)
	assert.Equal(t, int64(101), got[1].ID)
	assert.Equal(t, "bob", got[1].Author)
}

func TestMockClient_UnconfiguredMethod(t *testing.T) {
	m := &MockClient{}

	_, err := m.GetCommentsSince(context.Background(), 1, CommentKindIssue, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MockClient.GetCommentsSince not configured")
}

func TestMockClient_CallRecording_AllMethods(t *testing.T) {
	ctx := context.Background()
	m := &MockClient{
		CreatePRFn:         func(_ context.Context, _ CreatePRInput) (int, error) { return 0, nil },
		UpdatePRBodyFn:     func(_ context.Context, _ int, _ string) error { return nil },
		GetCommentsSinceFn: func(_ context.Context, _ int, _ CommentKind, _ int64) ([]PRComment, error) { return nil, nil },
		PostCommentReplyFn: func(_ context.Context, _ int, _ CommentKind, _ int64, _ string) (int64, error) { return 0, nil },
		ApplyLabelFn:       func(_ context.Context, _ int, _ string) error { return nil },
		IsMergedFn:         func(_ context.Context, _ int) (bool, error) { return false, nil },
		IsClosedFn:         func(_ context.Context, _ int) (bool, error) { return false, nil },
		ListOpenPRsFn:      func(_ context.Context, _ string) ([]PRSummary, error) { return nil, nil },
	}

	input := CreatePRInput{Title: "PR", Body: "body", Head: "feat", Base: "main"}

	m.CreatePR(ctx, input)
	m.UpdatePRBody(ctx, 1, "new body")
	m.GetCommentsSince(ctx, 2, CommentKindReview, int64(50))
	m.PostCommentReply(ctx, 3, CommentKindIssue, int64(0), "reply text")
	m.ApplyLabel(ctx, 4, "bug")
	m.IsMerged(ctx, 5)
	m.IsClosed(ctx, 6)
	m.ListOpenPRs(ctx, "agent:pending")

	require.Len(t, m.Calls, 8)

	tests := []struct {
		method string
		args   []any
	}{
		{"CreatePR", []any{input}},
		{"UpdatePRBody", []any{1, "new body"}},
		{"GetCommentsSince", []any{2, CommentKindReview, int64(50)}},
		{"PostCommentReply", []any{3, CommentKindIssue, int64(0), "reply text"}},
		{"ApplyLabel", []any{4, "bug"}},
		{"IsMerged", []any{5}},
		{"IsClosed", []any{6}},
		{"ListOpenPRs", []any{"agent:pending"}},
	}

	for i, tt := range tests {
		assert.Equal(t, tt.method, m.Calls[i].Method, "call %d method", i)
		assert.Equal(t, tt.args, m.Calls[i].Args, "call %d args", i)
	}
}

func TestMockClient_ErrorPredicates(t *testing.T) {
	tests := []struct {
		name          string
		err           *Error
		isNotFound    bool
		isRateLimited bool
		isAPI         bool
		isNetwork     bool
	}{
		{
			name:       "NotFound",
			err:        NewNotFoundError("not found", nil),
			isNotFound: true,
		},
		{
			name:          "RateLimited",
			err:           NewRateLimitedError("slow down", nil),
			isRateLimited: true,
		},
		{
			name:  "API",
			err:   NewAPIError("server error", nil),
			isAPI: true,
		},
		{
			name:      "Network",
			err:       NewNetworkError("connection refused", nil),
			isNetwork: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isNotFound, IsNotFound(tt.err), "IsNotFound")
			assert.Equal(t, tt.isRateLimited, IsRateLimited(tt.err), "IsRateLimited")
			assert.Equal(t, tt.isAPI, IsAPIError(tt.err), "IsAPIError")
			assert.Equal(t, tt.isNetwork, IsNetworkError(tt.err), "IsNetworkError")
		})
	}
}

func TestMockClient_ErrorPredicates_NilAndForeign(t *testing.T) {
	assert.False(t, IsNotFound(nil))
	assert.False(t, IsRateLimited(nil))
	assert.False(t, IsAPIError(nil))
	assert.False(t, IsNetworkError(nil))

	foreign := errors.New("unrelated error")
	assert.False(t, IsNotFound(foreign))
	assert.False(t, IsRateLimited(foreign))
	assert.False(t, IsAPIError(foreign))
	assert.False(t, IsNetworkError(foreign))
}

func TestMockClient_ErrorUnwrap(t *testing.T) {
	cause := errors.New("connection timeout")
	err := NewNetworkError("github api unreachable", cause)

	assert.Equal(t, "github api unreachable", err.Error())
	assert.Equal(t, cause, errors.Unwrap(err))
	assert.True(t, errors.Is(err, cause))
}

func TestMockClient_ConfiguredError(t *testing.T) {
	m := &MockClient{
		CreatePRFn: func(_ context.Context, _ CreatePRInput) (int, error) {
			return 0, NewNotFoundError("repo not found", nil)
		},
	}

	prNum, err := m.CreatePR(context.Background(), CreatePRInput{})
	assert.Equal(t, 0, prNum)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.False(t, IsRateLimited(err))
}

func TestMockClient_ConsumerWiring(t *testing.T) {
	m := &MockClient{
		ListOpenPRsFn: func(_ context.Context, _ string) ([]PRSummary, error) {
			return []PRSummary{{Number: 10, Title: "Test PR"}}, nil
		},
	}

	listAndCount := func(c Client, label string) (int, error) {
		prs, err := c.ListOpenPRs(context.Background(), label)
		if err != nil {
			return 0, err
		}
		return len(prs), nil
	}

	count, err := listAndCount(m, "agent:pending")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Len(t, m.Calls, 1)
	assert.Equal(t, "ListOpenPRs", m.Calls[0].Method)
}

func TestMockClient_CallAccumulation(t *testing.T) {
	m := &MockClient{
		ApplyLabelFn: func(_ context.Context, _ int, _ string) error { return nil },
	}

	m.ApplyLabel(context.Background(), 1, "bug")
	m.ApplyLabel(context.Background(), 2, "enhancement")
	m.ApplyLabel(context.Background(), 3, "agent:done")

	require.Len(t, m.Calls, 3)
	assert.Equal(t, 1, m.Calls[0].Args[0])
	assert.Equal(t, 2, m.Calls[1].Args[0])
	assert.Equal(t, 3, m.Calls[2].Args[0])
}
