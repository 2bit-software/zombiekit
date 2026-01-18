package claude_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zombiekit/brains/internal/recall"
	"github.com/zombiekit/brains/internal/recall/claude"
)

// ============================================================
// T019: Integration tests for watch mode
// BR-005 (Auto-import)
// ============================================================

// setupWatchTestHarness creates a test harness similar to import_test.go
func setupWatchTestHarness(t *testing.T) *testHarness {
	return setupTestHarness(t)
}

func TestWatch_DetectsNewFile(t *testing.T) {
	h := setupWatchTestHarness(t)
	ctx := context.Background()

	// First scan - no files
	files1, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)
	assert.Len(t, files1, 0, "initial scan should find no files")

	// Add a new file
	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"New message"},"isMeta":false,"isSidechain":false}`,
	}
	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	// Second scan - finds new file
	files2, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)
	assert.Len(t, files2, 1, "second scan should find new file")

	// Import the new file
	newCount, _, err := h.importFile(ctx, files2[0])
	require.NoError(t, err)
	assert.Equal(t, 1, newCount, "should import the new message")
}

func TestWatch_DetectsNewMessages(t *testing.T) {
	h := setupWatchTestHarness(t)
	ctx := context.Background()

	// Create initial file
	entries1 := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"First message"},"isMeta":false,"isSidechain":false}`,
	}
	filePath := createHistoryFile(t, h.tmpDir, "/Users/test/project", entries1)

	// First import
	newCount1, _, err := h.importFile(ctx, filePath)
	require.NoError(t, err)
	assert.Equal(t, 1, newCount1)

	// Add new message to file (simulating ongoing conversation)
	entries2 := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"First message"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-002","sessionId":"session-1","timestamp":"2024-01-15T10:00:05Z","message":{"role":"assistant","content":"New response"},"isMeta":false,"isSidechain":false}`,
	}
	content := ""
	for _, e := range entries2 {
		content += e + "\n"
	}
	err = os.WriteFile(filePath, []byte(content), 0644)
	require.NoError(t, err)

	// Second import should detect new message
	newCount2, skipCount2, err := h.importFile(ctx, filePath)
	require.NoError(t, err)
	assert.Equal(t, 1, newCount2, "should detect and import new message")
	assert.Equal(t, 1, skipCount2, "should skip existing message")
}

func TestWatch_NewProject(t *testing.T) {
	h := setupWatchTestHarness(t)
	ctx := context.Background()

	// Create file in first project
	entries1 := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Project 1 message"},"isMeta":false,"isSidechain":false}`,
	}
	createHistoryFile(t, h.tmpDir, "/Users/test/project1", entries1)

	// First import
	files1, _ := claude.DiscoverHistoryFiles(h.tmpDir)
	for _, f := range files1 {
		_, _, err := h.importFile(ctx, f)
		require.NoError(t, err)
	}

	// Create file in second project (simulating user opening new project)
	entries2 := []string{
		`{"type":"user","uuid":"msg-002","sessionId":"session-2","timestamp":"2024-01-15T11:00:00Z","message":{"role":"user","content":"Project 2 message"},"isMeta":false,"isSidechain":false}`,
	}
	createHistoryFile(t, h.tmpDir, "/Users/test/project2", entries2)

	// Second scan should find new project's file
	files2, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)
	assert.Len(t, files2, 2, "should find both project files")

	// Import should pick up new file
	totalNew := 0
	for _, f := range files2 {
		newCount, _, err := h.importFile(ctx, f)
		require.NoError(t, err)
		totalNew += newCount
	}
	assert.Equal(t, 1, totalNew, "should import only the new message from project2")
}

func TestWatch_IntervalSimulation(t *testing.T) {
	h := setupWatchTestHarness(t)
	ctx := context.Background()

	// Simulate multiple watch intervals
	// Interval 1: Create initial file
	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"session-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Message 1"},"isMeta":false,"isSidechain":false}`,
	}
	filePath := createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, _ := claude.DiscoverHistoryFiles(h.tmpDir)
	total1, _, _ := h.importFile(ctx, files[0])

	// Interval 2: No changes
	total2, skip2, _ := h.importFile(ctx, files[0])
	assert.Equal(t, 0, total2, "no new messages in interval 2")
	assert.Equal(t, 1, skip2, "should skip existing")

	// Interval 3: New message added
	entries = append(entries, `{"type":"assistant","uuid":"msg-002","sessionId":"session-1","timestamp":"2024-01-15T10:00:05Z","message":{"role":"assistant","content":"Response"},"isMeta":false,"isSidechain":false}`)
	content := ""
	for _, e := range entries {
		content += e + "\n"
	}
	os.WriteFile(filePath, []byte(content), 0644)

	total3, skip3, _ := h.importFile(ctx, filePath)
	assert.Equal(t, 1, total3, "should import new message in interval 3")
	assert.Equal(t, 1, skip3, "should skip existing in interval 3")

	// Total across all intervals
	assert.Equal(t, 2, total1+total2+total3, "total imported across intervals")
}

// ============================================================
// T020: Integration tests for conversation retrieval
// BR-009 (Retrieve messages), BR-010 (Navigate to conversation)
// ============================================================

func TestConversation_AllMessagesReturned(t *testing.T) {
	h := setupWatchTestHarness(t)
	ctx := context.Background()

	// Create conversation with multiple messages
	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"conv-test-1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Hello"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-002","sessionId":"conv-test-1","timestamp":"2024-01-15T10:00:05Z","message":{"role":"assistant","content":"Hi there!"},"isMeta":false,"isSidechain":false}`,
		`{"type":"user","uuid":"msg-003","sessionId":"conv-test-1","timestamp":"2024-01-15T10:00:10Z","message":{"role":"user","content":"How are you?"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-004","sessionId":"conv-test-1","timestamp":"2024-01-15T10:00:15Z","message":{"role":"assistant","content":"I'm doing well!"},"isMeta":false,"isSidechain":false}`,
	}
	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, _ := claude.DiscoverHistoryFiles(h.tmpDir)
	h.importFile(ctx, files[0])

	// Retrieve conversation
	chunks, err := h.storage.GetByConversation(ctx, "conv-test-1")
	require.NoError(t, err)
	assert.Len(t, chunks, 4, "should return all 4 messages")
}

func TestConversation_ChronologicalOrder(t *testing.T) {
	h := setupWatchTestHarness(t)
	ctx := context.Background()

	// Create messages in non-chronological order in file
	entries := []string{
		`{"type":"assistant","uuid":"msg-003","sessionId":"conv-order","timestamp":"2024-01-15T10:00:15Z","message":{"role":"assistant","content":"Third"},"isMeta":false,"isSidechain":false}`,
		`{"type":"user","uuid":"msg-001","sessionId":"conv-order","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"First"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-002","sessionId":"conv-order","timestamp":"2024-01-15T10:00:05Z","message":{"role":"assistant","content":"Second"},"isMeta":false,"isSidechain":false}`,
	}
	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, _ := claude.DiscoverHistoryFiles(h.tmpDir)
	h.importFile(ctx, files[0])

	// Should be ordered by timestamp
	chunks, err := h.storage.GetByConversation(ctx, "conv-order")
	require.NoError(t, err)
	require.Len(t, chunks, 3)

	assert.Equal(t, "First", chunks[0].Content)
	assert.Equal(t, "Second", chunks[1].Content)
	assert.Equal(t, "Third", chunks[2].Content)
}

func TestConversation_IncludesUserAndAssistant(t *testing.T) {
	h := setupWatchTestHarness(t)
	ctx := context.Background()

	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"conv-roles","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"User message"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-002","sessionId":"conv-roles","timestamp":"2024-01-15T10:00:05Z","message":{"role":"assistant","content":"Assistant message"},"isMeta":false,"isSidechain":false}`,
	}
	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, _ := claude.DiscoverHistoryFiles(h.tmpDir)
	h.importFile(ctx, files[0])

	chunks, err := h.storage.GetByConversation(ctx, "conv-roles")
	require.NoError(t, err)
	require.Len(t, chunks, 2)

	// Verify both roles are included
	roles := make(map[string]bool)
	for _, c := range chunks {
		if c.Metadata != nil {
			roles[c.Metadata.Role] = true
		}
	}
	assert.True(t, roles["user"], "should include user messages")
	assert.True(t, roles["assistant"], "should include assistant messages")
}

func TestSearch_ReturnsConversationID(t *testing.T) {
	h := setupWatchTestHarness(t)
	ctx := context.Background()

	// Create a message with specific content
	entries := []string{
		`{"type":"user","uuid":"msg-search-1","sessionId":"searchable-conv-123","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Tell me about quantum computing"},"isMeta":false,"isSidechain":false}`,
	}
	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, _ := claude.DiscoverHistoryFiles(h.tmpDir)
	h.importFile(ctx, files[0])

	// Search for content
	queryEmbed, _ := h.embedder.Embed(ctx, "quantum computing", recall.PurposeQuery)
	results, err := h.storage.Search(ctx, queryEmbed, 5)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Result should include conversation_id for navigation
	found := false
	for _, r := range results {
		if r.Chunk.ConversationID == "searchable-conv-123" {
			found = true
			break
		}
	}
	assert.True(t, found, "search result should include conversation_id")
}

func TestConversation_FromSearchResult(t *testing.T) {
	h := setupWatchTestHarness(t)
	ctx := context.Background()

	// Create multi-message conversation
	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"nav-conv-456","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"First question about databases"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-002","sessionId":"nav-conv-456","timestamp":"2024-01-15T10:00:05Z","message":{"role":"assistant","content":"Here's info about databases"},"isMeta":false,"isSidechain":false}`,
		`{"type":"user","uuid":"msg-003","sessionId":"nav-conv-456","timestamp":"2024-01-15T10:00:10Z","message":{"role":"user","content":"Follow up question"},"isMeta":false,"isSidechain":false}`,
	}
	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, _ := claude.DiscoverHistoryFiles(h.tmpDir)
	h.importFile(ctx, files[0])

	// Search and find one message
	queryEmbed, _ := h.embedder.Embed(ctx, "databases", recall.PurposeQuery)
	results, err := h.storage.Search(ctx, queryEmbed, 5)
	require.NoError(t, err)

	// Find the conversation ID from search result
	var convID string
	for _, r := range results {
		if r.Chunk.ConversationID != "" {
			convID = r.Chunk.ConversationID
			break
		}
	}
	require.NotEmpty(t, convID, "should find conversation ID from search")

	// Navigate to full conversation
	fullConv, err := h.storage.GetByConversation(ctx, convID)
	require.NoError(t, err)
	assert.Len(t, fullConv, 3, "should retrieve entire conversation from search result")
}

func TestConversation_Empty(t *testing.T) {
	h := setupWatchTestHarness(t)
	ctx := context.Background()

	// Query non-existent conversation
	chunks, err := h.storage.GetByConversation(ctx, "non-existent-conv")
	require.NoError(t, err)
	assert.Empty(t, chunks, "should return empty for unknown conversation")
}

func TestConversation_MultipleConversations(t *testing.T) {
	h := setupWatchTestHarness(t)
	ctx := context.Background()

	// Create two separate conversations
	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"conv-A","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Conversation A message 1"},"isMeta":false,"isSidechain":false}`,
		`{"type":"user","uuid":"msg-002","sessionId":"conv-B","timestamp":"2024-01-15T10:00:05Z","message":{"role":"user","content":"Conversation B message 1"},"isMeta":false,"isSidechain":false}`,
		`{"type":"user","uuid":"msg-003","sessionId":"conv-A","timestamp":"2024-01-15T10:00:10Z","message":{"role":"user","content":"Conversation A message 2"},"isMeta":false,"isSidechain":false}`,
	}
	createHistoryFile(t, h.tmpDir, "/Users/test/project", entries)

	files, _ := claude.DiscoverHistoryFiles(h.tmpDir)
	h.importFile(ctx, files[0])

	// Retrieve each conversation separately
	convA, err := h.storage.GetByConversation(ctx, "conv-A")
	require.NoError(t, err)
	assert.Len(t, convA, 2, "conv-A should have 2 messages")

	convB, err := h.storage.GetByConversation(ctx, "conv-B")
	require.NoError(t, err)
	assert.Len(t, convB, 1, "conv-B should have 1 message")
}
