package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zombiekit/brains/internal/memory"
)

func setupTestStorage(t *testing.T) *SQLiteStorage {
	t.Helper()

	// Create temp directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := NewSQLiteStorage(context.Background(), dbPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		storage.Close()
	})

	return storage
}

func TestGet_ExistingMemory(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Set a memory
	err := storage.Set(ctx, "test-key", "test content")
	require.NoError(t, err)

	// Get it back
	result, err := storage.Get(ctx, "test-key")
	require.NoError(t, err)
	require.True(t, result.HasValue(), "expected memory to exist")

	item := result.Value()
	assert.Equal(t, "test-key", item.Name)
	assert.Equal(t, "test content", item.Content)
	assert.Equal(t, 1, item.Version)
	assert.False(t, item.Deleted)
}

func TestGet_NonExistent_ReturnsNothing(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	result, err := storage.Get(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, result.HasValue(), "expected Nothing for nonexistent key")
}

func TestGet_DeletedMemory_ReturnsNothing(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Set and delete a memory
	err := storage.Set(ctx, "test-key", "test content")
	require.NoError(t, err)

	err = storage.Delete(ctx, "test-key")
	require.NoError(t, err)

	// Get should return Nothing
	result, err := storage.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.False(t, result.HasValue(), "expected Nothing for deleted key")
}

func TestSet_NewMemory_Version1(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	err := storage.Set(ctx, "new-key", "new content")
	require.NoError(t, err)

	result, err := storage.Get(ctx, "new-key")
	require.NoError(t, err)
	require.True(t, result.HasValue())

	assert.Equal(t, 1, result.Value().Version)
}

func TestSet_UpdateExisting_VersionIncrements(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Set initial version
	err := storage.Set(ctx, "test-key", "version 1")
	require.NoError(t, err)

	// Set again - should create version 2
	err = storage.Set(ctx, "test-key", "version 2")
	require.NoError(t, err)

	// Set again - should create version 3
	err = storage.Set(ctx, "test-key", "version 3")
	require.NoError(t, err)

	// Get should return latest version
	result, err := storage.Get(ctx, "test-key")
	require.NoError(t, err)
	require.True(t, result.HasValue())

	assert.Equal(t, 3, result.Value().Version)
	assert.Equal(t, "version 3", result.Value().Content)
}

func TestDelete_SoftDeletesAllVersions(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create multiple versions
	err := storage.Set(ctx, "test-key", "version 1")
	require.NoError(t, err)
	err = storage.Set(ctx, "test-key", "version 2")
	require.NoError(t, err)

	// Delete should mark all as deleted
	err = storage.Delete(ctx, "test-key")
	require.NoError(t, err)

	// Get should return Nothing
	result, err := storage.Get(ctx, "test-key")
	require.NoError(t, err)
	assert.False(t, result.HasValue())
}

func TestDelete_NonExistent_NoError(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Deleting nonexistent key should not error
	err := storage.Delete(ctx, "nonexistent")
	assert.NoError(t, err)
}

func TestList_ReturnsLatestVersionPerName(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create multiple memories with multiple versions
	require.NoError(t, storage.Set(ctx, "key-a", "a v1"))
	require.NoError(t, storage.Set(ctx, "key-a", "a v2"))
	require.NoError(t, storage.Set(ctx, "key-b", "b v1"))

	items, err := storage.List(ctx, "")
	require.NoError(t, err)

	// Should return 2 items (one per unique name)
	assert.Len(t, items, 2)

	// Verify versions are latest
	nameToVersion := make(map[string]int)
	for _, item := range items {
		nameToVersion[item.Name] = item.Version
	}
	assert.Equal(t, 2, nameToVersion["key-a"])
	assert.Equal(t, 1, nameToVersion["key-b"])
}

func TestList_OrderedByUpdatedAt(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create memories in order
	require.NoError(t, storage.Set(ctx, "first", "content"))
	require.NoError(t, storage.Set(ctx, "second", "content"))
	require.NoError(t, storage.Set(ctx, "third", "content"))

	items, err := storage.List(ctx, "")
	require.NoError(t, err)

	// Should be ordered by updated_at DESC (most recent first)
	require.Len(t, items, 3)
	assert.Equal(t, "third", items[0].Name)
	assert.Equal(t, "second", items[1].Name)
	assert.Equal(t, "first", items[2].Name)
}

func TestList_WithSearch_MatchesName(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	require.NoError(t, storage.Set(ctx, "apple-config", "some content"))
	require.NoError(t, storage.Set(ctx, "banana-config", "other content"))
	require.NoError(t, storage.Set(ctx, "orange-data", "more content"))

	items, err := storage.List(ctx, "config")
	require.NoError(t, err)

	assert.Len(t, items, 2)
	names := []string{items[0].Name, items[1].Name}
	assert.Contains(t, names, "apple-config")
	assert.Contains(t, names, "banana-config")
}

func TestList_WithSearch_MatchesContent(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	require.NoError(t, storage.Set(ctx, "key-a", "contains secret value"))
	require.NoError(t, storage.Set(ctx, "key-b", "contains nothing special"))
	require.NoError(t, storage.Set(ctx, "key-c", "just data"))

	items, err := storage.List(ctx, "secret")
	require.NoError(t, err)

	assert.Len(t, items, 1)
	assert.Equal(t, "key-a", items[0].Name)
}

func TestList_WithSearch_CaseInsensitive(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	require.NoError(t, storage.Set(ctx, "MyConfig", "content"))
	require.NoError(t, storage.Set(ctx, "other", "content"))

	// Search with different case
	items, err := storage.List(ctx, "myconfig")
	require.NoError(t, err)

	assert.Len(t, items, 1)
	assert.Equal(t, "MyConfig", items[0].Name)
}

func TestClear_SoftDeletesAll_ReturnsCount(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Create some memories
	require.NoError(t, storage.Set(ctx, "key-a", "content"))
	require.NoError(t, storage.Set(ctx, "key-a", "content v2"))
	require.NoError(t, storage.Set(ctx, "key-b", "content"))
	require.NoError(t, storage.Set(ctx, "key-c", "content"))

	// Clear should return count of distinct names
	count, err := storage.Clear(ctx)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// List should return empty
	items, err := storage.List(ctx, "")
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestSanitizesNameOnSet(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	// Set with invalid characters
	err := storage.Set(ctx, "my key with spaces!", "content")
	require.NoError(t, err)

	// Get with sanitized name
	result, err := storage.Get(ctx, "my_key_with_spaces_")
	require.NoError(t, err)
	require.True(t, result.HasValue())
	assert.Equal(t, "my_key_with_spaces_", result.Value().Name)
}

func TestNewSQLiteStorage_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "nested", "test.db")

	storage, err := NewSQLiteStorage(context.Background(), dbPath)
	require.NoError(t, err)
	defer storage.Close()

	// Verify the file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}

func TestStorage_ImplementsInterface(t *testing.T) {
	// Compile-time check that SQLiteStorage implements Storage interface
	var _ memory.Storage = (*SQLiteStorage)(nil)
}
