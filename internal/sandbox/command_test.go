package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildSbxExecCommand(t *testing.T) {
	t.Run("no env no prompt", func(t *testing.T) {
		cmd, err := buildSbxExecCommand("zk-dev-123", nil, "claude --dangerously-skip-permissions", "")
		require.NoError(t, err)
		assert.Equal(t, `bash -c "sbx exec -it zk-dev-123 claude --dangerously-skip-permissions"`, cmd)
	})

	t.Run("with env", func(t *testing.T) {
		env := map[string]string{"WORK_CALLBACK_URL": "http://host.docker.internal:8666/DEV-123"}
		cmd, err := buildSbxExecCommand("zk-dev-123", env, "claude --dangerously-skip-permissions", "")
		require.NoError(t, err)
		assert.Equal(t,
			`bash -c "sbx exec -it -e WORK_CALLBACK_URL='http://host.docker.internal:8666/DEV-123' zk-dev-123 claude --dangerously-skip-permissions"`,
			cmd,
		)
	})

	t.Run("with prompt", func(t *testing.T) {
		cmd, err := buildSbxExecCommand("zk-dev-123", nil, "claude --dangerously-skip-permissions", "Read .ai/ticket.md")
		require.NoError(t, err)
		assert.Equal(t,
			`bash -c "sbx exec -it zk-dev-123 claude --dangerously-skip-permissions -p 'Read .ai/ticket.md'"`,
			cmd,
		)
	})

	t.Run("with env and prompt", func(t *testing.T) {
		env := map[string]string{"WORK_CALLBACK_URL": "http://host.docker.internal:8666/DEV-123"}
		cmd, err := buildSbxExecCommand("zk-dev-123", env, "claude --dangerously-skip-permissions", "Read .ai/ticket.md")
		require.NoError(t, err)
		assert.Equal(t,
			`bash -c "sbx exec -it -e WORK_CALLBACK_URL='http://host.docker.internal:8666/DEV-123' zk-dev-123 claude --dangerously-skip-permissions -p 'Read .ai/ticket.md'"`,
			cmd,
		)
	})

	t.Run("multiple env sorted", func(t *testing.T) {
		env := map[string]string{"ZEBRA": "z", "ALPHA": "a"}
		cmd, err := buildSbxExecCommand("zk-dev-123", env, "claude", "")
		require.NoError(t, err)
		assert.Equal(t, `bash -c "sbx exec -it -e ALPHA='a' -e ZEBRA='z' zk-dev-123 claude"`, cmd)
	})

	t.Run("single quote in prompt escaped", func(t *testing.T) {
		cmd, err := buildSbxExecCommand("zk-dev-123", nil, "claude", "it's important")
		require.NoError(t, err)
		// Inner single-quote escaping ('\'' ) gets \ escaped to \\ in the outer double-quote layer.
		assert.Equal(t, `bash -c "sbx exec -it zk-dev-123 claude -p 'it'\\''s important'"`, cmd)
	})

	t.Run("dollar sign in env value escaped", func(t *testing.T) {
		env := map[string]string{"TOKEN": "sk-$ecret"}
		cmd, err := buildSbxExecCommand("zk-dev-123", env, "claude", "")
		require.NoError(t, err)
		assert.Equal(t, `bash -c "sbx exec -it -e TOKEN='sk-\$ecret' zk-dev-123 claude"`, cmd)
	})

	t.Run("invalid env key", func(t *testing.T) {
		_, err := buildSbxExecCommand("zk-dev-123", map[string]string{"BAD KEY": "val"}, "claude", "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid env key")
	})
}

func TestNewCommandBuilder_SandboxNameFromEnv(t *testing.T) {
	cfg := Config{CallbackHost: "host.docker.internal"}
	builder := NewCommandBuilder(cfg)

	t.Run("uses sandbox name from env", func(t *testing.T) {
		env := map[string]string{
			EnvSandboxName:    "zk-dev-123",
			"WORK_CALLBACK_URL": "http://localhost:8666/DEV-123",
		}
		cmd, cwd, err := builder("/tmp/worktrees/DEV-123", env, "claude", "hello")
		require.NoError(t, err)
		assert.Contains(t, cmd, "zk-dev-123")
		assert.NotContains(t, cmd, EnvSandboxName, "sandbox name key should be stripped from command")
		assert.Contains(t, cmd, "host.docker.internal", "callback URL should be rewritten")
		assert.Equal(t, "/tmp/worktrees/DEV-123", cwd)
	})

	t.Run("errors when sandbox name missing", func(t *testing.T) {
		env := map[string]string{"WORK_CALLBACK_URL": "http://localhost:8666/DEV-123"}
		_, _, err := builder("/tmp/worktrees/DEV-123", env, "claude", "hello")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), EnvSandboxName)
	})
}
