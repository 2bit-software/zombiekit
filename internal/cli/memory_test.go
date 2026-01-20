package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"

	"github.com/zombiekit/brains/internal/memory/sqlite"
)

func setupMemoryTest(t *testing.T) (*sqlite.SQLiteStorage, string) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := sqlite.NewSQLiteStorage(context.Background(), dbPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		storage.Close()
	})

	return storage, dbPath
}

func runMemoryCmd(t *testing.T, dbPath string, args ...string) (string, error) {
	t.Helper()

	var buf bytes.Buffer
	app := &cli.App{
		Name:   "brains",
		Writer: &buf,
		Commands: []*cli.Command{
			newMemoryCommand(),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "db-path",
				Value: dbPath,
			},
		},
	}

	// Set env vars for test isolation - must override any .env settings
	os.Setenv("BRAINS_BACKEND", "sqlite")
	os.Setenv("BRAINS_SQLITE_PATH", dbPath)
	defer func() {
		os.Unsetenv("BRAINS_BACKEND")
		os.Unsetenv("BRAINS_SQLITE_PATH")
	}()

	allArgs := append([]string{"brains", "memory"}, args...)
	err := app.Run(allArgs)

	return buf.String(), err
}

func TestMemoryList_Success(t *testing.T) {
	storage, dbPath := setupMemoryTest(t)
	ctx := context.Background()

	// Add some memories
	storage.Set(ctx, "key-a", "content a")
	storage.Set(ctx, "key-b", "content b")

	output, err := runMemoryCmd(t, dbPath, "list")
	require.NoError(t, err)

	assert.Contains(t, output, "key-a")
	assert.Contains(t, output, "key-b")
}

func TestMemoryList_JSONFormat(t *testing.T) {
	storage, dbPath := setupMemoryTest(t)
	ctx := context.Background()

	storage.Set(ctx, "key-a", "content a")

	output, err := runMemoryCmd(t, dbPath, "list", "--format", "json")
	require.NoError(t, err)

	// Should be valid JSON array
	var items []interface{}
	err = json.Unmarshal([]byte(output), &items)
	require.NoError(t, err)
	assert.Len(t, items, 1)
}

func TestMemoryGet_Success(t *testing.T) {
	storage, dbPath := setupMemoryTest(t)
	ctx := context.Background()

	storage.Set(ctx, "my-key", "my content here")

	output, err := runMemoryCmd(t, dbPath, "get", "my-key")
	require.NoError(t, err)

	assert.Contains(t, output, "my content here")
}

func TestMemoryGet_NotFound(t *testing.T) {
	_, dbPath := setupMemoryTest(t)

	_, err := runMemoryCmd(t, dbPath, "get", "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestMemorySet_Success(t *testing.T) {
	_, dbPath := setupMemoryTest(t)

	output, err := runMemoryCmd(t, dbPath, "set", "new-key", "new content")
	require.NoError(t, err)

	assert.Contains(t, strings.ToLower(output), "saved")

	// Verify we can get it back
	output, err = runMemoryCmd(t, dbPath, "get", "new-key")
	require.NoError(t, err)
	assert.Contains(t, output, "new content")
}

func TestMemoryDelete_Success(t *testing.T) {
	storage, dbPath := setupMemoryTest(t)
	ctx := context.Background()

	storage.Set(ctx, "to-delete", "content")

	output, err := runMemoryCmd(t, dbPath, "delete", "to-delete")
	require.NoError(t, err)

	assert.Contains(t, strings.ToLower(output), "deleted")

	// Verify it's gone
	_, err = runMemoryCmd(t, dbPath, "get", "to-delete")
	assert.Error(t, err)
}

func TestMemorySearch_Success(t *testing.T) {
	storage, dbPath := setupMemoryTest(t)
	ctx := context.Background()

	storage.Set(ctx, "config-a", "content a")
	storage.Set(ctx, "config-b", "content b")
	storage.Set(ctx, "data-c", "content c")

	output, err := runMemoryCmd(t, dbPath, "search", "config")
	require.NoError(t, err)

	assert.Contains(t, output, "config-a")
	assert.Contains(t, output, "config-b")
	assert.NotContains(t, output, "data-c")
}

func TestMemoryClear_Success(t *testing.T) {
	storage, dbPath := setupMemoryTest(t)
	ctx := context.Background()

	storage.Set(ctx, "key-a", "content")
	storage.Set(ctx, "key-b", "content")

	output, err := runMemoryCmd(t, dbPath, "clear", "--force")
	require.NoError(t, err)

	assert.Contains(t, output, "2")

	// Verify all are gone
	output, err = runMemoryCmd(t, dbPath, "list")
	require.NoError(t, err)
	// Should show no items or empty message
	assert.NotContains(t, output, "key-a")
}
