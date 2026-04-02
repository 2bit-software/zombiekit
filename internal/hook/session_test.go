package hook

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/2bit-software/zombiekit/internal/rules"
)

func TestSessionState_Lifecycle(t *testing.T) {
	sessionID := "test-session-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	// Fresh state
	state := LoadState(sessionID, AgentClaude)
	assert.Equal(t, sessionID, state.SessionID)
	assert.Equal(t, "claude", state.Agent)
	assert.Empty(t, state.InjectedRules)

	// Mark rule injected
	MarkRuleInjected(state, "project:go.md")
	assert.True(t, IsRuleInjected(state, "project:go.md"))
	assert.False(t, IsRuleInjected(state, "project:py.md"))

	// Save and reload
	require.NoError(t, SaveState(state))
	loaded := LoadState(sessionID, AgentClaude)
	assert.True(t, IsRuleInjected(loaded, "project:go.md"))

	// Reset
	ResetInjectedRules(loaded)
	assert.Empty(t, loaded.InjectedRules)
	assert.Equal(t, 1, loaded.CompactionCount)

	// Delete
	require.NoError(t, SaveState(loaded))
	require.NoError(t, DeleteState(sessionID))

	// After delete, should get fresh state
	fresh := LoadState(sessionID, AgentClaude)
	assert.Empty(t, fresh.InjectedRules)
	assert.Equal(t, 0, fresh.CompactionCount)
}

func TestSessionState_CorruptJSON(t *testing.T) {
	sessionID := "test-corrupt-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	// Write corrupt JSON to state file
	path := filepath.Join(os.TempDir(), "zk-session-"+sessionID+".json")
	require.NoError(t, os.WriteFile(path, []byte("{invalid json"), 0o600))

	// Should return fresh state
	state := LoadState(sessionID, AgentGemini)
	assert.Equal(t, sessionID, state.SessionID)
	assert.Equal(t, "gemini", state.Agent)
	assert.Empty(t, state.InjectedRules)
}

func TestDeleteState_NonExistent(t *testing.T) {
	err := DeleteState("nonexistent-session-id")
	assert.NoError(t, err)
}

func TestMarkRuleInjected_NilMap(t *testing.T) {
	state := &rules.SessionState{}
	MarkRuleInjected(state, "project:go.md")
	assert.True(t, IsRuleInjected(state, "project:go.md"))
}
