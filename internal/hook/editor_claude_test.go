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

func TestClaudeEditor_ExtractFilePaths_Read(t *testing.T) {
	paths := claudeFormatter{}.ExtractFilePaths(&HookEvent{
		ToolName:  "Read",
		ToolInput: &ToolInput{FilePath: "/tmp/x.go"},
	})
	assert.Equal(t, []string{"/tmp/x.go"}, paths)
}

func TestClaudeEditor_ExtractFilePaths_WriteEdit(t *testing.T) {
	for _, tool := range []string{"Write", "Edit"} {
		paths := claudeFormatter{}.ExtractFilePaths(&HookEvent{
			ToolName:  tool,
			ToolInput: &ToolInput{FilePath: "/tmp/x.go"},
		})
		assert.Equal(t, []string{"/tmp/x.go"}, paths, "tool=%s", tool)
	}
}

func TestClaudeEditor_ExtractFilePaths_MultiEdit(t *testing.T) {
	paths := claudeFormatter{}.ExtractFilePaths(&HookEvent{
		ToolName: "MultiEdit",
		ToolInput: &ToolInput{
			Edits: []EditEntry{
				{FilePath: "/tmp/a.go"},
				{FilePath: "/tmp/b.go"},
			},
		},
	})
	assert.Equal(t, []string{"/tmp/a.go", "/tmp/b.go"}, paths)
}

func TestClaudeEditor_ExtractFilePaths_IgnoresGeminiToolNames(t *testing.T) {
	for _, tool := range []string{"read_file", "write_file", "replace"} {
		paths := claudeFormatter{}.ExtractFilePaths(&HookEvent{
			ToolName:  tool,
			ToolInput: &ToolInput{FilePath: "/tmp/x.go"},
		})
		assert.Nil(t, paths, "Claude editor must not recognize Gemini tool %q", tool)
	}
}

func TestClaudeEditor_IsShellTool(t *testing.T) {
	assert.True(t, claudeFormatter{}.IsShellTool("Bash"))
	assert.False(t, claudeFormatter{}.IsShellTool("run_shell_command"))
	assert.False(t, claudeFormatter{}.IsShellTool("Read"))
}
