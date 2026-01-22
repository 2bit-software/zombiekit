// Package recall provides MCP tools for conversation retrieval.
package recall

// ListResponse is the recall-list-conversations response.
type ListResponse struct {
	Page    int                   `json:"page"`
	Limit   int                   `json:"limit"`
	HasMore bool                  `json:"has_more"`
	Items   []ConversationSummary `json:"items"`
}

// ConversationSummary is a summary of a conversation for list responses.
type ConversationSummary struct {
	ConversationID string `json:"conversation_id"`
	Title          string `json:"title"`
	MessageCount   int    `json:"message_count"`
	FirstMessage   string `json:"first_message"` // ISO-8601
	LastMessage    string `json:"last_message"`  // ISO-8601
	Source         string `json:"source"`
	Project        string `json:"project"`
}

// ReadResponse is the recall-read-conversation response.
type ReadResponse struct {
	ConversationID string        `json:"conversation_id"`
	Page           int           `json:"page"`
	Limit          int           `json:"limit"`
	HasMore        bool          `json:"has_more"`
	Items          []ChunkOutput `json:"items"`
}

// ChunkOutput is a conversation chunk for MCP output.
// Flattens metadata into top-level fields for easier consumption.
type ChunkOutput struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Role      string `json:"role"`
	Timestamp string `json:"timestamp"` // ISO-8601
	Project   string `json:"project"`   // CWD
	GitBranch string `json:"git_branch"`
}

// ErrorResponse is returned for validation and not-found errors.
type ErrorResponse struct {
	Error string `json:"error"`
}
