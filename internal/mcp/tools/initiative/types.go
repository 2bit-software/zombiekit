// Package initiative provides the MCP initiative tool implementation.
package initiative

import "time"

// Request represents a request to the initiative tool.
type Request struct {
	Action      string `json:"action"`      // create | status | complete | list
	Dir         string `json:"dir"`         // Working directory
	Type        string `json:"type"`        // For create: feature | bug | refactor
	Name        string `json:"name"`        // For create: initiative name
	Description string `json:"description"` // For create: optional description
}

// CreateResponse is returned for action=create.
type CreateResponse struct {
	Action         string `json:"action"`
	InitiativeID   string `json:"initiative_id"`
	InitiativePath string `json:"initiative_path"`
	CycleID        string `json:"cycle_id"`
	CyclePath      string `json:"cycle_path"`
	Branch         string `json:"branch"`
	Type           string `json:"type"`
	Name           string `json:"name"`
	NextStep       string `json:"next_step"`
	// Idempotency fields - indicate whether an existing initiative was returned
	AlreadyExisted bool     `json:"already_existed"`
	SkippedFiles   []string `json:"skipped_files,omitempty"`
	CopiedFiles    []string `json:"copied_files,omitempty"`
}

// StatusResponse is returned for action=status.
type StatusResponse struct {
	Action         string   `json:"action"`
	Active         bool     `json:"active"`
	InitiativeID   string   `json:"initiative_id,omitempty"`
	InitiativeType string   `json:"initiative_type,omitempty"`
	CurrentStep    string   `json:"current_step,omitempty"`
	CycleID        string   `json:"cycle_id,omitempty"`
	AvailableDocs  []string `json:"available_docs,omitempty"`
	SuggestedNext  string   `json:"suggested_next,omitempty"`
	HistoryPath    string   `json:"history_path,omitempty"`
	InitiativeFile string   `json:"initiative_file,omitempty"`
	Files          []string `json:"files,omitempty"`
}

// CompleteResponse is returned for action=complete.
type CompleteResponse struct {
	Action       string    `json:"action"`
	InitiativeID string    `json:"initiative_id"`
	CompletedAt  time.Time `json:"completed_at"`
}

// ListResponse is returned for action=list.
type ListResponse struct {
	Action      string              `json:"action"`
	Initiatives []InitiativeSummary `json:"initiatives"`
}

// InitiativeSummary provides a brief summary of an initiative.
type InitiativeSummary struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Path   string `json:"path"`
}

// ErrorResponse represents an error response from the tool.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains error information.
type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}
