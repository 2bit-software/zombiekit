package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeRulesFile(t *testing.T, dir, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644))
}

func setupCommandRulesRepo(t *testing.T) (repoDir, rulesDir string) {
	t.Helper()
	repoDir = t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755))
	rulesDir = filepath.Join(repoDir, ".brains", "rules")
	require.NoError(t, os.MkdirAll(rulesDir, 0o755))
	return repoDir, rulesDir
}

// Authoring a command rule with a requires_files gate fires only when the
// gated file exists in the repo.
func TestService_ResolveForCommand_TaskfilePresent(t *testing.T) {
	repoDir, rulesDir := setupCommandRulesRepo(t)
	require.NoError(t, os.WriteFile(filepath.Join(repoDir, "Taskfile.yml"), []byte("version: 3"), 0o644))

	writeRulesFile(t, rulesDir, "go-use-taskfile.md", `---
commands:
  - "go test"
  - "go run"
requires_files:
  - Taskfile.yml
---
# Use the Taskfile

Run `+"`task dev -- test`"+` instead of bare `+"`go test`"+`.
`)

	svc := NewService(repoDir, t.TempDir())
	matches, err := svc.ResolveForCommand("go test ./...", repoDir)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, "go test", matches[0].Trigger)
	assert.Contains(t, matches[0].Rule.Body, "task dev -- test")
}

// The symmetrical requires_files_absent rule fires in repos with no
// Taskfile on disk.
func TestService_ResolveForCommand_TaskfileAbsent(t *testing.T) {
	repoDir, rulesDir := setupCommandRulesRepo(t)

	writeRulesFile(t, rulesDir, "go-suggest-taskfile.md", `---
commands:
  - "go test"
requires_files_absent:
  - Taskfile.yml
---
# Consider a Taskfile

Add one.
`)

	svc := NewService(repoDir, t.TempDir())
	matches, err := svc.ResolveForCommand("go test ./...", repoDir)
	require.NoError(t, err)
	require.Len(t, matches, 1)
	assert.Equal(t, "go test", matches[0].Trigger)
}

// A presence-gated rule stays silent when the gate file is missing.
func TestService_ResolveForCommand_GateBlocks(t *testing.T) {
	repoDir, rulesDir := setupCommandRulesRepo(t)

	writeRulesFile(t, rulesDir, "gated.md", `---
commands:
  - "go test"
requires_files:
  - Taskfile.yml
---
# body
`)

	svc := NewService(repoDir, t.TempDir())
	matches, err := svc.ResolveForCommand("go test ./...", repoDir)
	require.NoError(t, err)
	assert.Empty(t, matches)
}

// Commands that do not match any rule return no matches.
func TestService_ResolveForCommand_NoMatch(t *testing.T) {
	repoDir, rulesDir := setupCommandRulesRepo(t)
	writeRulesFile(t, rulesDir, "gt.md", `---
commands:
  - "go test"
---
# body
`)

	svc := NewService(repoDir, t.TempDir())
	matches, err := svc.ResolveForCommand("ls -la", repoDir)
	require.NoError(t, err)
	assert.Empty(t, matches)
}

// A rule with only commands declared must not be treated as unconditional
// (i.e. it must not fire at SessionStart via ResolveUnconditional).
func TestService_CommandRule_NotUnconditional(t *testing.T) {
	repoDir, rulesDir := setupCommandRulesRepo(t)
	writeRulesFile(t, rulesDir, "cmd-only.md", `---
commands:
  - "go test"
---
# body
`)

	svc := NewService(repoDir, t.TempDir())
	uncond, err := svc.ResolveUnconditional()
	require.NoError(t, err)
	for _, r := range uncond {
		assert.NotEqual(t, "cmd-only.md", r.FileName, "command-only rule must not be unconditional")
	}
}
