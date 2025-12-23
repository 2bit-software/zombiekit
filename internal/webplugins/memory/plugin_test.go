package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zombiekit/brains/internal/memory"
	"github.com/zombiekit/brains/internal/mo"
	"github.com/zombiekit/brains/internal/search"
	memoryPlugin "github.com/zombiekit/brains/internal/webplugins/memory"
)

// mockStorage implements memory.Storage for testing.
type mockStorage struct {
	items map[string]memory.MemoryItem
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		items: make(map[string]memory.MemoryItem),
	}
}

func (m *mockStorage) Set(ctx context.Context, name, content string) error {
	now := time.Now()
	if existing, ok := m.items[name]; ok {
		m.items[name] = memory.MemoryItem{
			Name:      name,
			Content:   content,
			Version:   existing.Version + 1,
			Deleted:   false,
			CreatedAt: existing.CreatedAt,
			UpdatedAt: now,
		}
	} else {
		m.items[name] = memory.MemoryItem{
			Name:      name,
			Content:   content,
			Version:   1,
			Deleted:   false,
			CreatedAt: now,
			UpdatedAt: now,
		}
	}
	return nil
}

func (m *mockStorage) Get(ctx context.Context, name string) (mo.Maybe[memory.MemoryItem], error) {
	if item, ok := m.items[name]; ok && !item.Deleted {
		return mo.Just(item), nil
	}
	return mo.Nothing[memory.MemoryItem](), nil
}

func (m *mockStorage) Delete(ctx context.Context, name string) error {
	if item, ok := m.items[name]; ok {
		item.Deleted = true
		m.items[name] = item
	}
	return nil
}

func (m *mockStorage) List(ctx context.Context, search string) ([]memory.MemoryMetadata, error) {
	var result []memory.MemoryMetadata
	for name, item := range m.items {
		if item.Deleted {
			continue
		}
		result = append(result, memory.MemoryMetadata{
			Name:      name,
			Size:      len(item.Content),
			Version:   item.Version,
			CreatedAt: item.CreatedAt,
			UpdatedAt: item.UpdatedAt,
		})
	}
	return result, nil
}

func (m *mockStorage) Clear(ctx context.Context) (int, error) {
	count := len(m.items)
	m.items = make(map[string]memory.MemoryItem)
	return count, nil
}

func (m *mockStorage) Close() error {
	return nil
}

// === Basic search tests ===

func TestSearchEmptyQuery(t *testing.T) {
	storage := newMockStorage()
	storage.Set(context.Background(), "test", "content")
	plugin := memoryPlugin.NewPlugin(storage)

	results, err := plugin.Search("", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Empty(t, results, "empty query should return empty results")
	assert.NotNil(t, results, "results should never be nil")
}

func TestSearchNoMatches(t *testing.T) {
	storage := newMockStorage()
	storage.Set(context.Background(), "test", "content")
	plugin := memoryPlugin.NewPlugin(storage)

	results, err := plugin.Search("nonexistent", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Empty(t, results, "no matches should return empty results")
	assert.NotNil(t, results, "results should never be nil")
}

func TestSearchMatchesName(t *testing.T) {
	storage := newMockStorage()
	storage.Set(context.Background(), "developer-notes", "some content")
	storage.Set(context.Background(), "design-docs", "other content")
	plugin := memoryPlugin.NewPlugin(storage)

	results, err := plugin.Search("dev", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Len(t, results, 1, "should match 'developer-notes'")
	assert.Equal(t, "developer-notes", results[0].Title)
}

func TestSearchMatchesContent(t *testing.T) {
	storage := newMockStorage()
	storage.Set(context.Background(), "notes", "React and Vue development")
	storage.Set(context.Background(), "docs", "Python documentation")
	plugin := memoryPlugin.NewPlugin(storage)

	results, err := plugin.Search("react", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Len(t, results, 1, "should match 'notes' by content")
	assert.Equal(t, "notes", results[0].Title)
}

func TestSearchCaseInsensitive(t *testing.T) {
	storage := newMockStorage()
	storage.Set(context.Background(), "DevNotes", "IMPORTANT info")
	plugin := memoryPlugin.NewPlugin(storage)

	testCases := []string{"devnotes", "DEVNOTES", "DevNotes", "devNOTES"}
	for _, query := range testCases {
		results, err := plugin.Search(query, 10, search.SortRelevance)
		require.NoError(t, err, "query: %s", query)
		assert.Len(t, results, 1, "should find result for query: %s", query)
	}

	// Also test case-insensitive content search
	results, err := plugin.Search("important", 10, search.SortRelevance)
	require.NoError(t, err)
	assert.Len(t, results, 1, "should find 'IMPORTANT' with lowercase query")
}

func TestSearchMaxResults(t *testing.T) {
	storage := newMockStorage()
	for i := 1; i <= 5; i++ {
		storage.Set(context.Background(), "dev"+string(rune('0'+i)), "content")
	}
	plugin := memoryPlugin.NewPlugin(storage)

	results, err := plugin.Search("dev", 3, search.SortRelevance)
	require.NoError(t, err)
	assert.Len(t, results, 3, "should respect maxResults limit")
}

func TestSearchURLFormat(t *testing.T) {
	storage := newMockStorage()
	storage.Set(context.Background(), "my-memory", "test content")
	plugin := memoryPlugin.NewPlugin(storage)

	results, err := plugin.Search("memory", 10, search.SortRelevance)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "/my-memory", results[0].URL, "URL should be relative - system prefixes automatically")
}

func TestSearchSortByName(t *testing.T) {
	storage := newMockStorage()
	storage.Set(context.Background(), "zebra-dev", "developer")
	storage.Set(context.Background(), "alpha-dev", "developer")
	storage.Set(context.Background(), "middle-dev", "developer")
	plugin := memoryPlugin.NewPlugin(storage)

	results, err := plugin.Search("dev", 10, search.SortName)
	require.NoError(t, err)
	require.Len(t, results, 3)

	// Should be sorted A-Z
	assert.Equal(t, "alpha-dev", results[0].Title)
	assert.Equal(t, "middle-dev", results[1].Title)
	assert.Equal(t, "zebra-dev", results[2].Title)
}

func TestSearchSortByCreatedDate(t *testing.T) {
	storage := newMockStorage()
	ctx := context.Background()

	// Create items with different timestamps
	storage.Set(ctx, "older", "dev")
	time.Sleep(10 * time.Millisecond)
	storage.Set(ctx, "newer", "dev")

	plugin := memoryPlugin.NewPlugin(storage)

	results, err := plugin.Search("dev", 10, search.SortCreatedDate)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Newest first
	assert.Equal(t, "newer", results[0].Title)
	assert.Equal(t, "older", results[1].Title)
}

func TestSearchSortByUpdatedDate(t *testing.T) {
	storage := newMockStorage()
	ctx := context.Background()

	// Create both items
	storage.Set(ctx, "first", "dev")
	time.Sleep(10 * time.Millisecond)
	storage.Set(ctx, "second", "dev")
	time.Sleep(10 * time.Millisecond)
	// Update first item to make it most recently updated
	storage.Set(ctx, "first", "updated dev")

	plugin := memoryPlugin.NewPlugin(storage)

	results, err := plugin.Search("dev", 10, search.SortUpdatedDate)
	require.NoError(t, err)
	require.Len(t, results, 2)

	// Most recently updated first
	assert.Equal(t, "first", results[0].Title)
	assert.Equal(t, "second", results[1].Title)
}
