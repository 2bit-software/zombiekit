package postgres

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/zombiekit/brains/internal/recall"
)

// setupTestStorage creates a PostgreSQL testcontainer with pgvector and returns a connected storage.
func setupTestStorage(t *testing.T) *Storage {
	t.Helper()

	ctx := context.Background()

	// Use pgvector-enabled postgres image
	container, err := tcpostgres.Run(ctx,
		"pgvector/pgvector:pg16",
		tcpostgres.WithDatabase("test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		if err := container.Terminate(ctx); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// First, connect without pgvector types to create the extension
	initPool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)

	// Create extension before registering types
	_, err = initPool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS vector`)
	require.NoError(t, err)
	initPool.Close()

	// Now create pool with pgvector types registered
	poolConfig, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err)

	// Register pgvector types (now the extension exists)
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS recall_chunks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			content TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			embedding vector(768),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			source TEXT,
			source_id TEXT,
			conversation_id TEXT,
			metadata JSONB
		)
	`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_recall_chunks_content_hash ON recall_chunks(content_hash)`)
	require.NoError(t, err)

	// Index for source tracking duplicate detection
	_, err = pool.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_recall_chunks_source_id ON recall_chunks(source, source_id) WHERE source_id IS NOT NULL`)
	require.NoError(t, err)

	// Index for conversation retrieval
	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_recall_chunks_conversation ON recall_chunks(conversation_id) WHERE conversation_id IS NOT NULL`)
	require.NoError(t, err)

	storage := &Storage{pool: pool}
	return storage
}

// generateTestEmbedding creates a deterministic test embedding based on the content.
func generateTestEmbedding(content string) []float32 {
	embedding := make([]float32, 768)
	for i := 0; i < len(content) && i < 768; i++ {
		embedding[i] = float32(content[i]) / 255.0
	}
	return embedding
}

func TestSave_NewContent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	content := "The deployment failed because of memory limits"
	embedding := generateTestEmbedding(content)

	id, created, err := storage.Save(ctx, content, embedding)
	require.NoError(t, err)
	assert.True(t, created, "expected content to be created")
	assert.NotEmpty(t, id, "expected non-empty ID")
}

func TestSave_DuplicateContent_ReturnsFalse(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	content := "Duplicate test content"
	embedding := generateTestEmbedding(content)

	// First save
	id1, created1, err := storage.Save(ctx, content, embedding)
	require.NoError(t, err)
	assert.True(t, created1)
	assert.NotEmpty(t, id1)

	// Second save with same content
	id2, created2, err := storage.Save(ctx, content, embedding)
	require.NoError(t, err)
	assert.False(t, created2, "expected duplicate to return created=false")
	assert.Empty(t, id2, "expected no ID for duplicate")
}

func TestList_ReturnsChunksOrderedByCreatedAtDesc(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create chunks with small delay to ensure different timestamps
	content1 := "First content"
	_, _, err := storage.Save(ctx, content1, generateTestEmbedding(content1))
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	content2 := "Second content"
	_, _, err = storage.Save(ctx, content2, generateTestEmbedding(content2))
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	content3 := "Third content"
	_, _, err = storage.Save(ctx, content3, generateTestEmbedding(content3))
	require.NoError(t, err)

	// List should return most recent first
	chunks, err := storage.List(ctx, 10)
	require.NoError(t, err)
	require.Len(t, chunks, 3)

	assert.Equal(t, "Third content", chunks[0].Content)
	assert.Equal(t, "Second content", chunks[1].Content)
	assert.Equal(t, "First content", chunks[2].Content)
}

func TestList_Empty_ReturnsEmptySlice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	chunks, err := storage.List(ctx, 10)
	require.NoError(t, err)
	assert.Empty(t, chunks)
}

func TestSearch_ReturnsSimilarContent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	// Store some content
	content1 := "The deployment failed because of memory limits"
	content2 := "CSS styling for the login page was updated"
	content3 := "Database connection pool exhausted"

	_, _, err := storage.Save(ctx, content1, generateTestEmbedding(content1))
	require.NoError(t, err)
	_, _, err = storage.Save(ctx, content2, generateTestEmbedding(content2))
	require.NoError(t, err)
	_, _, err = storage.Save(ctx, content3, generateTestEmbedding(content3))
	require.NoError(t, err)

	// Search with similar embedding to content1
	queryEmbedding := generateTestEmbedding("The deployment failed because of memory limits")
	results, err := storage.Search(ctx, queryEmbedding, 5)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(results), 1)

	// First result should be the most similar (content1)
	assert.Equal(t, content1, results[0].Chunk.Content)
	assert.Greater(t, results[0].Similarity, 0.5, "expected high similarity for exact match")
}

func TestSearch_OrderedBySimilarity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	// Store content
	_, _, err := storage.Save(ctx, "AAA", generateTestEmbedding("AAA"))
	require.NoError(t, err)
	_, _, err = storage.Save(ctx, "BBB", generateTestEmbedding("BBB"))
	require.NoError(t, err)
	_, _, err = storage.Save(ctx, "CCC", generateTestEmbedding("CCC"))
	require.NoError(t, err)

	// Search
	results, err := storage.Search(ctx, generateTestEmbedding("AAA"), 10)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Verify ordered by similarity DESC
	for i := 1; i < len(results); i++ {
		assert.GreaterOrEqual(t, results[i-1].Similarity, results[i].Similarity,
			"results should be ordered by similarity DESC")
	}
}

func TestSearch_Empty_ReturnsEmptySlice(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	results, err := storage.Search(ctx, generateTestEmbedding("query"), 5)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestStorage_ImplementsInterface(t *testing.T) {
	// Compile-time check that Storage implements recall.Storage interface
	var _ recall.Storage = (*Storage)(nil)
}

// ============================================================
// T017: Tests for source tracking methods (BR-002, BR-008, BR-009, BR-010)
// ============================================================

func TestExistsBySourceID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	exists, err := storage.ExistsBySourceID(ctx, "claude", "nonexistent-uuid")
	require.NoError(t, err)
	assert.False(t, exists, "expected nonexistent source_id to return false")
}

func TestExistsBySourceID_Found(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	// Save a chunk with source tracking
	input := recall.ChunkInput{
		Content:        "Test content",
		Source:         "claude",
		SourceID:       "test-uuid-123",
		ConversationID: "conv-1",
	}
	_, created, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
	require.NoError(t, err)
	require.True(t, created)

	// Check it exists
	exists, err := storage.ExistsBySourceID(ctx, "claude", "test-uuid-123")
	require.NoError(t, err)
	assert.True(t, exists, "expected existing source_id to return true")
}

func TestExistsBySourceID_SameIDDifferentSource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	// Save with source "claude"
	input1 := recall.ChunkInput{
		Content:        "Content from claude",
		Source:         "claude",
		SourceID:       "shared-uuid",
		ConversationID: "conv-1",
	}
	_, _, err := storage.SaveWithSource(ctx, input1, generateTestEmbedding(input1.Content))
	require.NoError(t, err)

	// Same source_id but different source should not exist
	exists, err := storage.ExistsBySourceID(ctx, "slack", "shared-uuid")
	require.NoError(t, err)
	assert.False(t, exists, "same source_id from different source should not exist")

	// Same source and source_id should exist
	exists, err = storage.ExistsBySourceID(ctx, "claude", "shared-uuid")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestSaveWithSource_NewMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	input := recall.ChunkInput{
		Content:        "Hello from Claude",
		Source:         "claude",
		SourceID:       "msg-001",
		ConversationID: "session-abc",
		Metadata: &recall.Metadata{
			Role:      "user",
			GitBranch: "main",
			CWD:       "/home/user/project",
		},
	}

	id, created, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
	require.NoError(t, err)
	assert.True(t, created, "expected new message to be created")
	assert.NotEmpty(t, id)
}

func TestSaveWithSource_DuplicateMessage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	input := recall.ChunkInput{
		Content:        "Duplicate test message",
		Source:         "claude",
		SourceID:       "dup-uuid",
		ConversationID: "conv-1",
	}

	// First save
	id1, created1, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
	require.NoError(t, err)
	assert.True(t, created1)
	assert.NotEmpty(t, id1)

	// Second save with same source+source_id
	id2, created2, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
	require.NoError(t, err)
	assert.False(t, created2, "expected duplicate to return created=false")
	assert.Empty(t, id2)
}

func TestSaveWithSource_StoresSourceID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	input := recall.ChunkInput{
		Content:        "Test content with source",
		Source:         "claude",
		SourceID:       "unique-source-id-xyz",
		ConversationID: "conv-test",
	}

	_, created, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
	require.NoError(t, err)
	require.True(t, created)

	// Verify by checking existence
	exists, err := storage.ExistsBySourceID(ctx, "claude", "unique-source-id-xyz")
	require.NoError(t, err)
	assert.True(t, exists, "source_id should be persisted")
}

func TestSaveWithSource_StoresConversationID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	convID := "conversation-123"
	input := recall.ChunkInput{
		Content:        "Message in conversation",
		Source:         "claude",
		SourceID:       "msg-in-conv",
		ConversationID: convID,
	}

	_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
	require.NoError(t, err)

	// Retrieve by conversation
	chunks, err := storage.GetByConversation(ctx, convID)
	require.NoError(t, err)
	require.Len(t, chunks, 1)
	assert.Equal(t, convID, chunks[0].ConversationID)
}

func TestSaveWithSource_StoresMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	convID := "conv-metadata-test"
	input := recall.ChunkInput{
		Content:        "Message with metadata",
		Source:         "claude",
		SourceID:       "meta-msg",
		ConversationID: convID,
		Metadata: &recall.Metadata{
			Role:      "assistant",
			GitBranch: "feature/test",
			CWD:       "/workspace",
			ParentID:  "parent-uuid",
		},
	}

	_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
	require.NoError(t, err)

	chunks, err := storage.GetByConversation(ctx, convID)
	require.NoError(t, err)
	require.Len(t, chunks, 1)

	meta := chunks[0].Metadata
	require.NotNil(t, meta)
	assert.Equal(t, "assistant", meta.Role)
	assert.Equal(t, "feature/test", meta.GitBranch)
	assert.Equal(t, "/workspace", meta.CWD)
	assert.Equal(t, "parent-uuid", meta.ParentID)
}

func TestGetByConversation_ReturnsAllMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	convID := "multi-message-conv"

	// Add multiple messages to same conversation
	for i, content := range []string{"First message", "Second message", "Third message"} {
		input := recall.ChunkInput{
			Content:        content,
			Source:         "claude",
			SourceID:       "msg-" + string(rune('a'+i)),
			ConversationID: convID,
		}
		_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(content))
		require.NoError(t, err)
	}

	chunks, err := storage.GetByConversation(ctx, convID)
	require.NoError(t, err)
	assert.Len(t, chunks, 3, "expected all 3 messages in conversation")
}

func TestGetByConversation_OrderedByTimestamp(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	convID := "ordered-conv"

	// Create messages with timestamps in reverse order
	times := []string{"2024-01-15T10:00:00Z", "2024-01-15T11:00:00Z", "2024-01-15T09:00:00Z"}
	contents := []string{"Middle message", "Latest message", "Earliest message"}

	for i, content := range contents {
		ts, _ := time.Parse(time.RFC3339, times[i])
		input := recall.ChunkInput{
			Content:        content,
			Source:         "claude",
			SourceID:       "ordered-" + string(rune('a'+i)),
			ConversationID: convID,
			Metadata: &recall.Metadata{
				Timestamp: ts,
			},
		}
		_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(content))
		require.NoError(t, err)
	}

	chunks, err := storage.GetByConversation(ctx, convID)
	require.NoError(t, err)
	require.Len(t, chunks, 3)

	// Should be ordered by timestamp ASC
	assert.Equal(t, "Earliest message", chunks[0].Content)
	assert.Equal(t, "Middle message", chunks[1].Content)
	assert.Equal(t, "Latest message", chunks[2].Content)
}

func TestGetByConversation_EmptyResult(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	chunks, err := storage.GetByConversation(ctx, "nonexistent-conversation")
	require.NoError(t, err)
	assert.Empty(t, chunks, "expected empty slice for unknown conversation")
}

func TestGetByConversation_PreservesMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	convID := "metadata-preserve-conv"
	ts, _ := time.Parse(time.RFC3339, "2024-06-15T14:30:00Z")

	input := recall.ChunkInput{
		Content:        "Message with full metadata",
		Source:         "claude",
		SourceID:       "full-meta-msg",
		ConversationID: convID,
		Metadata: &recall.Metadata{
			Role:      "user",
			Timestamp: ts,
			GitBranch: "main",
			CWD:       "/home/dev",
			ParentID:  "prev-msg-uuid",
		},
	}

	_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
	require.NoError(t, err)

	chunks, err := storage.GetByConversation(ctx, convID)
	require.NoError(t, err)
	require.Len(t, chunks, 1)

	chunk := chunks[0]
	assert.Equal(t, "claude", chunk.Source)
	assert.Equal(t, "full-meta-msg", chunk.SourceID)
	assert.Equal(t, convID, chunk.ConversationID)

	require.NotNil(t, chunk.Metadata)
	assert.Equal(t, "user", chunk.Metadata.Role)
	assert.True(t, chunk.Metadata.Timestamp.Equal(ts))
	assert.Equal(t, "main", chunk.Metadata.GitBranch)
	assert.Equal(t, "/home/dev", chunk.Metadata.CWD)
	assert.Equal(t, "prev-msg-uuid", chunk.Metadata.ParentID)
}

func TestGetByConversation_IncludesChunkedMessages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	convID := "chunked-conv"

	// Simulate a message that was split into 3 chunks
	for i := range 3 {
		input := recall.ChunkInput{
			Content:        "Chunk part " + string(rune('A'+i)),
			Source:         "claude",
			SourceID:       "original-uuid-" + string(rune('0'+i)), // original-uuid-0, -1, -2
			ConversationID: convID,
		}
		_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
		require.NoError(t, err)
	}

	chunks, err := storage.GetByConversation(ctx, convID)
	require.NoError(t, err)
	assert.Len(t, chunks, 3, "expected all chunks from split message")
}

// ============================================================
// Tests for ListConversations and ListDistinctProjects (DEV-70)
// ============================================================

func TestListConversations_ReturnsConversationSummaries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create two conversations
	convID1 := "conv-list-1"
	convID2 := "conv-list-2"

	// First conversation with user and assistant messages
	for i, msg := range []struct {
		content string
		role    string
	}{
		{"Hello, how can I help?", "user"},
		{"I can help with coding tasks", "assistant"},
	} {
		ts, _ := time.Parse(time.RFC3339, "2024-01-15T10:00:00Z")
		ts = ts.Add(time.Duration(i) * time.Minute)
		input := recall.ChunkInput{
			Content:        msg.content,
			Source:         "claude",
			SourceID:       "conv1-msg-" + string(rune('a'+i)),
			ConversationID: convID1,
			Metadata: &recall.Metadata{
				Role:      msg.role,
				Timestamp: ts,
				CWD:       "/project/a",
			},
		}
		_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
		require.NoError(t, err)
	}

	// Second conversation
	for i, msg := range []struct {
		content string
		role    string
	}{
		{"What's the weather?", "user"},
		{"I cannot check weather", "assistant"},
		{"Ok thanks", "user"},
	} {
		ts, _ := time.Parse(time.RFC3339, "2024-01-15T11:00:00Z")
		ts = ts.Add(time.Duration(i) * time.Minute)
		input := recall.ChunkInput{
			Content:        msg.content,
			Source:         "claude",
			SourceID:       "conv2-msg-" + string(rune('a'+i)),
			ConversationID: convID2,
			Metadata: &recall.Metadata{
				Role:      msg.role,
				Timestamp: ts,
				CWD:       "/project/b",
			},
		}
		_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
		require.NoError(t, err)
	}

	summaries, err := storage.ListConversations(ctx, 10, 0, "")
	require.NoError(t, err)
	require.Len(t, summaries, 2)

	// Should be ordered by last message (most recent first)
	// conv2 has messages at 11:00, 11:01, 11:02 - latest is 11:02
	// conv1 has messages at 10:00, 10:01 - latest is 10:01
	assert.Equal(t, convID2, summaries[0].ConversationID)
	assert.Equal(t, convID1, summaries[1].ConversationID)

	// First summary should have title from first user message
	assert.Equal(t, "What's the weather?", summaries[0].Title)
	assert.Equal(t, 3, summaries[0].MessageCount)

	assert.Equal(t, "Hello, how can I help?", summaries[1].Title)
	assert.Equal(t, 2, summaries[1].MessageCount)
}

func TestListConversations_FiltersbyProject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create conversations in different projects
	projects := []string{"/project/frontend", "/project/backend", "/other/project"}

	for i, proj := range projects {
		input := recall.ChunkInput{
			Content:        "Message in " + proj,
			Source:         "claude",
			SourceID:       "proj-msg-" + string(rune('a'+i)),
			ConversationID: "conv-proj-" + string(rune('a'+i)),
			Metadata: &recall.Metadata{
				Role: "user",
				CWD:  proj,
			},
		}
		_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
		require.NoError(t, err)
	}

	// Filter by /project prefix - should match first two
	summaries, err := storage.ListConversations(ctx, 10, 0, "/project")
	require.NoError(t, err)
	assert.Len(t, summaries, 2)

	// Filter by exact path
	summaries, err = storage.ListConversations(ctx, 10, 0, "/project/frontend")
	require.NoError(t, err)
	assert.Len(t, summaries, 1)
	assert.Contains(t, summaries[0].Project, "/project/frontend")
}

func TestListConversations_Pagination(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create 5 conversations
	for i := range 5 {
		ts, _ := time.Parse(time.RFC3339, "2024-01-15T10:00:00Z")
		ts = ts.Add(time.Duration(i) * time.Hour)
		input := recall.ChunkInput{
			Content:        "Conversation " + string(rune('A'+i)),
			Source:         "claude",
			SourceID:       "page-msg-" + string(rune('a'+i)),
			ConversationID: "conv-page-" + string(rune('a'+i)),
			Metadata: &recall.Metadata{
				Role:      "user",
				Timestamp: ts,
			},
		}
		_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
		require.NoError(t, err)
	}

	// Get first page (limit 2)
	page1, err := storage.ListConversations(ctx, 2, 0, "")
	require.NoError(t, err)
	assert.Len(t, page1, 2)
	// Most recent first
	assert.Equal(t, "conv-page-e", page1[0].ConversationID)
	assert.Equal(t, "conv-page-d", page1[1].ConversationID)

	// Get second page
	page2, err := storage.ListConversations(ctx, 2, 2, "")
	require.NoError(t, err)
	assert.Len(t, page2, 2)
	assert.Equal(t, "conv-page-c", page2[0].ConversationID)
	assert.Equal(t, "conv-page-b", page2[1].ConversationID)

	// Get third page (only 1 remaining)
	page3, err := storage.ListConversations(ctx, 2, 4, "")
	require.NoError(t, err)
	assert.Len(t, page3, 1)
	assert.Equal(t, "conv-page-a", page3[0].ConversationID)
}

func TestListConversations_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	summaries, err := storage.ListConversations(ctx, 10, 0, "")
	require.NoError(t, err)
	assert.Empty(t, summaries)
}

func TestListConversations_NoTitleFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create conversation with only assistant message (no user message for title)
	input := recall.ChunkInput{
		Content:        "Assistant response without user prompt",
		Source:         "claude",
		SourceID:       "no-title-msg",
		ConversationID: "conv-no-title",
		Metadata: &recall.Metadata{
			Role: "assistant",
		},
	}
	_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
	require.NoError(t, err)

	summaries, err := storage.ListConversations(ctx, 10, 0, "")
	require.NoError(t, err)
	require.Len(t, summaries, 1)
	assert.Equal(t, "[No title]", summaries[0].Title)
}

func TestListDistinctProjects_ReturnsUniqueProjects(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create chunks with various CWDs (some duplicates)
	cwds := []string{"/project/a", "/project/b", "/project/a", "/other", "/project/b"}

	for i, cwd := range cwds {
		input := recall.ChunkInput{
			Content:        "Message " + string(rune('a'+i)),
			Source:         "claude",
			SourceID:       "dist-proj-" + string(rune('a'+i)),
			ConversationID: "conv-dist-" + string(rune('a'+i)),
			Metadata: &recall.Metadata{
				CWD: cwd,
			},
		}
		_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
		require.NoError(t, err)
	}

	projects, err := storage.ListDistinctProjects(ctx)
	require.NoError(t, err)

	// Should have 3 unique projects, sorted
	assert.Len(t, projects, 3)
	assert.Equal(t, "/other", projects[0])
	assert.Equal(t, "/project/a", projects[1])
	assert.Equal(t, "/project/b", projects[2])
}

func TestListDistinctProjects_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	projects, err := storage.ListDistinctProjects(ctx)
	require.NoError(t, err)
	assert.Empty(t, projects)
}

func TestListDistinctProjects_IgnoresNullCWD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create chunk without CWD
	input := recall.ChunkInput{
		Content:        "Message without CWD",
		Source:         "claude",
		SourceID:       "no-cwd-msg",
		ConversationID: "conv-no-cwd",
		Metadata:       &recall.Metadata{Role: "user"},
	}
	_, _, err := storage.SaveWithSource(ctx, input, generateTestEmbedding(input.Content))
	require.NoError(t, err)

	// Create chunk with CWD
	input2 := recall.ChunkInput{
		Content:        "Message with CWD",
		Source:         "claude",
		SourceID:       "with-cwd-msg",
		ConversationID: "conv-with-cwd",
		Metadata: &recall.Metadata{
			Role: "user",
			CWD:  "/project/x",
		},
	}
	_, _, err = storage.SaveWithSource(ctx, input2, generateTestEmbedding(input2.Content))
	require.NoError(t, err)

	projects, err := storage.ListDistinctProjects(ctx)
	require.NoError(t, err)

	// Should only have the one with CWD
	assert.Len(t, projects, 1)
	assert.Equal(t, "/project/x", projects[0])
}
