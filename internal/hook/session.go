package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/2bit-software/zombiekit/internal/rules"
)

func stateFilePath(sessionID string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("zk-session-%s.json", sessionID))
}

// LoadState reads the session state from disk. Returns a fresh state if
// the file is missing or corrupt. Legacy state files whose InjectedRules
// keys are bare rule IDs are migrated in-place to the composite
// "ruleID|trigger" form used by the current dedup scheme.
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

	migrateInjectedKeys(&state)
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

// IsRuleInjected reports whether a file-glob rule (empty trigger) has
// already been injected this session.
func IsRuleInjected(state *rules.SessionState, ruleID string) bool {
	return IsRuleInjectedFor(state, ruleID, "")
}

// IsRuleInjectedFor reports whether a rule has already fired for a
// specific trigger this session. The trigger is empty for file-glob rules
// and the matched command prefix for command rules.
func IsRuleInjectedFor(state *rules.SessionState, ruleID, trigger string) bool {
	_, ok := state.InjectedRules[injectionKey(ruleID, trigger)]
	return ok
}

// MarkRuleInjected records that a file-glob rule has been injected.
func MarkRuleInjected(state *rules.SessionState, ruleID string) {
	MarkRuleInjectedFor(state, ruleID, "")
}

// MarkRuleInjectedFor records that a rule has fired for the given trigger.
func MarkRuleInjectedFor(state *rules.SessionState, ruleID, trigger string) {
	if state.InjectedRules == nil {
		state.InjectedRules = make(map[string]time.Time)
	}
	state.InjectedRules[injectionKey(ruleID, trigger)] = time.Now()
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

// injectionKey builds the composite key used in SessionState.InjectedRules.
// The pipe separator is not valid in either a rule ID (which is
// "source:filename") or a command matcher (which shell-splits earlier in
// the pipeline), so collisions are structurally impossible.
func injectionKey(ruleID, trigger string) string {
	return ruleID + "|" + trigger
}

// migrateInjectedKeys rewrites legacy state entries whose keys are bare
// rule IDs (no pipe separator) into the composite form with an empty
// trigger, so older state files load cleanly.
func migrateInjectedKeys(state *rules.SessionState) {
	if state.InjectedRules == nil {
		return
	}
	for key, ts := range state.InjectedRules {
		if !strings.Contains(key, "|") {
			delete(state.InjectedRules, key)
			state.InjectedRules[key+"|"] = ts
		}
	}
}
