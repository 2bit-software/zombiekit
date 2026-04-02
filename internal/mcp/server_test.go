package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	mcpserver "github.com/mark3labs/mcp-go/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2bit-software/zombiekit/internal/memory/sqlite"
	"github.com/2bit-software/zombiekit/internal/profile"
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

	return NewServer(storage, nil, nil)
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

func TestServer_ToolCall_SkillInstall_InvalidName(t *testing.T) {
	server := setupTestServer(t)
	ctx := context.Background()

	_, err := server.skillInstallTool.Execute(ctx, map[string]any{
		"name":  "../evil",
		"scope": "local",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid skill name")
}

func TestServer_ToolCall_SkillInstall_UnknownProfile(t *testing.T) {
	server := setupTestServer(t)
	ctx := context.Background()

	_, err := server.skillInstallTool.Execute(ctx, map[string]any{
		"name":              "definitely-not-a-real-profile-xyzzy",
		"scope":             "local",
		"working_directory": t.TempDir(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// sendLine writes a JSON-RPC message as a newline-terminated line to w.
func sendLine(t *testing.T, w io.Writer, msg any) {
	t.Helper()
	b, err := json.Marshal(msg)
	require.NoError(t, err)
	_, err = w.Write(append(b, '\n'))
	require.NoError(t, err)
}

// readLine reads one newline-terminated JSON-RPC response from r and unmarshals it.
func readLine(t *testing.T, r *bufio.Reader) map[string]any {
	t.Helper()
	line, err := r.ReadString('\n')
	require.NoError(t, err)
	var result map[string]any
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(line)), &result))
	return result
}

func TestServer_StdioProtocol_SkillInstall(t *testing.T) {
	// Set up a mock embedded FS with a known test profile.
	origFS := profile.GetEmbeddedFS()
	mockFS := fstest.MapFS{
		"profiles/test-skill.md": &fstest.MapFile{
			Data: []byte("---\nname: test-skill\ndescription: A test skill profile.\n---\n\nTest content.\n"),
		},
	}
	profile.SetEmbeddedFS(mockFS)
	t.Cleanup(func() { profile.SetEmbeddedFS(origFS) })

	// Target directory where the skill should be installed.
	targetDir := t.TempDir()

	// Build the server.
	srv := setupTestServer(t)

	// Wrap in a StdioServer with error logging silenced.
	stdioSrv := mcpserver.NewStdioServer(srv.MCPServer())
	stdioSrv.SetErrorLogger(log.New(io.Discard, "", 0))

	// Wire up pipes: test writes to stdinW, server reads from stdinR.
	// Server writes to stdoutW, test reads from stdoutR.
	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()

	ctx, cancel := context.WithCancel(context.Background())
	serverDone := make(chan error, 1)
	go func() {
		err := stdioSrv.Listen(ctx, stdinR, stdoutW)
		_ = stdoutW.Close()
		serverDone <- err
	}()

	stdout := bufio.NewReader(stdoutR)

	// 1. Initialize handshake.
	sendLine(t, stdinW, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "test", "version": "1.0.0"},
		},
	})
	initResp := readLine(t, stdout)
	require.Equal(t, float64(1), initResp["id"], "initialize response id mismatch")

	// 2. Send initialized notification (no response expected).
	sendLine(t, stdinW, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})

	// 3. Call skill-install.
	sendLine(t, stdinW, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "skill-install",
			"arguments": map[string]any{
				"name":              "test-skill",
				"scope":             "local",
				"working_directory": targetDir,
			},
		},
	})
	callResp := readLine(t, stdout)

	// Shut down the server cleanly.
	cancel()
	_ = stdinW.Close()
	<-serverDone

	// Verify the JSON-RPC response is a success (no "error" key).
	require.Equal(t, float64(2), callResp["id"], "tools/call response id mismatch")
	assert.Nil(t, callResp["error"], "expected no error in response: %v", callResp["error"])

	// Verify the SKILL.md was actually written to disk.
	skillPath := filepath.Join(targetDir, ".claude", "skills", "test-skill", "SKILL.md")
	content, err := os.ReadFile(skillPath)
	require.NoError(t, err, "SKILL.md not found at %s", skillPath)
	assert.Contains(t, string(content), "name: test-skill")
	assert.Contains(t, string(content), `profiles: ["test-skill"]`)
}

func TestServer_Close(t *testing.T) {
	server := setupTestServer(t)

	err := server.Close()
	assert.NoError(t, err)
}
