package hook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClaudeFormatter_SessionStart(t *testing.T) {
	output := claudeFormatter{}.FormatSessionStart([]string{"# Rule 1", "# Rule 2"})
	expected := "<system-reminder>\n# Rule 1\n\n# Rule 2\n</system-reminder>"
	assert.Equal(t, expected, output)
}

func TestClaudeFormatter_SessionStart_Empty(t *testing.T) {
	assert.Empty(t, claudeFormatter{}.FormatSessionStart(nil))
	assert.Empty(t, claudeFormatter{}.FormatSessionStart([]string{}))
}

func TestClaudeFormatter_PreToolUse(t *testing.T) {
	output := claudeFormatter{}.FormatPreToolUse([]string{"# Rule 1", "# Rule 2"})
	assert.Contains(t, output, `"hookSpecificOutput"`)
	assert.Contains(t, output, `"permissionDecision":"allow"`)
	assert.Contains(t, output, `"hookEventName":"PreToolUse"`)
	assert.Contains(t, output, "# Rule 1")
	assert.Contains(t, output, "# Rule 2")
}

func TestClaudeFormatter_PreToolUse_Empty(t *testing.T) {
	assert.Empty(t, claudeFormatter{}.FormatPreToolUse(nil))
	assert.Empty(t, claudeFormatter{}.FormatPreToolUse([]string{}))
}

func TestClaudeFormatter_SessionEnd(t *testing.T) {
	assert.Empty(t, claudeFormatter{}.FormatSessionEnd([]string{"# Rule 1"}))
	assert.Empty(t, claudeFormatter{}.FormatSessionEnd(nil))
}
