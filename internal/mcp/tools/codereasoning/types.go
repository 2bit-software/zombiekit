// Package codereasoning provides the MCP code-reasoning tool implementation.
package codereasoning

import (
	"time"
)

// Thought represents a single step in a reasoning chain.
type Thought struct {
	Number           int       `json:"number"`
	Content          string    `json:"content"`
	IsRevision       bool      `json:"is_revision"`
	RevisesNumber    int       `json:"revises_thought,omitempty"`
	BranchID         string    `json:"branch_id,omitempty"`
	BranchFromNumber int       `json:"branch_from_thought,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// ThoughtRequest represents the input for adding a thought.
type ThoughtRequest struct {
	Thought           string `json:"thought"`
	ThoughtNumber     int    `json:"thought_number"`
	TotalThoughts     int    `json:"total_thoughts"`
	NextThoughtNeeded bool   `json:"next_thought_needed"`
	IsRevision        bool   `json:"is_revision"`
	RevisesThought    int    `json:"revises_thought"`
	BranchFromThought int    `json:"branch_from_thought"`
	BranchID          string `json:"branch_id"`
}

// ThoughtResponse represents the output after adding a thought.
type ThoughtResponse struct {
	ThoughtNumber int      `json:"thought_number"`
	TotalThoughts int      `json:"total_thoughts"`
	Chain         string   `json:"chain"`
	Status        string   `json:"status"`
	RevisedNumber int      `json:"revised_thought,omitempty"`
	BranchID      string   `json:"branch_id,omitempty"`
	BranchedFrom  int      `json:"branched_from,omitempty"`
	Branches      []string `json:"branches,omitempty"`
}

// MaxThoughts is the maximum number of thoughts allowed in a chain.
const MaxThoughts = 20
