// Package recall provides semantic search storage and retrieval functionality.
package recall

import "time"

// Chunk represents a stored piece of content with its embedding.
type Chunk struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// SearchResult wraps a chunk with its similarity score.
type SearchResult struct {
	Chunk      Chunk   `json:"chunk"`
	Similarity float64 `json:"similarity"` // 0.0 to 1.0, higher = more similar
}
