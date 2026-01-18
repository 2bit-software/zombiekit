package recall

import "context"

// Storage defines the contract for recall chunk persistence.
type Storage interface {
	// Save stores content with its embedding.
	// Returns (id, created, error) where created=false indicates duplicate.
	Save(ctx context.Context, content string, embedding []float32) (id string, created bool, err error)

	// SaveWithSource stores content with source tracking and embedding.
	// Returns (id, created, error) where created=false indicates duplicate (same source+source_id).
	SaveWithSource(ctx context.Context, input ChunkInput, embedding []float32) (id string, created bool, err error)

	// ExistsBySourceID checks if a chunk with the given source and source_id already exists.
	// Fast lookup for duplicate detection before generating embeddings.
	ExistsBySourceID(ctx context.Context, source, sourceID string) (bool, error)

	// GetByConversation returns all chunks belonging to a conversation, ordered by timestamp.
	GetByConversation(ctx context.Context, conversationID string) ([]Chunk, error)

	// List returns all chunks ordered by created_at DESC.
	List(ctx context.Context, limit int) ([]Chunk, error)

	// Search finds chunks by cosine similarity to the query embedding.
	Search(ctx context.Context, embedding []float32, limit int) ([]SearchResult, error)

	// Close releases any resources held by the storage.
	Close() error
}
