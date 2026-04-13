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

// A session file written before the per-trigger dedup key format must
// still load correctly, with bare rule IDs rewritten to empty-trigger keys.
func TestLoadState_MigratesLegacyKeys(t *testing.T) {
	sessionID := "test-legacy-" + t.Name()
	defer func() { _ = DeleteState(sessionID) }()

	path := filepath.Join(os.TempDir(), "zk-session-"+sessionID+".json")
	legacy := `{
  "session_id": "` + sessionID + `",
  "agent": "claude",
  "started_at": "2024-01-01T00:00:00Z",
  "compaction_count": 0,
  "injected_rules": {
    "project:go.md": "2024-01-01T00:00:01Z",
    "global:general.md": "2024-01-01T00:00:02Z"
  }
}`
	require.NoError(t, os.WriteFile(path, []byte(legacy), 0o600))

	state := LoadState(sessionID, AgentClaude)
	assert.True(t, IsRuleInjectedFor(state, "project:go.md", ""))
	assert.True(t, IsRuleInjected(state, "project:go.md"))
	assert.True(t, IsRuleInjected(state, "global:general.md"))
	_, bareStillThere := state.InjectedRules["project:go.md"]
	assert.False(t, bareStillThere, "bare-key entry should have been migrated away")
}

// Per-trigger dedup: two triggers mapped to the same rule inject
// independently, while the same trigger seen twice is suppressed.
func TestMarkRuleInjectedFor_PerTriggerDedup(t *testing.T) {
	state := &rules.SessionState{}
	MarkRuleInjectedFor(state, "project:tf.md", "go test")
	assert.True(t, IsRuleInjectedFor(state, "project:tf.md", "go test"))
	assert.False(t, IsRuleInjectedFor(state, "project:tf.md", "go run"))

	MarkRuleInjectedFor(state, "project:tf.md", "go run")
	assert.True(t, IsRuleInjectedFor(state, "project:tf.md", "go run"))
}
