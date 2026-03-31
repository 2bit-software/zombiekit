package claude_test

import (
	"context"
	"os"
	"path/filepath"
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

	"github.com/2bit-software/zombiekit/internal/recall"
	"github.com/2bit-software/zombiekit/internal/recall/claude"
	"github.com/2bit-software/zombiekit/internal/recall/postgres"
)

// testHarness provides a complete test environment for import integration tests.
type testHarness struct {
	storage  *postgres.Storage
	embedder *mockEmbedder
	tmpDir   string
}

// mockEmbedder returns deterministic embeddings for testing.
type mockEmbedder struct{}

func (m *mockEmbedder) Embed(ctx context.Context, text string, purpose recall.EmbedPurpose) ([]float32, error) {
	// Generate deterministic embedding based on text content
	embedding := make([]float32, 768)
	for i := range 768 {
		if i < len(text) {
			embedding[i] = float32(text[i%len(text)]) / 255.0
		} else {
			embedding[i] = 0.1
		}
	}
	return embedding, nil
}

func setupTestHarness(t *testing.T) *testHarness {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Create postgres testcontainer
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

	// Create extension first
	initPool, err := pgxpool.New(ctx, connStr)
	require.NoError(t, err)
	_, err = initPool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS vector`)
	require.NoError(t, err)
	initPool.Close()

	// Create pool with pgvector types
	poolConfig, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err)
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	// Create schema
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
			metadata JSONB,
			history_gap BOOLEAN NOT NULL DEFAULT FALSE
		)
	`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_recall_chunks_content_hash ON recall_chunks(content_hash)`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_recall_chunks_source_id ON recall_chunks(source, source_id) WHERE source_id IS NOT NULL`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `CREATE INDEX IF NOT EXISTS idx_recall_chunks_conversation ON recall_chunks(conversation_id) WHERE conversation_id IS NOT NULL`)
	require.NoError(t, err)

	storage := postgres.NewStorageWithPool(pool)

	// Create temp directory for test history files
	tmpDir := t.TempDir()

	return &testHarness{
		storage:  storage,
		embedder: &mockEmbedder{},
		tmpDir:   tmpDir,
	}
}

// createHistoryFile creates a test .jsonl history file and returns its path.
func createHistoryFile(t *testing.T, baseDir, projectPath string, entries []string) string {
	t.Helper()

	projectsDir := filepath.Join(baseDir, "projects", claude.EncodeProjectPath(projectPath))
	err := os.MkdirAll(projectsDir, 0755)
	require.NoError(t, err)

	filePath := filepath.Join(projectsDir, "test-session.jsonl")
	content := ""
	for _, entry := range entries {
		content += entry + "\n"
	}

	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	return filePath
}

// importFile imports a single history file into storage.
func (h *testHarness) importFile(ctx context.Context, filePath string) (newCount, skipCount int, err error) {
	entries, err := claude.ParseFile(filePath)
	if err != nil {
		return 0, 0, err
	}

	importable := claude.FilterImportable(entries)

	for _, entry := range importable {
		exists, err := h.storage.ExistsBySourceID(ctx, "claude", entry.UUID)
		if err != nil {
			return newCount, skipCount, err
		}
		if exists {
			skipCount++
			continue
		}

		content := claude.ExtractContent(entry)
		if content == "" {
			continue
		}

		chunks := claude.ChunkMessage(content)
		for i, chunkContent := range chunks {
			sourceID := claude.ChunkSourceID(entry.UUID, i, len(chunks))

			embedding, err := h.embedder.Embed(ctx, chunkContent, recall.PurposeDocument)
			if err != nil {
				return newCount, skipCount, err
			}

			var parentID string
			if entry.ParentUUID != nil {
				parentID = *entry.ParentUUID
			}

			input := recall.ChunkInput{
				Content:        chunkContent,
				Source:         "claude",
				SourceID:       sourceID,
				ConversationID: entry.SessionID,
				Metadata: &recall.Metadata{
					Role:      entry.Message.Role,
					Timestamp: entry.Timestamp,
					GitBranch: entry.GitBranch,
					CWD:       entry.CWD,
					ParentID:  parentID,
				},
			}

			_, created, err := h.storage.SaveWithSource(ctx, input, embedding)
			if err != nil {
				return newCount, skipCount, err
			}

			if created {
				newCount++
			} else {
				skipCount++
			}
		}
	}

	return newCount, skipCount, nil
}

// ============================================================
// T018: Integration tests for import flow
// BR-001 (Import history)
// ============================================================

func TestImport_RealHistoryFile(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	// Create a realistic history file
	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Hello, how are you?"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-002","parentUuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:05Z","message":{"role":"assistant","content":"I'm doing well, thank you! How can I help you today?"},"isMeta":false,"isSidechain":false}`,
	}

	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	// Run import
	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	newCount, skipCount, err := h.importFile(ctx, files[0])
	require.NoError(t, err)

	assert.Equal(t, 2, newCount, "expected 2 new messages")
	assert.Equal(t, 0, skipCount, "expected 0 skipped messages")
}

func TestImport_MultipleFiles(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	// Create two project directories with files
	entries1 := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"First project message"},"isMeta":false,"isSidechain":false}`,
	}
	entries2 := []string{
		`{"type":"user","uuid":"msg-002","sessionId":"session-2","timestamp":"2024-01-15T11:00:00Z","message":{"role":"user","content":"Second project message"},"isMeta":false,"isSidechain":false}`,
	}

	createHistoryFile(t, h.tmpDir, "/Users/test/project1", entries1)
	createHistoryFile(t, h.tmpDir, "/Users/test/project2", entries2)

	// Import all files
	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)
	require.Len(t, files, 2)

	totalNew := 0
	for _, f := range files {
		newCount, _, err := h.importFile(ctx, f)
		require.NoError(t, err)
		totalNew += newCount
	}

	assert.Equal(t, 2, totalNew, "expected 2 total messages from both files")
}

func TestImport_LargeMessage(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	// Create a message larger than MaxChunkSize
	largeContent := make([]byte, claude.MaxChunkSize+1000)
	for i := range largeContent {
		largeContent[i] = byte('a' + (i % 26))
	}

	entries := []string{
		`{"type":"user","uuid":"large-msg","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"` + string(largeContent) + `"},"isMeta":false,"isSidechain":false}`,
	}

	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)

	newCount, _, err := h.importFile(ctx, files[0])
	require.NoError(t, err)

	// Should create multiple chunks
	assert.GreaterOrEqual(t, newCount, 2, "expected at least 2 chunks for large message")
}

// ============================================================
// BR-002 (No duplicates)
// ============================================================

func TestImport_RerunSkipsDuplicates(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Test message"},"isMeta":false,"isSidechain":false}`,
	}

	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)

	// First import
	newCount1, skipCount1, err := h.importFile(ctx, files[0])
	require.NoError(t, err)
	assert.Equal(t, 1, newCount1, "first run should create 1 message")
	assert.Equal(t, 0, skipCount1, "first run should skip 0 messages")

	// Second import (should skip)
	newCount2, skipCount2, err := h.importFile(ctx, files[0])
	require.NoError(t, err)
	assert.Equal(t, 0, newCount2, "rerun should create 0 messages")
	assert.Equal(t, 1, skipCount2, "rerun should skip 1 message")
}

func TestImport_PartialRerun(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	// First import with one message
	entries1 := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"First message"},"isMeta":false,"isSidechain":false}`,
	}

	filePath := createHistoryFile(t, h.tmpDir, "/Users/test/project", entries1)

	newCount, _, err := h.importFile(ctx, filePath)
	require.NoError(t, err)
	assert.Equal(t, 1, newCount)

	// Add a new message to the file
	entries2 := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"First message"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-002","parentUuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:05Z","message":{"role":"assistant","content":"New message"},"isMeta":false,"isSidechain":false}`,
	}

	// Rewrite file with additional message
	content := ""
	for _, entry := range entries2 {
		content += entry + "\n"
	}
	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	// Partial rerun should only import new message
	newCount2, skipCount2, err := h.importFile(ctx, filePath)
	require.NoError(t, err)
	assert.Equal(t, 1, newCount2, "should import only the new message")
	assert.Equal(t, 1, skipCount2, "should skip the existing message")
}

func TestImport_DuplicateAcrossFiles(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	// Same message in two files (same UUID)
	entries := []string{
		`{"type":"user","uuid":"shared-uuid","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Shared message"},"isMeta":false,"isSidechain":false}`,
	}

	createHistoryFile(t, h.tmpDir, "/Users/test/project1", entries)
	createHistoryFile(t, h.tmpDir, "/Users/test/project2", entries)

	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)
	require.Len(t, files, 2)

	totalNew := 0
	totalSkip := 0
	for _, f := range files {
		newCount, skipCount, err := h.importFile(ctx, f)
		require.NoError(t, err)
		totalNew += newCount
		totalSkip += skipCount
	}

	assert.Equal(t, 1, totalNew, "should import only one copy")
	assert.Equal(t, 1, totalSkip, "should skip duplicate")
}

// ============================================================
// BR-003 (Manual trigger)
// ============================================================

func TestImport_OnceMode(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Test message"},"isMeta":false,"isSidechain":false}`,
	}

	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)

	// Single import operation (manual trigger)
	newCount, _, err := h.importFile(ctx, files[0])
	require.NoError(t, err)
	assert.Equal(t, 1, newCount, "manual import should work")
}

// ============================================================
// BR-004 (Progress feedback)
// ============================================================

func TestImport_ReportsProgress(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Message 1"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-002","sessionId":"session-1","timestamp":"2024-01-15T10:00:05Z","message":{"role":"assistant","content":"Message 2"},"isMeta":false,"isSidechain":false}`,
		`{"type":"user","uuid":"msg-003","sessionId":"session-1","timestamp":"2024-01-15T10:00:10Z","message":{"role":"user","content":"Message 3"},"isMeta":false,"isSidechain":false}`,
	}

	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)

	newCount, skipCount, err := h.importFile(ctx, files[0])
	require.NoError(t, err)

	// Verify we get accurate counts for progress reporting
	assert.Equal(t, 3, newCount, "should report correct new count")
	assert.Equal(t, 0, skipCount, "should report correct skip count")

	// Second run should report skips
	newCount2, skipCount2, err := h.importFile(ctx, files[0])
	require.NoError(t, err)
	assert.Equal(t, 0, newCount2, "rerun should report 0 new")
	assert.Equal(t, 3, skipCount2, "rerun should report 3 skipped")
}

// ============================================================
// BR-006 (Preserve structure) - Content extraction
// ============================================================

func TestImport_PreservesContentBlocks(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	// Message with content blocks array
	entries := []string{
		`{"type":"assistant","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"assistant","content":[{"type":"text","text":"First paragraph."},{"type":"text","text":"Second paragraph."}]},"isMeta":false,"isSidechain":false}`,
	}

	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)

	newCount, _, err := h.importFile(ctx, files[0])
	require.NoError(t, err)
	assert.Equal(t, 1, newCount)

	// Verify content was extracted and combined
	chunks, err := h.storage.GetByConversation(ctx, "session-1")
	require.NoError(t, err)
	require.Len(t, chunks, 1)

	assert.Contains(t, chunks[0].Content, "First paragraph")
	assert.Contains(t, chunks[0].Content, "Second paragraph")
}

func TestImport_SkipsToolBlocks(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	// Message with tool_use and tool_result blocks mixed with text
	entries := []string{
		`{"type":"assistant","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"assistant","content":[{"type":"text","text":"Let me help you."},{"type":"tool_use","id":"toolu_1","name":"read_file","input":{"path":"test.txt"}},{"type":"text","text":"Here is the result."}]},"isMeta":false,"isSidechain":false}`,
	}

	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)

	newCount, _, err := h.importFile(ctx, files[0])
	require.NoError(t, err)
	assert.Equal(t, 1, newCount)

	chunks, err := h.storage.GetByConversation(ctx, "session-1")
	require.NoError(t, err)
	require.Len(t, chunks, 1)

	// Should have text but not tool content
	assert.Contains(t, chunks[0].Content, "Let me help you")
	assert.Contains(t, chunks[0].Content, "Here is the result")
	assert.NotContains(t, chunks[0].Content, "tool_use")
	assert.NotContains(t, chunks[0].Content, "read_file")
}

// ============================================================
// BR-008 (Unique identifiers)
// ============================================================

func TestImport_PreservesUUIDAsSourceID(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	entries := []string{
		`{"type":"user","uuid":"unique-test-uuid-123","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Test message"},"isMeta":false,"isSidechain":false}`,
	}

	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)

	_, _, err = h.importFile(ctx, files[0])
	require.NoError(t, err)

	// Verify source_id matches original UUID
	exists, err := h.storage.ExistsBySourceID(ctx, "claude", "unique-test-uuid-123")
	require.NoError(t, err)
	assert.True(t, exists, "source_id should match original UUID")
}

// ============================================================
// BR-009/BR-010 (Conversation retrieval)
// ============================================================

func TestImport_PreservesConversationID(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"test-conversation-abc","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Hello"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-002","sessionId":"test-conversation-abc","timestamp":"2024-01-15T10:00:05Z","message":{"role":"assistant","content":"Hi there!"},"isMeta":false,"isSidechain":false}`,
	}

	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)

	_, _, err = h.importFile(ctx, files[0])
	require.NoError(t, err)

	// Retrieve by conversation
	chunks, err := h.storage.GetByConversation(ctx, "test-conversation-abc")
	require.NoError(t, err)
	assert.Len(t, chunks, 2, "should retrieve both messages by conversation_id")
}

func TestImport_PreservesMetadata(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Test"},"isMeta":false,"isSidechain":false,"cwd":"/home/user/project","gitBranch":"feature/test"}`,
	}

	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)

	_, _, err = h.importFile(ctx, files[0])
	require.NoError(t, err)

	chunks, err := h.storage.GetByConversation(ctx, "session-1")
	require.NoError(t, err)
	require.Len(t, chunks, 1)

	meta := chunks[0].Metadata
	require.NotNil(t, meta)
	assert.Equal(t, "user", meta.Role)
	assert.Equal(t, "/home/user/project", meta.CWD)
	assert.Equal(t, "feature/test", meta.GitBranch)
}
