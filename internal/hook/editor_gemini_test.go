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
	assert.Equal(t, "{\"decision\":\"allow\"}", geminiFormatter{}.FormatSessionStart(nil))
	assert.Equal(t, "{\"decision\":\"allow\"}", geminiFormatter{}.FormatSessionStart([]string{}))
}

func TestGeminiFormatter_EmptyBodies_PreToolUse(t *testing.T) {
	assert.Equal(t, "{\"decision\":\"allow\"}", geminiFormatter{}.FormatPreToolUse(nil))
	assert.Equal(t, "{\"decision\":\"allow\"}", geminiFormatter{}.FormatPreToolUse([]string{}))
}

func TestGeminiFormatter_SessionEnd(t *testing.T) {
	assert.Empty(t, geminiFormatter{}.FormatSessionEnd([]string{"# Rule"}))
	assert.Empty(t, geminiFormatter{}.FormatSessionEnd(nil))
}

func TestGeminiEditor_ExtractFilePaths_FileTools(t *testing.T) {
	for _, tool := range []string{"read_file", "write_file", "replace"} {
		paths := geminiFormatter{}.ExtractFilePaths(&HookEvent{
			ToolName:  tool,
			ToolInput: &ToolInput{FilePath: "internal/hook/handler.go"},
		})
		assert.Equal(t, []string{"internal/hook/handler.go"}, paths, "tool=%s", tool)
	}
}

func TestGeminiEditor_ExtractFilePaths_HandlesBothRelativeAndAbsolutePaths(t *testing.T) {
	rel := geminiFormatter{}.ExtractFilePaths(&HookEvent{
		ToolName:  "read_file",
		ToolInput: &ToolInput{FilePath: "README.md"},
	})
	assert.Equal(t, []string{"README.md"}, rel)

	abs := geminiFormatter{}.ExtractFilePaths(&HookEvent{
		ToolName:  "write_file",
		ToolInput: &ToolInput{FilePath: "/Users/morgan/tmp.txt"},
	})
	assert.Equal(t, []string{"/Users/morgan/tmp.txt"}, abs)
}

func TestGeminiEditor_ExtractFilePaths_IgnoresClaudeToolNames(t *testing.T) {
	for _, tool := range []string{"Read", "Write", "Edit", "MultiEdit"} {
		paths := geminiFormatter{}.ExtractFilePaths(&HookEvent{
			ToolName:  tool,
			ToolInput: &ToolInput{FilePath: "/tmp/x.go"},
		})
		assert.Nil(t, paths, "Gemini editor must not recognize Claude tool %q", tool)
	}
}

func TestGeminiEditor_ExtractFilePaths_CamelCase(t *testing.T) {
	paths := geminiFormatter{}.ExtractFilePaths(&HookEvent{
		ToolName:  "read_file",
		ToolInput: &ToolInput{FilePathAlt: "cmd/main.go"},
	})
	assert.Equal(t, []string{"cmd/main.go"}, paths)
}

func TestGeminiEditor_ExtractFilePaths_ToolResponse(t *testing.T) {
	paths := geminiFormatter{}.ExtractFilePaths(&HookEvent{
		ToolName:     "write_file",
		ToolResponse: &ToolResponse{FilePath: "out.txt", Success: true},
	})
	assert.Equal(t, []string{"out.txt"}, paths)
}

func TestGeminiEditor_IsShellTool(t *testing.T) {
	assert.True(t, geminiFormatter{}.IsShellTool("run_shell_command"))
	assert.False(t, geminiFormatter{}.IsShellTool("Bash"))
	assert.False(t, geminiFormatter{}.IsShellTool("read_file"))
}
