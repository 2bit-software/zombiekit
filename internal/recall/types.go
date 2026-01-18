// Package recall provides semantic search storage and retrieval functionality.
package recall

import "time"

// Chunk represents a stored piece of content with its embedding.
type Chunk struct {
	ID             string    `json:"id"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"created_at"`
	Source         string    `json:"source,omitempty"`          // Source identifier: "claude", "slack", etc.
	SourceID       string    `json:"source_id,omitempty"`       // Original message ID from source
	ConversationID string    `json:"conversation_id,omitempty"` // Groups messages into conversations
	Metadata       *Metadata `json:"metadata,omitempty"`        // Source-specific metadata
}

// Metadata contains source-specific information about a chunk.
type Metadata struct {
	Role      string    `json:"role,omitempty"`       // "user" or "assistant"
	Timestamp time.Time `json:"timestamp,omitempty"`  // Original message timestamp
	GitBranch string    `json:"git_branch,omitempty"` // Git branch at time of message
	CWD       string    `json:"cwd,omitempty"`        // Working directory
	ParentID  string    `json:"parent_id,omitempty"`  // Parent message UUID for threading
}

// ChunkInput is used when saving new chunks with source tracking.
type ChunkInput struct {
	Content        string
	Source         string
	SourceID       string
	ConversationID string
	Metadata       *Metadata
}

// SearchResult wraps a chunk with its similarity score.
type SearchResult struct {
	Chunk      Chunk   `json:"chunk"`
	Similarity float64 `json:"similarity"` // 0.0 to 1.0, higher = more similar
}
