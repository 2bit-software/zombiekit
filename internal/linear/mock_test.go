package linear

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockClient_InterfaceCompliance(t *testing.T) {
	// Compile-time assertion is in mock.go: var _ Client = (*MockClient)(nil)
	// This test just verifies the package builds.
	var c Client = &MockClient{}
	assert.NotNil(t, c)
}

func TestMockClient_ConfiguredResponse_PollReadyTickets(t *testing.T) {
	tickets := []Ticket{
		{ID: "1", Identifier: "DEV-100", Title: "First"},
		{ID: "2", Identifier: "DEV-101", Title: "Second"},
	}
	m := &MockClient{
		PollReadyTicketsFn: func(_ context.Context, _ string) ([]Ticket, error) {
			return tickets, nil
		},
	}

	got, err := m.PollReadyTickets(context.Background(), "ready")
	require.NoError(t, err)
	assert.Len(t, got, 2)
	assert.Equal(t, "DEV-100", got[0].Identifier)
	assert.Equal(t, "DEV-101", got[1].Identifier)
}

func TestMockClient_ConfiguredResponse_GetTicket(t *testing.T) {
	expected := &Ticket{
		ID:         "abc-123",
		Identifier: "DEV-156",
		Title:      "Define LinearClient",
		Status:     "In Progress",
		Labels:     []string{"backend"},
		Priority:   2,
	}
	m := &MockClient{
		GetTicketFn: func(_ context.Context, _ string) (*Ticket, error) {
			return expected, nil
		},
	}

	got, err := m.GetTicket(context.Background(), "abc-123")
	require.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestMockClient_UnconfiguredMethod(t *testing.T) {
	m := &MockClient{}

	_, err := m.GetTicket(context.Background(), "any-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "MockClient.GetTicket not configured")
}

func TestMockClient_CallRecording_AllMethods(t *testing.T) {
	ctx := context.Background()
	m := &MockClient{
		PollReadyTicketsFn: func(_ context.Context, _ string) ([]Ticket, error) { return nil, nil },
		GetTicketFn:        func(_ context.Context, _ string) (*Ticket, error) { return nil, nil },
		SetTicketStatusFn:  func(_ context.Context, _, _ string) error { return nil },
		ApplyLabelFn:       func(_ context.Context, _, _ string) error { return nil },
		RemoveLabelFn:      func(_ context.Context, _, _ string) error { return nil },
		CreateTicketFn: func(_ context.Context, _ CreateTicketInput) (*Ticket, error) {
			return nil, nil
		},
		UploadAttachmentFn: func(_ context.Context, _ string, _ AttachmentInput) error { return nil },
	}

	input := CreateTicketInput{TeamID: "team-1", Title: "New ticket"}
	attach := AttachmentInput{URL: "https://example.com", Title: "Report"}

	m.PollReadyTickets(ctx, "ready")
	m.GetTicket(ctx, "id-1")
	m.SetTicketStatus(ctx, "id-2", "Done")
	m.ApplyLabel(ctx, "id-3", "bug")
	m.RemoveLabel(ctx, "id-4", "wontfix")
	m.CreateTicket(ctx, input)
	m.UploadAttachment(ctx, "id-5", attach)

	require.Len(t, m.Calls, 7)

	tests := []struct {
		method string
		args   []any
	}{
		{"PollReadyTickets", []any{"ready"}},
		{"GetTicket", []any{"id-1"}},
		{"SetTicketStatus", []any{"id-2", "Done"}},
		{"ApplyLabel", []any{"id-3", "bug"}},
		{"RemoveLabel", []any{"id-4", "wontfix"}},
		{"CreateTicket", []any{input}},
		{"UploadAttachment", []any{"id-5", attach}},
	}

	for i, tt := range tests {
		assert.Equal(t, tt.method, m.Calls[i].Method, "call %d method", i)
		assert.Equal(t, tt.args, m.Calls[i].Args, "call %d args", i)
	}
}

func TestMockClient_ErrorPredicates(t *testing.T) {
	tests := []struct {
		name      string
		err       *Error
		isNotFound    bool
		isRateLimited bool
		isAPI         bool
		isNetwork     bool
	}{
		{
			name:       "NotFound",
			err:        NewNotFoundError("ticket not found", nil),
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
	err := NewNetworkError("linear api unreachable", cause)

	assert.Equal(t, "linear api unreachable", err.Error())
	assert.Equal(t, cause, errors.Unwrap(err))
	assert.True(t, errors.Is(err, cause))
}

func TestMockClient_ConfiguredError(t *testing.T) {
	m := &MockClient{
		GetTicketFn: func(_ context.Context, _ string) (*Ticket, error) {
			return nil, NewNotFoundError("ticket DEV-999 not found", nil)
		},
	}

	ticket, err := m.GetTicket(context.Background(), "DEV-999")
	assert.Nil(t, ticket)
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
	assert.False(t, IsRateLimited(err))
}

func TestMockClient_ConsumerWiring(t *testing.T) {
	m := &MockClient{
		PollReadyTicketsFn: func(_ context.Context, _ string) ([]Ticket, error) {
			return []Ticket{{ID: "1", Identifier: "DEV-200"}}, nil
		},
	}

	// Simulate a consumer that accepts the Client interface.
	pollAndCount := func(c Client, label string) (int, error) {
		tickets, err := c.PollReadyTickets(context.Background(), label)
		if err != nil {
			return 0, err
		}
		return len(tickets), nil
	}

	count, err := pollAndCount(m, "ready")
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Len(t, m.Calls, 1)
	assert.Equal(t, "PollReadyTickets", m.Calls[0].Method)
}

func TestMockClient_CallAccumulation(t *testing.T) {
	m := &MockClient{
		SetTicketStatusFn: func(_ context.Context, _, _ string) error { return nil },
	}

	m.SetTicketStatus(context.Background(), "id-1", "Done")
	m.SetTicketStatus(context.Background(), "id-2", "Cancelled")
	m.SetTicketStatus(context.Background(), "id-3", "In Progress")

	assert.Len(t, m.Calls, 3)
	assert.Equal(t, "id-1", m.Calls[0].Args[0])
	assert.Equal(t, "id-2", m.Calls[1].Args[0])
	assert.Equal(t, "id-3", m.Calls[2].Args[0])
}
