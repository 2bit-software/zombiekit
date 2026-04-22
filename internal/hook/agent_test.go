package hook

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveEditor_Flag_Claude(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	editor, source, err := ResolveEditor("claude")
	assert.NoError(t, err)
	assert.Equal(t, AgentClaude, editor)
	assert.Equal(t, EditorSourceFlag, source)
}

func TestResolveEditor_Flag_Gemini(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	editor, source, err := ResolveEditor("gemini")
	assert.NoError(t, err)
	assert.Equal(t, AgentGemini, editor)
	assert.Equal(t, EditorSourceFlag, source)
}

func TestResolveEditor_Flag_OpenCode(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	editor, source, err := ResolveEditor("opencode")
	assert.NoError(t, err)
	assert.Equal(t, AgentOpenCode, editor)
	assert.Equal(t, EditorSourceFlag, source)
}

func TestResolveEditor_Flag_Unknown(t *testing.T) {
	_, _, err := ResolveEditor("frobnitz")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown editor")
	assert.Contains(t, err.Error(), "frobnitz")
	assert.Contains(t, err.Error(), "claude")
	assert.Contains(t, err.Error(), "gemini")
	assert.Contains(t, err.Error(), "opencode")
}

func TestResolveEditor_Env_Claude(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "cli")
	editor, source, err := ResolveEditor("")
	assert.NoError(t, err)
	assert.Equal(t, AgentClaude, editor)
	assert.Equal(t, EditorSourceEnv, source)
}

func TestResolveEditor_NoEnv_DefaultsClaude(t *testing.T) {
	t.Setenv("CLAUDE_CODE_ENTRYPOINT", "")
	editor, source, err := ResolveEditor("")
	assert.NoError(t, err)
	assert.Equal(t, AgentClaude, editor)
	assert.Equal(t, EditorSourceDefault, source)
}
