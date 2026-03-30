package linear

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
	PollReadyTicketsFn func(ctx context.Context, label string, projectID string) ([]Ticket, error)
	GetTicketFn        func(ctx context.Context, id string) (*Ticket, error)
	SetTicketStatusFn  func(ctx context.Context, id string, status string) error
	ApplyLabelFn       func(ctx context.Context, id string, label string) error
	RemoveLabelFn      func(ctx context.Context, id string, label string) error
	CreateTicketFn     func(ctx context.Context, input CreateTicketInput) (*Ticket, error)
	UploadAttachmentFn func(ctx context.Context, ticketID string, input AttachmentInput) error
	PostCommentFn      func(ctx context.Context, issueID string, body string) error

	Calls []Call
}

func (m *MockClient) PollReadyTickets(ctx context.Context, label string, projectID string) ([]Ticket, error) {
	m.Calls = append(m.Calls, Call{Method: "PollReadyTickets", Args: []any{label, projectID}})
	if m.PollReadyTicketsFn != nil {
		return m.PollReadyTicketsFn(ctx, label, projectID)
	}
	return nil, fmt.Errorf("MockClient.PollReadyTickets not configured")
}

func (m *MockClient) GetTicket(ctx context.Context, id string) (*Ticket, error) {
	m.Calls = append(m.Calls, Call{Method: "GetTicket", Args: []any{id}})
	if m.GetTicketFn != nil {
		return m.GetTicketFn(ctx, id)
	}
	return nil, fmt.Errorf("MockClient.GetTicket not configured")
}

func (m *MockClient) SetTicketStatus(ctx context.Context, id string, status string) error {
	m.Calls = append(m.Calls, Call{Method: "SetTicketStatus", Args: []any{id, status}})
	if m.SetTicketStatusFn != nil {
		return m.SetTicketStatusFn(ctx, id, status)
	}
	return fmt.Errorf("MockClient.SetTicketStatus not configured")
}

func (m *MockClient) ApplyLabel(ctx context.Context, id string, label string) error {
	m.Calls = append(m.Calls, Call{Method: "ApplyLabel", Args: []any{id, label}})
	if m.ApplyLabelFn != nil {
		return m.ApplyLabelFn(ctx, id, label)
	}
	return fmt.Errorf("MockClient.ApplyLabel not configured")
}

func (m *MockClient) RemoveLabel(ctx context.Context, id string, label string) error {
	m.Calls = append(m.Calls, Call{Method: "RemoveLabel", Args: []any{id, label}})
	if m.RemoveLabelFn != nil {
		return m.RemoveLabelFn(ctx, id, label)
	}
	return fmt.Errorf("MockClient.RemoveLabel not configured")
}

func (m *MockClient) CreateTicket(ctx context.Context, input CreateTicketInput) (*Ticket, error) {
	m.Calls = append(m.Calls, Call{Method: "CreateTicket", Args: []any{input}})
	if m.CreateTicketFn != nil {
		return m.CreateTicketFn(ctx, input)
	}
	return nil, fmt.Errorf("MockClient.CreateTicket not configured")
}

func (m *MockClient) UploadAttachment(ctx context.Context, ticketID string, input AttachmentInput) error {
	m.Calls = append(m.Calls, Call{Method: "UploadAttachment", Args: []any{ticketID, input}})
	if m.UploadAttachmentFn != nil {
		return m.UploadAttachmentFn(ctx, ticketID, input)
	}
	return fmt.Errorf("MockClient.UploadAttachment not configured")
}

func (m *MockClient) PostComment(ctx context.Context, issueID string, body string) error {
	m.Calls = append(m.Calls, Call{Method: "PostComment", Args: []any{issueID, body}})
	if m.PostCommentFn != nil {
		return m.PostCommentFn(ctx, issueID, body)
	}
	return fmt.Errorf("MockClient.PostComment not configured")
}
