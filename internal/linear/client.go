package linear

import "context"

// Client defines the interface for Linear API operations.
type Client interface {
	PollReadyTickets(ctx context.Context, label string) ([]Ticket, error)
	GetTicket(ctx context.Context, id string) (*Ticket, error)
	SetTicketStatus(ctx context.Context, id string, status string) error
	ApplyLabel(ctx context.Context, id string, label string) error
	RemoveLabel(ctx context.Context, id string, label string) error
	CreateTicket(ctx context.Context, input CreateTicketInput) (*Ticket, error)
	UploadAttachment(ctx context.Context, ticketID string, input AttachmentInput) error
}

// Ticket represents a Linear issue.
type Ticket struct {
	ID          string
	Identifier  string
	Title       string
	Description string
	Status      string
	Labels      []string
	Priority    int
	URL         string
}

// CreateTicketInput holds parameters for creating a ticket.
type CreateTicketInput struct {
	TeamID      string
	Title       string
	Description string
	StateID     string
	LabelIDs    []string
	Priority    *int
	AssigneeID  string
}

// AttachmentInput holds parameters for creating an attachment.
type AttachmentInput struct {
	URL      string
	Title    string
	Subtitle string
}
