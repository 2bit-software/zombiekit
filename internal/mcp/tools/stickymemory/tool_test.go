package stickymemory

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2bit-software/zombiekit/internal/memory/sqlite"
)

func setupTestTool(t *testing.T) *Tool {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := sqlite.NewSQLiteStorage(context.Background(), dbPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		storage.Close()
	})

	return NewTool(storage)
}

func TestTool_Definition_MatchesContract(t *testing.T) {
	tool := setupTestTool(t)

	def := tool.Definition()

	assert.Equal(t, "stickymemory", def.Name)
	assert.Contains(t, def.Description, "memory")

	// Verify required properties exist
	schema := def.InputSchema
	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok, "expected properties in schema")

	assert.Contains(t, props, "operation")
	assert.Contains(t, props, "name")
	assert.Contains(t, props, "content")
	assert.Contains(t, props, "limit")
}

func TestTool_Get_ValidRequest(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	// First set a value
	_, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "set",
		"name":      "test-key",
		"content":   "test content",
	})
	require.NoError(t, err)

	// Then get it
	result, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "get",
		"name":      "test-key",
	})
	require.NoError(t, err)

	var response map[string]interface{}
	err = json.Unmarshal([]byte(result), &response)
	require.NoError(t, err)

	assert.Equal(t, "test-key", response["name"])
	assert.Equal(t, "test content", response["content"])
	assert.Equal(t, float64(1), response["version"])
}

func TestTool_Get_MissingName_Error(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "get",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestTool_Get_NotFound_Error(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "get",
		"name":      "nonexistent",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestTool_Set_ValidRequest(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	result, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "set",
		"name":      "new-key",
		"content":   "new content",
	})
	require.NoError(t, err)

	var response map[string]interface{}
	err = json.Unmarshal([]byte(result), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])
	assert.Equal(t, "new-key", response["name"])
	assert.Equal(t, float64(1), response["version"])
}

func TestTool_Set_MissingName_Error(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "set",
		"content":   "some content",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "name")
}

func TestTool_Set_MissingContent_Error(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "set",
		"name":      "some-key",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "content")
}

func TestTool_Set_ContentTooLarge_Error(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	// Create content larger than 1MB
	largeContent := make([]byte, 1048577) // 1MB + 1 byte
	for i := range largeContent {
		largeContent[i] = 'x'
	}

	_, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "set",
		"name":      "large-key",
		"content":   string(largeContent),
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too large")
}

func TestTool_List_ValidRequest(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	// Create some memories
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "key-a", "content": "a"})
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "key-b", "content": "b"})

	result, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "list",
	})
	require.NoError(t, err)

	var response []interface{}
	err = json.Unmarshal([]byte(result), &response)
	require.NoError(t, err)

	assert.Len(t, response, 2)
}

func TestTool_List_WithLimit(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	// Create some memories
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "key-a", "content": "a"})
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "key-b", "content": "b"})
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "key-c", "content": "c"})

	result, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "list",
		"limit":     float64(2),
	})
	require.NoError(t, err)

	var response []interface{}
	err = json.Unmarshal([]byte(result), &response)
	require.NoError(t, err)

	assert.Len(t, response, 2)
}

func TestTool_Delete_ValidRequest(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	// Create a memory
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "to-delete", "content": "content"})

	result, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "delete",
		"name":      "to-delete",
	})
	require.NoError(t, err)

	var response map[string]interface{}
	err = json.Unmarshal([]byte(result), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])
	assert.Equal(t, "to-delete", response["name"])
}

func TestTool_Search_ValidRequest(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	// Create some memories
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "config-a", "content": "a"})
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "data-b", "content": "b"})

	result, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "search",
		"name":      "config",
	})
	require.NoError(t, err)

	var response []interface{}
	err = json.Unmarshal([]byte(result), &response)
	require.NoError(t, err)

	assert.Len(t, response, 1)
}

func TestTool_Search_WithLimit(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	// Create some memories
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "config-a", "content": "a"})
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "config-b", "content": "b"})
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "config-c", "content": "c"})

	result, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "search",
		"name":      "config",
		"limit":     float64(2),
	})
	require.NoError(t, err)

	var response []interface{}
	err = json.Unmarshal([]byte(result), &response)
	require.NoError(t, err)

	assert.Len(t, response, 2)
}

func TestTool_Clear_ValidRequest(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	// Create some memories
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "key-a", "content": "a"})
	tool.Execute(ctx, map[string]interface{}{"operation": "set", "name": "key-b", "content": "b"})

	result, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "clear",
	})
	require.NoError(t, err)

	var response map[string]interface{}
	err = json.Unmarshal([]byte(result), &response)
	require.NoError(t, err)

	assert.Equal(t, true, response["success"])
	assert.Equal(t, float64(2), response["count"])
}

func TestTool_InvalidOperation_Error(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "invalid",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation")
}

func TestTool_MissingOperation_Error(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	_, err := tool.Execute(ctx, map[string]interface{}{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "operation")
}

func TestTool_NameSanitization(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	// Set with spaces and special chars
	_, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "set",
		"name":      "my key!",
		"content":   "content",
	})
	require.NoError(t, err)

	// Get with sanitized name
	result, err := tool.Execute(ctx, map[string]interface{}{
		"operation": "get",
		"name":      "my_key_",
	})
	require.NoError(t, err)

	var response map[string]interface{}
	err = json.Unmarshal([]byte(result), &response)
	require.NoError(t, err)

	assert.Equal(t, "my_key_", response["name"])
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
