package recall_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zombiekit/brains/internal/recall"
	"github.com/zombiekit/brains/internal/search"
	recallweb "github.com/zombiekit/brains/internal/webplugins/recall"
)

// mockStorage implements recall.Storage for testing.
type mockStorage struct {
	chunks        []recall.Chunk
	conversations map[string][]recall.Chunk
	projects      []string
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		chunks:        []recall.Chunk{},
		conversations: make(map[string][]recall.Chunk),
		projects:      []string{},
	}
}

func (m *mockStorage) Save(ctx context.Context, content string, embedding []float32) (string, bool, error) {
	return "", false, nil
}

func (m *mockStorage) SaveWithSource(ctx context.Context, input recall.ChunkInput, embedding []float32) (string, bool, error) {
	return "", false, nil
}

func (m *mockStorage) ExistsBySourceID(ctx context.Context, source, sourceID string) (bool, error) {
	return false, nil
}

func (m *mockStorage) GetByConversation(ctx context.Context, conversationID string) ([]recall.Chunk, error) {
	return m.conversations[conversationID], nil
}

func (m *mockStorage) List(ctx context.Context, limit int) ([]recall.Chunk, error) {
	return m.chunks, nil
}

func (m *mockStorage) Search(ctx context.Context, embedding []float32, limit int) ([]recall.SearchResult, error) {
	var results []recall.SearchResult
	for _, chunk := range m.chunks {
		results = append(results, recall.SearchResult{
			Chunk:      chunk,
			Similarity: 0.8,
		})
	}
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

func (m *mockStorage) ListConversations(ctx context.Context, limit, offset int, project string) ([]recall.ConversationSummary, error) {
	return []recall.ConversationSummary{}, nil
}

func (m *mockStorage) ListDistinctProjects(ctx context.Context) ([]string, error) {
	return m.projects, nil
}

func (m *mockStorage) GetImportState(ctx context.Context, filePath string) (*recall.ImportState, error) {
	return nil, nil
}

func (m *mockStorage) SaveImportState(ctx context.Context, state *recall.ImportState) error {
	return nil
}

func (m *mockStorage) DeleteImportState(ctx context.Context, filePath string) error {
	return nil
}

func (m *mockStorage) CleanupStaleImportStates(ctx context.Context, validPaths []string) error {
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}

// addConversation adds a test conversation to the mock storage
func (m *mockStorage) addConversation(convID string, messages []struct{ content, role string }) {
	for i, msg := range messages {
		chunk := recall.Chunk{
			ID:             "chunk-" + convID + "-" + string(rune('a'+i)),
			Content:        msg.content,
			CreatedAt:      time.Now(),
			ConversationID: convID,
			Metadata: &recall.Metadata{
				Role: msg.role,
			},
		}
		m.chunks = append(m.chunks, chunk)
		m.conversations[convID] = append(m.conversations[convID], chunk)
	}
}

// mockEmbedder implements recall.Embedder for testing
type mockEmbedder struct{}

func (m *mockEmbedder) Embed(ctx context.Context, text string, purpose recall.EmbedPurpose) ([]float32, error) {
	// Return a simple test embedding
	embedding := make([]float32, 768)
	for i := 0; i < len(text) && i < 768; i++ {
		embedding[i] = float32(text[i]) / 255.0
	}
	return embedding, nil
}

// === Search interface tests ===

func TestSearchEmptyQuery(t *testing.T) {
	storage := newMockStorage()
	storage.addConversation("conv-1", []struct{ content, role string }{
		{"Hello", "user"},
		{"Hi there", "assistant"},
	})
	plugin := recallweb.NewPlugin(storage, &mockEmbedder{})

	results, err := plugin.Search("", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Empty(t, results, "empty query should return empty results")
	assert.NotNil(t, results, "results should never be nil")
}

func TestSearchNilEmbedder(t *testing.T) {
	storage := newMockStorage()
	storage.addConversation("conv-1", []struct{ content, role string }{
		{"Hello", "user"},
	})
	// Plugin with nil embedder
	plugin := recallweb.NewPlugin(storage, nil)

	results, err := plugin.Search("hello", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Empty(t, results, "nil embedder should return empty results silently")
	assert.NotNil(t, results, "results should never be nil")
}

func TestSearchReturnsResults(t *testing.T) {
	storage := newMockStorage()
	storage.addConversation("conv-1", []struct{ content, role string }{
		{"How do I fix this bug?", "user"},
		{"Let me help you with that", "assistant"},
	})
	storage.addConversation("conv-2", []struct{ content, role string }{
		{"What's the weather?", "user"},
	})
	plugin := recallweb.NewPlugin(storage, &mockEmbedder{})

	results, err := plugin.Search("bug fix", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.NotEmpty(t, results, "should return results")
}

func TestSearchDeduplicatesConversations(t *testing.T) {
	storage := newMockStorage()
	// Add same conversation with multiple chunks
	storage.addConversation("conv-1", []struct{ content, role string }{
		{"First message", "user"},
		{"Second message", "assistant"},
		{"Third message", "user"},
	})
	plugin := recallweb.NewPlugin(storage, &mockEmbedder{})

	results, err := plugin.Search("message", 10, search.SortRelevance)
	require.NoError(t, err)
	// Should only return 1 result (deduplicated by conversation ID)
	assert.Len(t, results, 1, "should deduplicate by conversation")
}

func TestSearchURLFormat(t *testing.T) {
	storage := newMockStorage()
	storage.addConversation("abc-123", []struct{ content, role string }{
		{"Test content", "user"},
	})
	plugin := recallweb.NewPlugin(storage, &mockEmbedder{})

	results, err := plugin.Search("test", 10, search.SortRelevance)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "/abc-123", results[0].URL, "URL should be relative - system prefixes automatically")
}

func TestSearchMaxResults(t *testing.T) {
	storage := newMockStorage()
	// Add multiple conversations
	for i := 0; i < 15; i++ {
		storage.addConversation("conv-"+string(rune('a'+i)), []struct{ content, role string }{
			{"Content " + string(rune('a'+i)), "user"},
		})
	}
	plugin := recallweb.NewPlugin(storage, &mockEmbedder{})

	results, err := plugin.Search("content", 5, search.SortRelevance)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(results), 5, "should respect maxResults limit")
}

// === Plugin interface tests ===

func TestPluginImplementsSearchable(t *testing.T) {
	// Compile-time check
	var _ search.Searchable = (*recallweb.Plugin)(nil)
}

func TestSidebarItems(t *testing.T) {
	storage := newMockStorage()
	plugin := recallweb.NewPlugin(storage, nil)

	items := plugin.SidebarItems()
	require.Len(t, items, 1)

	item := items[0]
	assert.Equal(t, "conversations", item.ID)
	assert.Equal(t, "Conversations", item.Label)
	assert.Equal(t, "/", item.Path)
	assert.Equal(t, 30, item.Order, "order should be 30 (after memory)")
}
