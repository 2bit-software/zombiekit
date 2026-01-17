package recall

import "context"

// Storage defines the contract for recall chunk persistence.
type Storage interface {
	// Save stores content with its embedding.
	// Returns (id, created, error) where created=false indicates duplicate.
	Save(ctx context.Context, content string, embedding []float32) (id string, created bool, err error)

	// List returns all chunks ordered by created_at DESC.
	List(ctx context.Context, limit int) ([]Chunk, error)

	// Search finds chunks by cosine similarity to the query embedding.
	Search(ctx context.Context, embedding []float32, limit int) ([]SearchResult, error)

	// Close releases any resources held by the storage.
	Close() error
}
