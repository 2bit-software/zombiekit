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

	// ListConversations returns conversations ordered by last activity (most recent first).
	// limit=0 uses implementation default (100), offset supports pagination.
	// project="" returns all conversations; non-empty filters by CWD prefix.
	ListConversations(ctx context.Context, limit, offset int, project string) ([]ConversationSummary, error)

	// ListDistinctProjects returns all unique project paths (CWD) from stored conversations.
	// Used to populate the project filter dropdown.
	ListDistinctProjects(ctx context.Context) ([]string, error)

	// GetImportState retrieves the import state for a file.
	// Returns nil, nil if no state exists (new file).
	GetImportState(ctx context.Context, filePath string) (*ImportState, error)

	// SaveImportState creates or updates the import state for a file.
	SaveImportState(ctx context.Context, state *ImportState) error

	// DeleteImportState removes the import state for a file.
	DeleteImportState(ctx context.Context, filePath string) error

	// CleanupStaleImportStates removes import states for files not in validPaths.
	CleanupStaleImportStates(ctx context.Context, validPaths []string) error
}
