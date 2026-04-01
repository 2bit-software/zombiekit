package cmux

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseNewWorkspace(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		ref, err := parseNewWorkspace("OK workspace:9")
		require.NoError(t, err)
		assert.Equal(t, "workspace:9", ref)
	})

	t.Run("high number", func(t *testing.T) {
		ref, err := parseNewWorkspace("OK workspace:123")
		require.NoError(t, err)
		assert.Equal(t, "workspace:123", ref)
	})

	t.Run("empty string", func(t *testing.T) {
		_, err := parseNewWorkspace("")
		assert.Error(t, err)
	})

	t.Run("missing OK prefix", func(t *testing.T) {
		_, err := parseNewWorkspace("workspace:9")
		assert.Error(t, err)
	})

	t.Run("wrong prefix", func(t *testing.T) {
		_, err := parseNewWorkspace("ERROR workspace:9")
		assert.Error(t, err)
	})

	t.Run("extra fields", func(t *testing.T) {
		_, err := parseNewWorkspace("OK workspace:9 extra")
		assert.Error(t, err)
	})
}

func TestParseListWorkspaces(t *testing.T) {
	t.Run("multiple entries", func(t *testing.T) {
		input := `* workspace:5  zombiekit  [selected]
  workspace:4  clawbeam
  workspace:6  gogo`

		entries, err := parseListWorkspaces(input)
		require.NoError(t, err)
		require.Len(t, entries, 3)

		assert.Equal(t, "workspace:5", entries[0].ref)
		assert.Equal(t, "zombiekit", entries[0].name)
		assert.True(t, entries[0].selected)

		assert.Equal(t, "workspace:4", entries[1].ref)
		assert.Equal(t, "clawbeam", entries[1].name)
		assert.False(t, entries[1].selected)

		assert.Equal(t, "workspace:6", entries[2].ref)
		assert.Equal(t, "gogo", entries[2].name)
		assert.False(t, entries[2].selected)
	})

	t.Run("name with colon", func(t *testing.T) {
		input := `  workspace:9  DEV-186: implement session manager`

		entries, err := parseListWorkspaces(input)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, "DEV-186: implement session manager", entries[0].name)
	})

	t.Run("empty input", func(t *testing.T) {
		entries, err := parseListWorkspaces("")
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("whitespace only", func(t *testing.T) {
		entries, err := parseListWorkspaces("   \n  \n")
		require.NoError(t, err)
		assert.Empty(t, entries)
	})

	t.Run("unparseable non-empty input", func(t *testing.T) {
		_, err := parseListWorkspaces("gibberish\nmore gibberish")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "format may have changed")
	})

	t.Run("selected suffix stripped", func(t *testing.T) {
		input := `* workspace:1  myproject  [selected]`

		entries, err := parseListWorkspaces(input)
		require.NoError(t, err)
		require.Len(t, entries, 1)
		assert.Equal(t, "myproject", entries[0].name)
		assert.True(t, entries[0].selected)
	})
}

func TestFindByTicketID(t *testing.T) {
	entries := []workspaceEntry{
		{ref: "workspace:1", name: "DEV-100: first task"},
		{ref: "workspace:2", name: "DEV-200: second task"},
		{ref: "workspace:3", name: "unrelated workspace"},
	}

	t.Run("found", func(t *testing.T) {
		found := findByTicketID(entries, "DEV-100")
		require.NotNil(t, found)
		assert.Equal(t, "workspace:1", found.ref)
	})

	t.Run("not found", func(t *testing.T) {
		found := findByTicketID(entries, "DEV-999")
		assert.Nil(t, found)
	})

	t.Run("partial match rejected", func(t *testing.T) {
		found := findByTicketID(entries, "DEV-10")
		assert.Nil(t, found)
	})

	t.Run("empty entries", func(t *testing.T) {
		found := findByTicketID(nil, "DEV-100")
		assert.Nil(t, found)
	})
}

func TestBuildCommand(t *testing.T) {
	t.Run("empty env no prompt", func(t *testing.T) {
		cmd, err := buildCommand(nil, "claude", "")
		require.NoError(t, err)
		assert.Equal(t, "claude", cmd)
	})

	t.Run("empty map no prompt", func(t *testing.T) {
		cmd, err := buildCommand(map[string]string{}, "claude", "")
		require.NoError(t, err)
		assert.Equal(t, "claude", cmd)
	})

	t.Run("single var", func(t *testing.T) {
		cmd, err := buildCommand(map[string]string{"FOO": "bar"}, "claude", "")
		require.NoError(t, err)
		assert.Equal(t, `bash -c "export FOO='bar' && claude"`, cmd)
	})

	t.Run("multiple vars sorted", func(t *testing.T) {
		cmd, err := buildCommand(map[string]string{
			"ZEBRA": "z",
			"ALPHA": "a",
		}, "claude", "")
		require.NoError(t, err)
		assert.Equal(t, `bash -c "export ALPHA='a' ZEBRA='z' && claude"`, cmd)
	})

	t.Run("single quote in value", func(t *testing.T) {
		cmd, err := buildCommand(map[string]string{"MSG": "it's"}, "claude", "")
		require.NoError(t, err)
		// Inner: export MSG='it'\''s' && claude
		// Outer escapes \ to \\
		assert.Equal(t, `bash -c "export MSG='it'\\''s' && claude"`, cmd)
	})

	t.Run("special characters in value", func(t *testing.T) {
		cmd, err := buildCommand(map[string]string{
			"URL": "http://localhost:8666/DEV-186?foo=bar&baz=1",
		}, "claude", "")
		require.NoError(t, err)
		assert.Equal(t, `bash -c "export URL='http://localhost:8666/DEV-186?foo=bar&baz=1' && claude"`, cmd)
	})

	t.Run("dollar sign in value escaped", func(t *testing.T) {
		cmd, err := buildCommand(map[string]string{"PATH_VAR": "/home/$USER/bin"}, "claude", "")
		require.NoError(t, err)
		assert.Equal(t, `bash -c "export PATH_VAR='/home/\$USER/bin' && claude"`, cmd)
	})

	t.Run("invalid key with spaces", func(t *testing.T) {
		_, err := buildCommand(map[string]string{"BAD KEY": "val"}, "claude", "")
		assert.Error(t, err)
		assert.True(t, IsInvalidEnvKey(err))
	})

	t.Run("invalid key with dash", func(t *testing.T) {
		_, err := buildCommand(map[string]string{"BAD-KEY": "val"}, "claude", "")
		assert.Error(t, err)
		assert.True(t, IsInvalidEnvKey(err))
	})

	t.Run("valid key with underscore and digits", func(t *testing.T) {
		cmd, err := buildCommand(map[string]string{"_MY_VAR_2": "ok"}, "claude", "")
		require.NoError(t, err)
		assert.Equal(t, `bash -c "export _MY_VAR_2='ok' && claude"`, cmd)
	})

	t.Run("prompt without env", func(t *testing.T) {
		cmd, err := buildCommand(nil, "claude", "Read .ai/ticket.md and begin.")
		require.NoError(t, err)
		assert.Equal(t, "claude 'Read .ai/ticket.md and begin.'", cmd)
	})

	t.Run("prompt with env", func(t *testing.T) {
		cmd, err := buildCommand(map[string]string{"FOO": "bar"}, "claude", "Start working")
		require.NoError(t, err)
		assert.Equal(t, `bash -c "export FOO='bar' && claude 'Start working'"`, cmd)
	})

	t.Run("prompt with single quotes", func(t *testing.T) {
		cmd, err := buildCommand(nil, "claude", "Read the file — it's important")
		require.NoError(t, err)
		assert.Equal(t, "claude 'Read the file — it'\\''s important'", cmd)
	})

	t.Run("prompt with dollar sign", func(t *testing.T) {
		cmd, err := buildCommand(map[string]string{"X": "1"}, "claude", "Check $WORK_CALLBACK_URL")
		require.NoError(t, err)
		// Dollar sign in prompt is inside single quotes (preserved literally in inner layer)
		// but gets \$ escaped in the outer double-quote layer
		assert.Equal(t, `bash -c "export X='1' && claude 'Check \$WORK_CALLBACK_URL'"`, cmd)
	})
}

func TestBashQuote(t *testing.T) {
	assert.Equal(t, "'hello'", bashQuote("hello"))
	assert.Equal(t, "'it'\\''s'", bashQuote("it's"))
	assert.Equal(t, "''", bashQuote(""))
}
