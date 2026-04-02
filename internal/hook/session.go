package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/2bit-software/zombiekit/internal/rules"
)

func stateFilePath(sessionID string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("zk-session-%s.json", sessionID))
}

// LoadState reads the session state from disk. Returns a fresh state if
// the file is missing or corrupt.
func LoadState(sessionID string, agent Agent) *rules.SessionState {
	path := stateFilePath(sessionID)

	data, err := os.ReadFile(path)
	if err != nil {
		return newState(sessionID, agent)
	}

	var state rules.SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return newState(sessionID, agent)
	}

	return &state
}

// SaveState writes the session state to disk atomically (temp file + rename).
func SaveState(state *rules.SessionState) error {
	path := stateFilePath(state.SessionID)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}

	return os.Rename(tmpPath, path)
}

// DeleteState removes the session state file.
func DeleteState(sessionID string) error {
	path := stateFilePath(sessionID)
	err := os.Remove(path)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// IsRuleInjected reports whether a rule has already been injected this session.
func IsRuleInjected(state *rules.SessionState, ruleID string) bool {
	_, ok := state.InjectedRules[ruleID]
	return ok
}

// MarkRuleInjected records that a rule has been injected.
func MarkRuleInjected(state *rules.SessionState, ruleID string) {
	if state.InjectedRules == nil {
		state.InjectedRules = make(map[string]time.Time)
	}
	state.InjectedRules[ruleID] = time.Now()
}

// ResetInjectedRules clears the injected rules set and increments the
// compaction counter.
func ResetInjectedRules(state *rules.SessionState) {
	state.InjectedRules = make(map[string]time.Time)
	state.CompactionCount++
}

func newState(sessionID string, agent Agent) *rules.SessionState {
	return &rules.SessionState{
		SessionID:     sessionID,
		Agent:         string(agent),
		StartedAt:     time.Now(),
		InjectedRules: make(map[string]time.Time),
	}
}
