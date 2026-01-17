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

	poolConfig, err := pgxpool.ParseConfig(connStr)
	require.NoError(t, err)

	// Register pgvector types
	poolConfig.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	require.NoError(t, err)

	t.Cleanup(func() {
		pool.Close()
	})

	// Run migrations
	_, err = pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS vector`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS recall_chunks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			content TEXT NOT NULL,
			content_hash TEXT NOT NULL,
			embedding vector(768),
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_recall_chunks_content_hash ON recall_chunks(content_hash)`)
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
