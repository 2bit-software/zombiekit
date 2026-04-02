package hook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRules(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize git repo so resolver stops walking
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

	rulesDir := filepath.Join(dir, ".brains", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "go.md"), []byte(`---
paths:
  - "**/*.go"
---
# Go Standards

- Use any`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "ts.md"), []byte(`---
paths:
  - "**/*.{ts,tsx}"
---
# TypeScript Standards

- Use strict mode`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "general.md"), []byte(`# General Rules

- Always check errors`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "empty.md"), []byte(`---
paths:
  - "**/*.empty"
---
`), 0o644))

	return dir
}

func TestHandler_SessionStart_InjectsUnconditional(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-ss-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentGemini)
	output, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "SessionStart",
		CWD:           dir,
		Source:        "startup",
	})

	require.NoError(t, err)
	assert.Contains(t, output, "# General Rules")
	assert.NotContains(t, output, "Go Standards")
	assert.NotContains(t, output, "TypeScript")
}

func TestHandler_PreToolUse_Read_InjectsMatchingRules(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-ptr-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentGemini)

	// First read — should inject Go rules
	output, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})

	require.NoError(t, err)
	assert.Contains(t, output, "Go Standards")
}

func TestHandler_PreToolUse_Deduplication(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-dedup-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentGemini)

	// First read
	output1, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, output1)

	// Second read — should be empty (dedup)
	output2, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "other.go")},
	})
	require.NoError(t, err)
	assert.Empty(t, output2)
}

func TestHandler_PreToolUse_DifferentTypes(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-diff-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentGemini)

	// Read Go file
	output1, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})
	require.NoError(t, err)
	assert.Contains(t, output1, "Go Standards")
	assert.NotContains(t, output1, "TypeScript")

	// Read TS file — should inject TS rules only
	output2, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "app.tsx")},
	})
	require.NoError(t, err)
	assert.Contains(t, output2, "TypeScript Standards")
	assert.NotContains(t, output2, "Go Standards")
}

func TestHandler_Compaction_ResetsTracking(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-compact-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentGemini)

	// Inject Go rules
	_, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})
	require.NoError(t, err)

	// Compaction
	output, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "SessionStart",
		CWD:           dir,
		Source:        "compact",
	})
	require.NoError(t, err)
	assert.Contains(t, output, "General Rules") // unconditional re-injected

	// Read Go file again — should re-inject after compaction
	output2, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})
	require.NoError(t, err)
	assert.Contains(t, output2, "Go Standards")
}

func TestHandler_MultiEdit(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-multi-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentGemini)

	output, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "MultiEdit",
		ToolInput: &ToolInput{
			Edits: []EditEntry{
				{FilePath: filepath.Join(dir, "main.go")},
				{FilePath: filepath.Join(dir, "app.tsx")},
			},
		},
	})
	require.NoError(t, err)
	assert.Contains(t, output, "Go Standards")
	assert.Contains(t, output, "TypeScript Standards")
}

func TestHandler_SessionEnd_DeletesState(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-end-" + t.Name()

	handler := NewHandler(dir, t.TempDir(), AgentGemini)

	// Create state
	_, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "SessionStart",
		CWD:           dir,
		Source:        "startup",
	})
	require.NoError(t, err)

	// End session
	output, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "SessionEnd",
		CWD:           dir,
	})
	require.NoError(t, err)
	assert.Empty(t, output)

	// State file should be gone
	statePath := filepath.Join(os.TempDir(), "zk-session-"+sessionID+".json")
	_, err = os.Stat(statePath)
	assert.True(t, os.IsNotExist(err))
}

func TestHandler_EmptyBodyRules_Skipped(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-empty-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentGemini)

	output, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "test.empty")},
	})
	require.NoError(t, err)
	assert.Empty(t, output)
}

func TestHandler_Write_InjectsIfNotSeen(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-write-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentGemini)

	output, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Write",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "new.go")},
	})
	require.NoError(t, err)
	assert.Contains(t, output, "Go Standards")
}

func TestHandler_Resume_ResetsTracking(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-resume-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentGemini)

	// Inject Go rules
	_, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})
	require.NoError(t, err)

	// Resume
	output, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "SessionStart",
		CWD:           dir,
		Source:        "resume",
	})
	require.NoError(t, err)
	assert.Contains(t, output, "General Rules")

	// Read Go file again — should re-inject after resume
	output2, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})
	require.NoError(t, err)
	assert.Contains(t, output2, "Go Standards")
}

func TestHandler_ClaudeFormat_SessionStart(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-claude-ss-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	output, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "SessionStart",
		CWD:           dir,
		Source:        "startup",
	})
	require.NoError(t, err)
	assert.Contains(t, output, "<system-reminder>")
	assert.Contains(t, output, "</system-reminder>")
	assert.Contains(t, output, "General Rules")
}

func TestHandler_ClaudeFormat_PreToolUse(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-claude-ptu-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	output, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})
	require.NoError(t, err)
	assert.Contains(t, output, `"hookSpecificOutput"`)
	assert.Contains(t, output, `"permissionDecision":"allow"`)
	assert.Contains(t, output, `"hookEventName":"PreToolUse"`)
	assert.Contains(t, output, "Go Standards")
}
