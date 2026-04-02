package hook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectAgent_Claude(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "cli")
	t.Setenv("GEMINI_SESSION_ID", "")
	assert.Equal(t, AgentClaude, DetectAgent())
}

func TestDetectAgent_Gemini(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("GEMINI_SESSION_ID", "xyz")
	assert.Equal(t, AgentGemini, DetectAgent())
}

func TestDetectAgent_BothSet_ClaudeWins(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "cli")
	t.Setenv("GEMINI_SESSION_ID", "xyz")
	assert.Equal(t, AgentClaude, DetectAgent())
}

func TestDetectAgent_NeitherSet_DefaultsGemini(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	t.Setenv("GEMINI_SESSION_ID", "")
	assert.Equal(t, AgentGemini, DetectAgent())
}

func TestFormatOutput_Claude(t *testing.T) {
	output := FormatOutput(AgentClaude, []string{"# Rule 1", "# Rule 2"})
	expected := "<system-reminder>\n# Rule 1\n\n# Rule 2\n</system-reminder>"
	assert.Equal(t, expected, output)
}

func TestFormatOutput_Gemini(t *testing.T) {
	output := FormatOutput(AgentGemini, []string{"# Rule 1", "# Rule 2"})
	expected := "# Rule 1\n\n# Rule 2"
	assert.Equal(t, expected, output)
}

func TestFormatOutput_Empty(t *testing.T) {
	assert.Empty(t, FormatOutput(AgentClaude, nil))
	assert.Empty(t, FormatOutput(AgentClaude, []string{}))
}

func TestFormatPreToolOutput_Claude(t *testing.T) {
	output := FormatPreToolOutput(AgentClaude, []string{"# Rule 1", "# Rule 2"})
	assert.Contains(t, output, `"hookSpecificOutput"`)
	assert.Contains(t, output, `"permissionDecision":"allow"`)
	assert.Contains(t, output, `"hookEventName":"PreToolUse"`)
	assert.Contains(t, output, "# Rule 1")
	assert.Contains(t, output, "# Rule 2")
}

func TestFormatPreToolOutput_Gemini(t *testing.T) {
	output := FormatPreToolOutput(AgentGemini, []string{"# Rule 1", "# Rule 2"})
	expected := "# Rule 1\n\n# Rule 2"
	assert.Equal(t, expected, output)
}

func TestFormatPreToolOutput_Empty(t *testing.T) {
	assert.Empty(t, FormatPreToolOutput(AgentClaude, nil))
	assert.Empty(t, FormatPreToolOutput(AgentClaude, []string{}))
}
