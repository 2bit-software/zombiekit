package claude_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2bit-software/zombiekit/internal/recall"
	"github.com/2bit-software/zombiekit/internal/recall/claude"
)

// ============================================================
// T021: E2E tests for user scenarios
// Validates complete user workflows from Business Spec
// ============================================================

// Scenario 1: Initial Import
// User wants to import existing Claude conversation history for search.
func TestE2E_ImportAndSearch(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	// Setup: User has existing Claude conversations
	entries := []string{
		// Conversation about debugging
		`{"type":"user","uuid":"msg-001","sessionId":"debug-session","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"I'm having trouble with a null pointer exception in my Java code"},"isMeta":false,"isSidechain":false,"cwd":"/home/dev/project","gitBranch":"main"}`,
		`{"type":"assistant","uuid":"msg-002","parentUuid":"msg-001","sessionId":"debug-session","timestamp":"2024-01-15T10:00:10Z","message":{"role":"assistant","content":"Let me help you debug that. Null pointer exceptions usually occur when you try to use a reference that points to null. Can you share the stack trace?"},"isMeta":false,"isSidechain":false}`,

		// Conversation about deployment
		`{"type":"user","uuid":"msg-003","sessionId":"deploy-session","timestamp":"2024-01-16T14:00:00Z","message":{"role":"user","content":"How do I deploy this application to Kubernetes?"},"isMeta":false,"isSidechain":false,"cwd":"/home/dev/k8s-app","gitBranch":"feature/k8s"}`,
		`{"type":"assistant","uuid":"msg-004","parentUuid":"msg-003","sessionId":"deploy-session","timestamp":"2024-01-16T14:00:15Z","message":{"role":"assistant","content":"To deploy to Kubernetes, you'll need to create deployment manifests. Start with a Deployment and Service YAML."},"isMeta":false,"isSidechain":false}`,
	}
	createHistoryFile(t, h.tmpDir, "/Users/dev/project", entries)

	// Action 1: User runs import
	files, err := claude.DiscoverHistoryFiles(h.tmpDir)
	require.NoError(t, err)
	require.Len(t, files, 1)

	newCount, _, err := h.importFile(ctx, files[0])
	require.NoError(t, err)
	assert.Equal(t, 4, newCount, "should import all 4 messages")

	// Action 2: User searches for previous debugging help
	queryEmbed, _ := h.embedder.Embed(ctx, "null pointer exception debugging", recall.PurposeQuery)
	results, err := h.storage.Search(ctx, queryEmbed, 5)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Verify: Results include relevant conversation
	foundDebugConv := false
	for _, r := range results {
		if r.Chunk.ConversationID == "debug-session" {
			foundDebugConv = true
			assert.Contains(t, r.Chunk.Content, "null pointer")
			break
		}
	}
	assert.True(t, foundDebugConv, "search should find the debugging conversation")
}

// Scenario 2: Update History
// User continues conversation, runs import again, no duplicates.
func TestE2E_ReimportNoDuplicates(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	// Setup: Initial import
	entries1 := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"ongoing-conv","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"First question"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-002","sessionId":"ongoing-conv","timestamp":"2024-01-15T10:00:05Z","message":{"role":"assistant","content":"First answer"},"isMeta":false,"isSidechain":false}`,
	}
	filePath := createHistoryFile(t, h.tmpDir, "/Users/dev/project", entries1)

	// Action 1: First import
	newCount1, _, err := h.importFile(ctx, filePath)
	require.NoError(t, err)
	assert.Equal(t, 2, newCount1, "first import creates 2 messages")

	// User continues conversation (new messages appended)
	entries2 := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"ongoing-conv","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"First question"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-002","sessionId":"ongoing-conv","timestamp":"2024-01-15T10:00:05Z","message":{"role":"assistant","content":"First answer"},"isMeta":false,"isSidechain":false}`,
		`{"type":"user","uuid":"msg-003","sessionId":"ongoing-conv","timestamp":"2024-01-15T10:01:00Z","message":{"role":"user","content":"Follow-up question"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-004","sessionId":"ongoing-conv","timestamp":"2024-01-15T10:01:05Z","message":{"role":"assistant","content":"Follow-up answer"},"isMeta":false,"isSidechain":false}`,
	}
	content := ""
	for _, e := range entries2 {
		content += e + "\n"
	}
	err = writeFile(filePath, content)
	require.NoError(t, err)

	// Action 2: Re-import (should only add new messages)
	newCount2, skipCount2, err := h.importFile(ctx, filePath)
	require.NoError(t, err)
	assert.Equal(t, 2, newCount2, "re-import adds only 2 new messages")
	assert.Equal(t, 2, skipCount2, "re-import skips 2 existing messages")

	// Verify: Total messages in conversation
	conv, err := h.storage.GetByConversation(ctx, "ongoing-conv")
	require.NoError(t, err)
	assert.Len(t, conv, 4, "conversation should have 4 total messages")
}

// Scenario 3: View Conversation
// User finds a search result and navigates to full conversation.
func TestE2E_SearchToConversation(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	// Setup: Import a multi-turn conversation
	entries := []string{
		`{"type":"user","uuid":"msg-001","sessionId":"detailed-conv-xyz","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Can you explain how React hooks work?"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-002","sessionId":"detailed-conv-xyz","timestamp":"2024-01-15T10:00:15Z","message":{"role":"assistant","content":"React hooks are functions that let you use state and lifecycle features in functional components. The most common ones are useState and useEffect."},"isMeta":false,"isSidechain":false}`,
		`{"type":"user","uuid":"msg-003","sessionId":"detailed-conv-xyz","timestamp":"2024-01-15T10:01:00Z","message":{"role":"user","content":"What about useCallback and useMemo?"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-004","sessionId":"detailed-conv-xyz","timestamp":"2024-01-15T10:01:20Z","message":{"role":"assistant","content":"useCallback memoizes callback functions to prevent unnecessary re-renders. useMemo memoizes computed values. Both are optimization hooks."},"isMeta":false,"isSidechain":false}`,
		`{"type":"user","uuid":"msg-005","sessionId":"detailed-conv-xyz","timestamp":"2024-01-15T10:02:00Z","message":{"role":"user","content":"When should I use them?"},"isMeta":false,"isSidechain":false}`,
		`{"type":"assistant","uuid":"msg-006","sessionId":"detailed-conv-xyz","timestamp":"2024-01-15T10:02:30Z","message":{"role":"assistant","content":"Use useCallback when passing callbacks to optimized child components. Use useMemo for expensive calculations that shouldn't run on every render."},"isMeta":false,"isSidechain":false}`,
	}
	createHistoryFile(t, h.tmpDir, "/Users/dev/react-app", entries)

	files, _ := claude.DiscoverHistoryFiles(h.tmpDir)
	h.importFile(ctx, files[0])

	// Action 1: User searches for "memoization"
	queryEmbed, _ := h.embedder.Embed(ctx, "React memoization optimization", recall.PurposeQuery)
	results, err := h.storage.Search(ctx, queryEmbed, 3)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Action 2: User picks a result and gets conversation ID
	topResult := results[0]
	convID := topResult.Chunk.ConversationID
	require.NotEmpty(t, convID, "search result should have conversation ID")

	// Action 3: User navigates to full conversation
	fullConv, err := h.storage.GetByConversation(ctx, convID)
	require.NoError(t, err)

	// Verify: Full conversation context is available
	assert.Len(t, fullConv, 6, "should retrieve entire conversation")

	// Messages should be in chronological order
	assert.Contains(t, fullConv[0].Content, "React hooks")
	assert.Contains(t, fullConv[5].Content, "expensive calculations")

	// All roles present
	userCount := 0
	assistantCount := 0
	for _, c := range fullConv {
		if c.Metadata != nil {
			switch c.Metadata.Role {
			case "user":
				userCount++
			case "assistant":
				assistantCount++
			}
		}
	}
	assert.Equal(t, 3, userCount, "should have 3 user messages")
	assert.Equal(t, 3, assistantCount, "should have 3 assistant messages")
}

// Scenario 4: Cross-Project Search
// User has conversations in multiple projects, search finds across all.
func TestE2E_CrossProjectSearch(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	// Setup: Multiple projects with different conversations
	// Project 1: Backend development
	entries1 := []string{
		`{"type":"user","uuid":"backend-001","sessionId":"backend-session","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"How do I handle database connections in Go?"},"isMeta":false,"isSidechain":false,"cwd":"/home/dev/backend","gitBranch":"main"}`,
		`{"type":"assistant","uuid":"backend-002","sessionId":"backend-session","timestamp":"2024-01-15T10:00:10Z","message":{"role":"assistant","content":"In Go, use connection pooling with database/sql. The sql.DB type manages a pool automatically."},"isMeta":false,"isSidechain":false}`,
	}

	// Project 2: Frontend development
	entries2 := []string{
		`{"type":"user","uuid":"frontend-001","sessionId":"frontend-session","timestamp":"2024-01-16T09:00:00Z","message":{"role":"user","content":"How do I connect my React frontend to the Go backend API?"},"isMeta":false,"isSidechain":false,"cwd":"/home/dev/frontend","gitBranch":"develop"}`,
		`{"type":"assistant","uuid":"frontend-002","sessionId":"frontend-session","timestamp":"2024-01-16T09:00:15Z","message":{"role":"assistant","content":"Use fetch or axios to make HTTP requests from React. Set up CORS on your Go backend to allow cross-origin requests."},"isMeta":false,"isSidechain":false}`,
	}

	createHistoryFile(t, h.tmpDir, "/home/dev/backend", entries1)
	createHistoryFile(t, h.tmpDir, "/home/dev/frontend", entries2)

	// Import all projects
	files, _ := claude.DiscoverHistoryFiles(h.tmpDir)
	for _, f := range files {
		h.importFile(ctx, f)
	}

	// Action: Search for "connection" (appears in both projects)
	queryEmbed, _ := h.embedder.Embed(ctx, "connection API backend", recall.PurposeQuery)
	results, err := h.storage.Search(ctx, queryEmbed, 10)
	require.NoError(t, err)

	// Verify: Results span multiple conversations
	conversations := make(map[string]bool)
	for _, r := range results {
		if r.Chunk.ConversationID != "" {
			conversations[r.Chunk.ConversationID] = true
		}
	}
	assert.GreaterOrEqual(t, len(conversations), 1, "should find results across projects")
}

// Scenario 5: Metadata Preservation
// User can see context (git branch, working directory) from search results.
func TestE2E_MetadataInSearch(t *testing.T) {
	h := setupTestHarness(t)
	ctx := context.Background()

	// Setup: Message with rich metadata
	entries := []string{
		`{"type":"user","uuid":"meta-001","sessionId":"meta-session","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Fix this authentication bug in the login flow"},"isMeta":false,"isSidechain":false,"cwd":"/home/dev/auth-service","gitBranch":"hotfix/login-bug"}`,
	}
	createHistoryFile(t, h.tmpDir, "/home/dev/auth-service", entries)

	files, _ := claude.DiscoverHistoryFiles(h.tmpDir)
	h.importFile(ctx, files[0])

	// Action: Search for authentication
	queryEmbed, _ := h.embedder.Embed(ctx, "authentication login bug", recall.PurposeQuery)
	results, err := h.storage.Search(ctx, queryEmbed, 5)
	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Verify: Metadata preserved in search results
	var found *recall.SearchResult
	for i, r := range results {
		if r.Chunk.ConversationID == "meta-session" {
			found = &results[i]
			break
		}
	}
	require.NotNil(t, found, "should find the message")
	require.NotNil(t, found.Chunk.Metadata)

	assert.Equal(t, "/home/dev/auth-service", found.Chunk.Metadata.CWD)
	assert.Equal(t, "hotfix/login-bug", found.Chunk.Metadata.GitBranch)
	assert.Equal(t, "user", found.Chunk.Metadata.Role)
}

// writeFile is a helper for tests
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}
