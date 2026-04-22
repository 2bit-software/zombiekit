package hook

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpencodeFormatter_SessionStart(t *testing.T) {
	output := opencodeFormatter{}.FormatSessionStart([]string{"# Rule 1", "# Rule 2"})

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &decoded))

	hookOut, ok := decoded["hookSpecificOutput"].(map[string]any)
	require.True(t, ok, "expected hookSpecificOutput object")
	assert.Equal(t, "# Rule 1\n\n# Rule 2", hookOut["additionalContext"])

	_, hasDecision := decoded["decision"]
	_, hasEventName := decoded["hookEventName"]
	_, hasPermission := decoded["permissionDecision"]
	assert.False(t, hasDecision, "OpenCode envelope must not carry decision")
	assert.False(t, hasEventName, "OpenCode envelope must not carry hookEventName")
	assert.False(t, hasPermission, "OpenCode envelope must not carry permissionDecision")
}

func TestOpencodeFormatter_PostToolUse(t *testing.T) {
	output := opencodeFormatter{}.FormatPostToolUse([]string{"# Rule A"})

	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(output), &decoded))
	hookOut, ok := decoded["hookSpecificOutput"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "# Rule A", hookOut["additionalContext"])
}

func TestOpencodeFormatter_EmptyBodies(t *testing.T) {
	assert.Equal(t, "{}", opencodeFormatter{}.FormatSessionStart(nil))
	assert.Equal(t, "{}", opencodeFormatter{}.FormatSessionStart([]string{}))
	assert.Equal(t, "{}", opencodeFormatter{}.FormatPostToolUse(nil))
	assert.Equal(t, "{}", opencodeFormatter{}.FormatPostToolUse([]string{}))
}

func TestOpencodeFormatter_PreToolUseAndSessionEnd_AreNoOps(t *testing.T) {
	assert.Empty(t, opencodeFormatter{}.FormatPreToolUse([]string{"# Rule"}))
	assert.Empty(t, opencodeFormatter{}.FormatPreToolUse(nil))
	assert.Empty(t, opencodeFormatter{}.FormatSessionEnd([]string{"# Rule"}))
	assert.Empty(t, opencodeFormatter{}.FormatSessionEnd(nil))
}

func TestOpencodeEditor_ExtractFilePaths_FileTools(t *testing.T) {
	for _, tool := range []string{"write", "edit", "multi-edit"} {
		paths := opencodeFormatter{}.ExtractFilePaths(&HookEvent{
			ToolName:  tool,
			ToolInput: &ToolInput{FilePath: "internal/hook/handler.go"},
		})
		assert.Equal(t, []string{"internal/hook/handler.go"}, paths, "tool=%s", tool)
	}
}

func TestOpencodeEditor_ExtractFilePaths_CamelCase(t *testing.T) {
	paths := opencodeFormatter{}.ExtractFilePaths(&HookEvent{
		ToolName:  "write",
		ToolInput: &ToolInput{FilePathAlt: "cmd/main.go"},
	})
	assert.Equal(t, []string{"cmd/main.go"}, paths)
}

func TestOpencodeEditor_ExtractFilePaths_MultiEditFallbackToEdits(t *testing.T) {
	paths := opencodeFormatter{}.ExtractFilePaths(&HookEvent{
		ToolName: "multi-edit",
		ToolInput: &ToolInput{
			Edits: []EditEntry{{FilePath: "pkg/a.go"}},
		},
	})
	assert.Equal(t, []string{"pkg/a.go"}, paths)
}

func TestOpencodeEditor_ExtractFilePaths_IgnoresOtherEditorToolNames(t *testing.T) {
	for _, tool := range []string{"Read", "Write", "Edit", "MultiEdit", "read_file", "write_file", "replace"} {
		paths := opencodeFormatter{}.ExtractFilePaths(&HookEvent{
			ToolName:  tool,
			ToolInput: &ToolInput{FilePath: "/tmp/x.go"},
		})
		assert.Nil(t, paths, "OpenCode editor must not recognize %q", tool)
	}
}

func TestOpencodeEditor_ExtractFilePaths_NilToolInput(t *testing.T) {
	paths := opencodeFormatter{}.ExtractFilePaths(&HookEvent{ToolName: "write"})
	assert.Nil(t, paths)
}

func TestOpencodeEditor_IsShellTool(t *testing.T) {
	assert.True(t, opencodeFormatter{}.IsShellTool("bash"))
	assert.False(t, opencodeFormatter{}.IsShellTool("Bash"))
	assert.False(t, opencodeFormatter{}.IsShellTool("run_shell_command"))
}
