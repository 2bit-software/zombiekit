// Package postgres implements recall storage using PostgreSQL with pgvector.
package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"

	"github.com/2bit-software/zombiekit/internal/config"
	"github.com/2bit-software/zombiekit/internal/recall"
)

// Storage implements recall.Storage using PostgreSQL with pgvector.
type Storage struct {
	pool *pgxpool.Pool
}

// New creates a new PostgreSQL storage instance.
func New(ctx context.Context, cfg config.StorageConfig) (*Storage, error) {
	if cfg.PostgresURL == "" {
		return nil, fmt.Errorf("postgres URL is required for recall storage")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres URL: %w", err)
	}

	// Configure pool settings
	poolConfig.MaxConns = cfg.MaxConns
	poolConfig.MinConns = cfg.MinConns
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	// Register pgvector types after each connection
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	// Verify connectivity immediately (fail-fast)
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("cannot connect to PostgreSQL database: %w", err)
	}

	return &Storage{pool: pool}, nil
}

// NewStorageWithPool creates a Storage with an existing pool (for testing).
func NewStorageWithPool(pool *pgxpool.Pool) *Storage {
	return &Storage{pool: pool}
}

// Save stores content with its embedding.
// Returns (id, created, error) where created=false indicates duplicate.
func (s *Storage) Save(ctx context.Context, content string, embedding []float32) (string, bool, error) {
	hash := recall.ContentHash(content)
	id := uuid.New().String()

	result, err := s.pool.Exec(ctx, `
		INSERT INTO recall_chunks (id, content, content_hash, embedding)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (content_hash) DO NOTHING
	`, id, content, hash, pgvector.NewVector(embedding))
	if err != nil {
		return "", false, fmt.Errorf("save chunk: %w", err)
	}

	// RowsAffected() == 0 means duplicate
	if result.RowsAffected() == 0 {
		// Duplicate - return silently (no output per spec)
		return "", false, nil
	}

	return id, true, nil
}

// List returns all chunks ordered by created_at DESC.
func (s *Storage) List(ctx context.Context, limit int) ([]recall.Chunk, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, content, created_at
		FROM recall_chunks
		ORDER BY created_at DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("list chunks: %w", err)
	}
	defer rows.Close()

	var chunks []recall.Chunk
	for rows.Next() {
		var chunk recall.Chunk
		if err := rows.Scan(&chunk.ID, &chunk.Content, &chunk.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}
		chunks = append(chunks, chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chunks: %w", err)
	}

	return chunks, nil
}

// Search finds chunks by cosine similarity to the query embedding.
func (s *Storage) Search(ctx context.Context, embedding []float32, limit int) ([]recall.SearchResult, error) {
	if limit <= 0 {
		limit = 5
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, content, created_at, source, source_id, conversation_id, metadata,
		       1 - (embedding <=> $1) AS similarity
		FROM recall_chunks
		WHERE embedding IS NOT NULL
		ORDER BY embedding <=> $1
		LIMIT $2
	`, pgvector.NewVector(embedding), limit)
	if err != nil {
		return nil, fmt.Errorf("search chunks: %w", err)
	}
	defer rows.Close()

	var results []recall.SearchResult
	for rows.Next() {
		var result recall.SearchResult
		var source, sourceID, convID *string
		var metadataJSON []byte

		if err := rows.Scan(
			&result.Chunk.ID,
			&result.Chunk.Content,
			&result.Chunk.CreatedAt,
			&source, &sourceID, &convID, &metadataJSON,
			&result.Similarity,
		); err != nil {
			return nil, fmt.Errorf("scan result: %w", err)
		}

		populateChunkNullables(&result.Chunk, source, sourceID, convID, metadataJSON)
		results = append(results, result)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate results: %w", err)
	}

	return results, nil
}

// Close releases any resources held by the storage.
func (s *Storage) Close() error {
	if s.pool != nil {
		s.pool.Close()
	}
	return nil
}

// Ping verifies the database connection is alive.
func (s *Storage) Ping(ctx context.Context) error {
	return s.pool.Ping(ctx)
}

// SaveWithSource stores content with source tracking and embedding.
// Returns (id, created, error) where created=false indicates duplicate (same source+source_id).
func (s *Storage) SaveWithSource(ctx context.Context, input recall.ChunkInput, embedding []float32) (string, bool, error) {
	hash := recall.ContentHash(input.Content)
	id := uuid.New().String()

	// Convert metadata to JSON
	var metadataJSON []byte
	var err error
	if input.Metadata != nil {
		metadataJSON, err = json.Marshal(input.Metadata)
		if err != nil {
			return "", false, fmt.Errorf("marshal metadata: %w", err)
		}
	}

	result, err := s.pool.Exec(ctx, `
		INSERT INTO recall_chunks (id, content, content_hash, embedding, source, source_id, conversation_id, metadata, history_gap)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (source, source_id) WHERE source_id IS NOT NULL DO NOTHING
	`, id, input.Content, hash, pgvector.NewVector(embedding),
		nullableString(input.Source), nullableString(input.SourceID),
		nullableString(input.ConversationID), metadataJSON, input.HistoryGap)
	if err != nil {
		return "", false, fmt.Errorf("save chunk with source: %w", err)
	}

	// RowsAffected() == 0 means duplicate
	if result.RowsAffected() == 0 {
		return "", false, nil
	}

	return id, true, nil
}

// ExistsBySourceID checks if a chunk with the given source and source_id already exists.
func (s *Storage) ExistsBySourceID(ctx context.Context, source, sourceID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM recall_chunks
			WHERE source = $1 AND source_id = $2
		)
	`, source, sourceID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check source_id exists: %w", err)
	}
	return exists, nil
}

// GetByConversation returns all chunks belonging to a conversation, ordered by timestamp.
func (s *Storage) GetByConversation(ctx context.Context, conversationID string) ([]recall.Chunk, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, content, created_at, source, source_id, conversation_id, metadata
		FROM recall_chunks
		WHERE conversation_id = $1
		ORDER BY (metadata->>'timestamp')::timestamptz ASC NULLS LAST, created_at ASC
	`, conversationID)
	if err != nil {
		return nil, fmt.Errorf("get by conversation: %w", err)
	}
	defer rows.Close()

	return scanChunkRows(rows)
}

// scanChunkRows iterates rows containing standard chunk columns
// (id, content, created_at, source, source_id, conversation_id, metadata)
// and returns the collected chunks.
func scanChunkRows(rows pgx.Rows) ([]recall.Chunk, error) {
	var chunks []recall.Chunk
	for rows.Next() {
		var chunk recall.Chunk
		var source, sourceID, convID *string
		var metadataJSON []byte

		if err := rows.Scan(&chunk.ID, &chunk.Content, &chunk.CreatedAt,
			&source, &sourceID, &convID, &metadataJSON); err != nil {
			return nil, fmt.Errorf("scan chunk: %w", err)
		}

		populateChunkNullables(&chunk, source, sourceID, convID, metadataJSON)
		chunks = append(chunks, chunk)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate chunks: %w", err)
	}
	return chunks, nil
}

// populateChunkNullables fills optional chunk fields from nullable scan results.
func populateChunkNullables(chunk *recall.Chunk, source, sourceID, convID *string, metadataJSON []byte) {
	if source != nil {
		chunk.Source = *source
	}
	if sourceID != nil {
		chunk.SourceID = *sourceID
	}
	if convID != nil {
		chunk.ConversationID = *convID
	}
	if len(metadataJSON) > 0 {
		var meta recall.Metadata
		if err := json.Unmarshal(metadataJSON, &meta); err == nil {
			chunk.Metadata = &meta
		}
	}
}

// nullableString returns nil for empty strings, or a pointer to the string.
func nullableString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// ListConversations returns conversations ordered by last activity (most recent first).
func (s *Storage) ListConversations(ctx context.Context, limit, offset int, project string) ([]recall.ConversationSummary, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.pool.Query(ctx, `
		WITH first_user_msg AS (
			SELECT DISTINCT ON (conversation_id)
				conversation_id,
				SUBSTRING(content, 1, 100) as title
			FROM recall_chunks
			WHERE conversation_id IS NOT NULL
			  AND metadata->>'role' = 'user'
			ORDER BY conversation_id, COALESCE((metadata->>'timestamp')::timestamptz, created_at) ASC
		)
		SELECT
			rc.conversation_id,
			COALESCE(fum.title, '[No title]') as title,
			COUNT(*) as message_count,
			MIN(COALESCE((rc.metadata->>'timestamp')::timestamptz, rc.created_at)) as first_message,
			MAX(COALESCE((rc.metadata->>'timestamp')::timestamptz, rc.created_at)) as last_message,
			rc.source,
			COALESCE(MAX(rc.metadata->>'cwd'), '') as project
		FROM recall_chunks rc
		LEFT JOIN first_user_msg fum ON rc.conversation_id = fum.conversation_id
		WHERE rc.conversation_id IS NOT NULL
		  AND ($3 = '' OR rc.metadata->>'cwd' LIKE $3 || '%')
		GROUP BY rc.conversation_id, rc.source, fum.title
		HAVING COUNT(*) > 0
		ORDER BY last_message DESC
		LIMIT $1 OFFSET $2
	`, limit, offset, project)
	if err != nil {
		return nil, fmt.Errorf("list conversations: %w", err)
	}
	defer rows.Close()

	var summaries []recall.ConversationSummary
	for rows.Next() {
		var summary recall.ConversationSummary
		var source *string
		if err := rows.Scan(
			&summary.ConversationID,
			&summary.Title,
			&summary.MessageCount,
			&summary.FirstMessage,
			&summary.LastMessage,
			&source,
			&summary.Project,
		); err != nil {
			return nil, fmt.Errorf("scan conversation: %w", err)
		}
		if source != nil {
			summary.Source = *source
		}
		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conversations: %w", err)
	}

	return summaries, nil
}

// ListDistinctProjects returns all unique project paths (CWD) from stored conversations.
func (s *Storage) ListDistinctProjects(ctx context.Context) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT metadata->>'cwd' as project
		FROM recall_chunks
		WHERE metadata->>'cwd' IS NOT NULL
		ORDER BY project
	`)
	if err != nil {
		return nil, fmt.Errorf("list distinct projects: %w", err)
	}
	defer rows.Close()

	var projects []string
	for rows.Next() {
		var project string
		if err := rows.Scan(&project); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		projects = append(projects, project)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate projects: %w", err)
	}

	return projects, nil
}

// GetImportState retrieves the import state for a file.
// Returns nil, nil if no state exists (new file).
func (s *Storage) GetImportState(ctx context.Context, filePath string) (*recall.ImportState, error) {
	var state recall.ImportState
	err := s.pool.QueryRow(ctx, `
		SELECT file_path, last_entry_uuid, file_mtime, updated_at
		FROM recall_import_state
		WHERE file_path = $1
	`, filePath).Scan(&state.FilePath, &state.LastEntryUUID, &state.FileMtime, &state.UpdatedAt)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get import state: %w", err)
	}
	return &state, nil
}

// SaveImportState creates or updates the import state for a file.
func (s *Storage) SaveImportState(ctx context.Context, state *recall.ImportState) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO recall_import_state (file_path, last_entry_uuid, file_mtime, updated_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT (file_path) DO UPDATE
		SET last_entry_uuid = $2, file_mtime = $3, updated_at = NOW()
	`, state.FilePath, state.LastEntryUUID, state.FileMtime)
	if err != nil {
		return fmt.Errorf("save import state: %w", err)
	}
	return nil
}

// DeleteImportState removes the import state for a file.
func (s *Storage) DeleteImportState(ctx context.Context, filePath string) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM recall_import_state WHERE file_path = $1
	`, filePath)
	if err != nil {
		return fmt.Errorf("delete import state: %w", err)
	}
	return nil
}

// CleanupStaleImportStates removes import states for files not in validPaths.
func (s *Storage) CleanupStaleImportStates(ctx context.Context, validPaths []string) error {
	if len(validPaths) == 0 {
		// If no valid paths, delete all import states
		_, err := s.pool.Exec(ctx, `DELETE FROM recall_import_state`)
		if err != nil {
			return fmt.Errorf("cleanup all import states: %w", err)
		}
		return nil
	}

	_, err := s.pool.Exec(ctx, `
		DELETE FROM recall_import_state
		WHERE file_path != ALL($1)
	`, validPaths)
	if err != nil {
		return fmt.Errorf("cleanup stale import states: %w", err)
	}
	return nil
}

// GetConversationChunks returns chunks for a conversation with pagination.
// Ordered by timestamp ascending (oldest first), then by ID for determinism.
func (s *Storage) GetConversationChunks(ctx context.Context, conversationID string, limit, offset int) ([]recall.Chunk, error) {
	if limit <= 0 {
		limit = 20
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, content, created_at, source, source_id, conversation_id, metadata
		FROM recall_chunks
		WHERE conversation_id = $1
		ORDER BY (metadata->>'timestamp')::timestamptz ASC NULLS LAST, id ASC
		LIMIT $2 OFFSET $3
	`, conversationID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("get conversation chunks: %w", err)
	}
	defer rows.Close()

	return scanChunkRows(rows)
}

// ConversationExists checks if any chunks exist for the given conversation ID.
func (s *Storage) ConversationExists(ctx context.Context, conversationID string) (bool, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM recall_chunks WHERE conversation_id = $1
		)
	`, conversationID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check conversation exists: %w", err)
	}
	return exists, nil
}
