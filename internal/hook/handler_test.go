package hook

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// handleText is a test helper that returns the injected rule bodies joined
// with blank lines, since most tests assert on substrings of rule content.
// Handler output formatting is no longer the handler's responsibility.
func handleText(t *testing.T, h *Handler, e *HookEvent) (string, error) {
	t.Helper()
	r, err := h.Handle(e)
	return strings.Join(r.Bodies, "\n\n"), err
}

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

	handler := NewHandler(dir, t.TempDir(), AgentClaude)
	output, err := handleText(t, handler, &HookEvent{
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

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	// First read — should inject Go rules
	output, err := handleText(t, handler, &HookEvent{
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

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	// First read
	output1, err := handleText(t, handler, &HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, output1)

	// Second read — should be empty (dedup)
	output2, err := handleText(t, handler, &HookEvent{
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

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	// Read Go file
	output1, err := handleText(t, handler, &HookEvent{
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
	output2, err := handleText(t, handler, &HookEvent{
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

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	// Inject Go rules
	_, err := handleText(t, handler, &HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})
	require.NoError(t, err)

	// Compaction
	output, err := handleText(t, handler, &HookEvent{
		SessionID:     sessionID,
		HookEventName: "SessionStart",
		CWD:           dir,
		Source:        "compact",
	})
	require.NoError(t, err)
	assert.Contains(t, output, "General Rules") // unconditional re-injected

	// Read Go file again — should re-inject after compaction
	output2, err := handleText(t, handler, &HookEvent{
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

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	output, err := handleText(t, handler, &HookEvent{
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

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	// Create state
	_, err := handleText(t, handler, &HookEvent{
		SessionID:     sessionID,
		HookEventName: "SessionStart",
		CWD:           dir,
		Source:        "startup",
	})
	require.NoError(t, err)

	// End session
	output, err := handleText(t, handler, &HookEvent{
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

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	output, err := handleText(t, handler, &HookEvent{
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

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	output, err := handleText(t, handler, &HookEvent{
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

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	// Inject Go rules
	_, err := handleText(t, handler, &HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})
	require.NoError(t, err)

	// Resume
	output, err := handleText(t, handler, &HookEvent{
		SessionID:     sessionID,
		HookEventName: "SessionStart",
		CWD:           dir,
		Source:        "resume",
	})
	require.NoError(t, err)
	assert.Contains(t, output, "General Rules")

	// Read Go file again — should re-inject after resume
	output2, err := handleText(t, handler, &HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Read",
		ToolInput:     &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})
	require.NoError(t, err)
	assert.Contains(t, output2, "Go Standards")
}

func TestHandler_UnrecognizedEvent_Errors(t *testing.T) {
	dir := setupTestRules(t)
	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	_, err := handler.Handle(&HookEvent{
		SessionID:     "test-unrec-" + t.Name(),
		HookEventName: "BogusEvent",
		CWD:           dir,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognized event")
	assert.Contains(t, err.Error(), "BogusEvent")
}

// setupBashRulesRepo wires a temp repo with a Taskfile-aware command rule
// and a symmetrical absent-Taskfile rule, returning the dir for event cwd.
func setupBashRulesRepo(t *testing.T, withTaskfile bool) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git"), 0o755))

	rulesDir := filepath.Join(dir, ".brains", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "tf-present.md"), []byte(`---
commands:
  - "go test"
  - "go run"
requires_files:
  - Taskfile.yml
---
# Use the Taskfile

Run `+"`task dev -- test`"+` instead.
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(rulesDir, "tf-absent.md"), []byte(`---
commands:
  - "go test"
requires_files_absent:
  - Taskfile.yml
---
# No Taskfile

Consider adding one.
`), 0o644))

	if withTaskfile {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "Taskfile.yml"), []byte("version: 3\n"), 0o644))
	}

	return dir
}

func TestHandler_PreToolUse_Bash_FiresOnMatch(t *testing.T) {
	dir := setupBashRulesRepo(t, true)
	sessionID := "test-bash-fires-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentClaude)
	result, err := handler.Handle(&HookEvent{
		SessionID:     sessionID,
		HookEventName: "PreToolUse",
		CWD:           dir,
		ToolName:      "Bash",
		ToolInput:     &ToolInput{Command: "go test ./..."},
	})
	require.NoError(t, err)
	assert.Contains(t, strings.Join(result.Bodies, "\n\n"), "task dev -- test")
	require.Len(t, result.MatchedRules, 1)
	assert.Equal(t, "go test", result.MatchedRules[0].Trigger)
}

func TestHandler_PreToolUse_Bash_DedupPerTrigger(t *testing.T) {
	dir := setupBashRulesRepo(t, true)
	sessionID := "test-bash-dedup-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	r1, err := handler.Handle(&HookEvent{
		SessionID: sessionID, HookEventName: "PreToolUse", CWD: dir,
		ToolName: "Bash", ToolInput: &ToolInput{Command: "go test ./..."},
	})
	require.NoError(t, err)
	assert.NotEmpty(t, r1.Bodies)

	r2, err := handler.Handle(&HookEvent{
		SessionID: sessionID, HookEventName: "PreToolUse", CWD: dir,
		ToolName: "Bash", ToolInput: &ToolInput{Command: "go test -count=1 ./..."},
	})
	require.NoError(t, err)
	assert.Empty(t, r2.Bodies)
	require.Len(t, r2.SkippedRules, 1)
	assert.Equal(t, "go test", r2.SkippedRules[0].Trigger)

	r3, err := handler.Handle(&HookEvent{
		SessionID: sessionID, HookEventName: "PreToolUse", CWD: dir,
		ToolName: "Bash", ToolInput: &ToolInput{Command: "go run main.go"},
	})
	require.NoError(t, err)
	assert.Contains(t, strings.Join(r3.Bodies, "\n\n"), "task dev -- test")
	require.Len(t, r3.MatchedRules, 1)
	assert.Equal(t, "go run", r3.MatchedRules[0].Trigger)
}

func TestHandler_PreToolUse_Bash_NoMatch(t *testing.T) {
	dir := setupBashRulesRepo(t, true)
	sessionID := "test-bash-nomatch-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentClaude)
	result, err := handler.Handle(&HookEvent{
		SessionID: sessionID, HookEventName: "PreToolUse", CWD: dir,
		ToolName: "Bash", ToolInput: &ToolInput{Command: "ls -la"},
	})
	require.NoError(t, err)
	assert.Empty(t, result.Bodies)
	assert.Empty(t, result.MatchedRules)
}

func TestHandler_PreToolUse_Bash_TaskfileGateAbsent(t *testing.T) {
	dir := setupBashRulesRepo(t, false)
	sessionID := "test-bash-gate-off-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentClaude)
	result, err := handler.Handle(&HookEvent{
		SessionID: sessionID, HookEventName: "PreToolUse", CWD: dir,
		ToolName: "Bash", ToolInput: &ToolInput{Command: "go test ./..."},
	})
	require.NoError(t, err)
	joined := strings.Join(result.Bodies, "\n\n")
	assert.Contains(t, joined, "No Taskfile")
	assert.NotContains(t, joined, "task dev -- test")
}

func TestHandler_PreToolUse_Bash_EnvPrefixStripped(t *testing.T) {
	dir := setupBashRulesRepo(t, true)
	sessionID := "test-bash-env-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentClaude)
	result, err := handler.Handle(&HookEvent{
		SessionID: sessionID, HookEventName: "PreToolUse", CWD: dir,
		ToolName: "Bash", ToolInput: &ToolInput{Command: "CGO_ENABLED=0 go test ./..."},
	})
	require.NoError(t, err)
	assert.Contains(t, strings.Join(result.Bodies, "\n\n"), "task dev -- test")
}

func TestHandler_PreToolUse_Bash_ChainedCommand(t *testing.T) {
	dir := setupBashRulesRepo(t, true)
	sessionID := "test-bash-chain-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentClaude)
	result, err := handler.Handle(&HookEvent{
		SessionID: sessionID, HookEventName: "PreToolUse", CWD: dir,
		ToolName: "Bash", ToolInput: &ToolInput{Command: "cd pkg && go test ./..."},
	})
	require.NoError(t, err)
	assert.Contains(t, strings.Join(result.Bodies, "\n\n"), "task dev -- test")
}

func TestHandler_PreToolUse_Bash_EmptyCommand(t *testing.T) {
	dir := setupBashRulesRepo(t, true)
	sessionID := "test-bash-empty-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentClaude)
	result, err := handler.Handle(&HookEvent{
		SessionID: sessionID, HookEventName: "PreToolUse", CWD: dir,
		ToolName: "Bash", ToolInput: &ToolInput{Command: ""},
	})
	require.NoError(t, err)
	assert.Empty(t, result.Bodies)
}

// --- OpenCode session-inject / compact regression tests ---

func TestHandler_SessionInject_InjectsUnconditionalOnFirstCall(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-inject-first-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentOpenCode)
	output, err := handleText(t, handler, &HookEvent{
		SessionID:     sessionID,
		HookEventName: "SessionStart",
		CWD:           dir,
		Source:        "inject",
	})
	require.NoError(t, err)
	assert.Contains(t, output, "# General Rules")
}

func TestHandler_SessionInject_AlwaysInjects(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-inject-always-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	// OpenCode fires experimental.chat.system.transform on every LLM
	// stream, and each stream's output.system is a fresh array. If
	// brains returned empty on the second call, the rules would drop
	// out of the system prompt after the first turn.
	handler := NewHandler(dir, t.TempDir(), AgentOpenCode)

	for i := range 3 {
		output, err := handleText(t, handler, &HookEvent{
			SessionID: sessionID, HookEventName: "SessionStart", CWD: dir, Source: "inject",
		})
		require.NoError(t, err)
		assert.Contains(t, output, "# General Rules", "call %d must emit rules", i+1)
	}
}

func TestHandler_SessionInject_DoesNotWriteSessionState(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-inject-stateless-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentOpenCode)

	_, err := handleText(t, handler, &HookEvent{
		SessionID: sessionID, HookEventName: "SessionStart", CWD: dir, Source: "inject",
	})
	require.NoError(t, err)

	// A stateless inject path should not create the /tmp state file.
	_, statErr := os.Stat(stateFilePath(sessionID))
	assert.True(t, os.IsNotExist(statErr), "session-inject must not touch session state file")
}

func TestHandler_SessionInject_DoesNotClobberExistingDedup(t *testing.T) {
	dir := setupTestRules(t)
	sessionID := "test-inject-preserve-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	handler := NewHandler(dir, t.TempDir(), AgentClaude)

	// Claude reads a .go file — Go Standards is marked injected.
	first, err := handleText(t, handler, &HookEvent{
		SessionID: sessionID, HookEventName: "PreToolUse", CWD: dir,
		ToolName: "Read", ToolInput: &ToolInput{FilePath: filepath.Join(dir, "main.go")},
	})
	require.NoError(t, err)
	assert.Contains(t, first, "Go Standards")

	// A session-inject call after that must NOT reset dedup — so a
	// subsequent Read of another .go file is still suppressed.
	_, err = handleText(t, handler, &HookEvent{
		SessionID: sessionID, HookEventName: "SessionStart", CWD: dir, Source: "inject",
	})
	require.NoError(t, err)

	second, err := handleText(t, handler, &HookEvent{
		SessionID: sessionID, HookEventName: "PreToolUse", CWD: dir,
		ToolName: "Read", ToolInput: &ToolInput{FilePath: filepath.Join(dir, "other.go")},
	})
	require.NoError(t, err)
	assert.Empty(t, second, "session-inject must not reset existing file-glob dedup")
}
