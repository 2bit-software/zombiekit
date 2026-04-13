package hook

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiFormatter_SessionStart(t *testing.T) {
	output := geminiFormatter{}.FormatSessionStart([]string{"# Rule 1", "# Rule 2"})

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &decoded))

	hookOut, ok := decoded["hookSpecificOutput"].(map[string]any)
	require.True(t, ok, "expected hookSpecificOutput object")
	assert.Equal(t, "# Rule 1\n\n# Rule 2", hookOut["additionalContext"])

	_, hasEventName := decoded["hookEventName"]
	_, hasPermission := decoded["permissionDecision"]
	assert.False(t, hasEventName, "Gemini envelope must not carry hookEventName")
	assert.False(t, hasPermission, "Gemini envelope must not carry permissionDecision")
}

func TestGeminiFormatter_PreToolUse(t *testing.T) {
	output := geminiFormatter{}.FormatPreToolUse([]string{"# Rule A"})

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &decoded))
	hookOut, ok := decoded["hookSpecificOutput"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "# Rule A", hookOut["additionalContext"])
}

func TestGeminiFormatter_EmptyBodies_SessionStart(t *testing.T) {
	assert.Equal(t, "{}", geminiFormatter{}.FormatSessionStart(nil))
	assert.Equal(t, "{}", geminiFormatter{}.FormatSessionStart([]string{}))
}

func TestGeminiFormatter_EmptyBodies_PreToolUse(t *testing.T) {
	assert.Equal(t, "{}", geminiFormatter{}.FormatPreToolUse(nil))
	assert.Equal(t, "{}", geminiFormatter{}.FormatPreToolUse([]string{}))
}

func TestGeminiFormatter_SessionEnd(t *testing.T) {
	assert.Empty(t, geminiFormatter{}.FormatSessionEnd([]string{"# Rule"}))
	assert.Empty(t, geminiFormatter{}.FormatSessionEnd(nil))
}
