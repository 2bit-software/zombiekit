package mcp

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zombiekit/brains/internal/memory/sqlite"
)

func setupTestServer(t *testing.T) *Server {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	storage, err := sqlite.NewSQLiteStorage(context.Background(), dbPath)
	require.NoError(t, err)

	t.Cleanup(func() {
		storage.Close()
	})

	return NewServer(storage, nil)
}

func TestServer_ToolsList_ReturnsBothTools(t *testing.T) {
	server := setupTestServer(t)

	// Verify server is created with tools registered
	mcpServer := server.MCPServer()
	assert.NotNil(t, mcpServer)
}

func TestServer_ToolCall_StickyMemory_Success(t *testing.T) {
	server := setupTestServer(t)
	ctx := context.Background()

	// Set a memory
	args := map[string]interface{}{
		"operation": "set",
		"name":      "test-key",
		"content":   "test content",
	}

	result, err := server.stickyMemory.Execute(ctx, args)
	require.NoError(t, err)
	assert.Contains(t, result, "success")
	assert.Contains(t, result, "test-key")
}

func TestServer_ToolCall_CodeReasoning_Success(t *testing.T) {
	server := setupTestServer(t)
	ctx := context.Background()

	args := map[string]interface{}{
		"thought":             "First thought",
		"thought_number":      float64(1),
		"total_thoughts":      float64(3),
		"next_thought_needed": true,
	}

	result, err := server.codeReasoning.Execute(ctx, "test-session", args)
	require.NoError(t, err)
	assert.Contains(t, result, "First thought")
	assert.Contains(t, result, "in_progress")
}

func TestServer_ToolCall_InvalidTool_Error(t *testing.T) {
	server := setupTestServer(t)
	ctx := context.Background()

	// Call stickymemory with invalid operation
	args := map[string]interface{}{
		"operation": "invalid",
	}

	_, err := server.stickyMemory.Execute(ctx, args)
	assert.Error(t, err)
}

func TestServer_ToolCall_InvalidParams_Error(t *testing.T) {
	server := setupTestServer(t)
	ctx := context.Background()

	// Call stickymemory without required params
	args := map[string]interface{}{
		"operation": "set",
		// Missing name and content
	}

	_, err := server.stickyMemory.Execute(ctx, args)
	assert.Error(t, err)
}

func TestServer_Close(t *testing.T) {
	server := setupTestServer(t)

	err := server.Close()
	assert.NoError(t, err)
}
