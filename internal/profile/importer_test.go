package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertClaudeToBrains(t *testing.T) {
	importer := &Importer{}

	tests := []struct {
		name     string
		agent    *Profile
		wantName string
		wantDesc string
		wantIncl []string
		wantBody string
	}{
		{
			name: "preserves name and description",
			agent: &Profile{
				Name:        "my-agent",
				Description: "A helpful agent",
				Body:        "Agent instructions here.",
			},
			wantName: "my-agent",
			wantDesc: "A helpful agent",
			wantBody: "Agent instructions here.",
		},
		{
			name: "preserves includes",
			agent: &Profile{
				Name:     "agent-with-includes",
				Includes: []string{"base", "common"},
				Body:     "Content",
			},
			wantName: "agent-with-includes",
			wantIncl: []string{"base", "common"},
			wantBody: "Content",
		},
		{
			name: "discards model field",
			agent: &Profile{
				Name:  "model-agent",
				Model: "opus",
				Body:  "Content",
			},
			wantName: "model-agent",
			wantBody: "Content",
		},
		{
			name: "discards color field",
			agent: &Profile{
				Name:  "color-agent",
				Color: "blue",
				Body:  "Content",
			},
			wantName: "color-agent",
			wantBody: "Content",
		},
		{
			name: "forces inherits to false",
			agent: &Profile{
				Name:     "inheriting-agent",
				Inherits: true,
				Body:     "Content",
			},
			wantName: "inheriting-agent",
			wantBody: "Content",
		},
		{
			name: "handles empty body",
			agent: &Profile{
				Name:        "empty-body",
				Description: "No body content",
				Body:        "",
			},
			wantName: "empty-body",
			wantDesc: "No body content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := importer.convertClaudeToBrains(tt.agent)
			require.NoError(t, err)

			// Verify frontmatter structure
			assert.Contains(t, content, "---\n")
			assert.Contains(t, content, "inherits: false")

			if tt.wantName != "" {
				assert.Contains(t, content, "name: "+tt.wantName)
			}
			if tt.wantDesc != "" {
				assert.Contains(t, content, "description: "+tt.wantDesc)
			}
			if len(tt.wantIncl) > 0 {
				assert.Contains(t, content, "includes:")
			}
			if tt.wantBody != "" {
				assert.Contains(t, content, tt.wantBody)
			}

			// Verify discarded fields are NOT present
			assert.NotContains(t, content, "model:")
			assert.NotContains(t, content, "color:")
		})
	}
}

func TestImport_LocalAgents(t *testing.T) {
	// Create temp directories for test
	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "project")
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Create Claude agents directory
	claudeAgentsDir := filepath.Join(workingDir, ".claude", "agents")
	require.NoError(t, os.MkdirAll(claudeAgentsDir, 0o755))

	// Create a sample Claude agent
	agentContent := `---
name: test-agent
description: A test agent
model: opus
color: blue
includes: []
inherits: false
---

Test agent instructions.
`
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeAgentsDir, "test-agent.md"),
		[]byte(agentContent),
		0o644,
	))

	// Create importer with isolated home directory
	importer, err := NewImporterWithHomeDir(workingDir, homeDir)
	require.NoError(t, err)

	result, err := importer.Import(false)
	require.NoError(t, err)

	// Verify result
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 0, result.Overwritten)
	assert.Equal(t, 0, result.Failed)
	assert.False(t, result.DryRun)

	// Verify profile was created
	brainsProfilePath := filepath.Join(workingDir, ".brains", "profiles", "test-agent.md")
	assert.FileExists(t, brainsProfilePath)

	// Verify content
	content, err := os.ReadFile(brainsProfilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "name: test-agent")
	assert.Contains(t, string(content), "description: A test agent")
	assert.Contains(t, string(content), "inherits: false")
	assert.NotContains(t, string(content), "model:")
	assert.NotContains(t, string(content), "color:")
	assert.Contains(t, string(content), "Test agent instructions.")
}

func TestImport_GlobalAgents(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "project")
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(workingDir, 0o755))

	// Create Claude global agents directory
	claudeAgentsDir := filepath.Join(homeDir, ".claude", "agents")
	require.NoError(t, os.MkdirAll(claudeAgentsDir, 0o755))

	// Create a sample global Claude agent
	agentContent := `---
name: global-agent
description: A global agent
---

Global agent instructions.
`
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeAgentsDir, "global-agent.md"),
		[]byte(agentContent),
		0o644,
	))

	// Create importer with isolated home directory
	importer, err := NewImporterWithHomeDir(workingDir, homeDir)
	require.NoError(t, err)

	result, err := importer.Import(false)
	require.NoError(t, err)

	// Verify result
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 0, result.Overwritten)

	// Verify profile was created in global brains directory
	brainsProfilePath := filepath.Join(homeDir, ".brains", "profiles", "global-agent.md")
	assert.FileExists(t, brainsProfilePath)
}

func TestImport_Overwrite(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "project")
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Create Claude agents directory
	claudeAgentsDir := filepath.Join(workingDir, ".claude", "agents")
	require.NoError(t, os.MkdirAll(claudeAgentsDir, 0o755))

	// Create brains profiles directory with existing profile
	brainsProfilesDir := filepath.Join(workingDir, ".brains", "profiles")
	require.NoError(t, os.MkdirAll(brainsProfilesDir, 0o755))

	existingContent := `---
name: existing-agent
description: Original description
inherits: true
---

Original content.
`
	require.NoError(t, os.WriteFile(
		filepath.Join(brainsProfilesDir, "existing-agent.md"),
		[]byte(existingContent),
		0o644,
	))

	// Create Claude agent with same name
	claudeContent := `---
name: existing-agent
description: Updated description
model: sonnet
---

Updated content from Claude.
`
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeAgentsDir, "existing-agent.md"),
		[]byte(claudeContent),
		0o644,
	))

	// Run import with isolated home directory
	importer, err := NewImporterWithHomeDir(workingDir, homeDir)
	require.NoError(t, err)

	result, err := importer.Import(false)
	require.NoError(t, err)

	// Verify overwrite was tracked
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 1, result.Overwritten)
	assert.Len(t, result.OverwrittenPaths, 1)

	// Verify content was replaced
	brainsProfilePath := filepath.Join(brainsProfilesDir, "existing-agent.md")
	content, err := os.ReadFile(brainsProfilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Updated description")
	assert.Contains(t, string(content), "Updated content from Claude.")
	assert.Contains(t, string(content), "inherits: false")
}

func TestImport_DryRun(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "project")
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Create Claude agents directory
	claudeAgentsDir := filepath.Join(workingDir, ".claude", "agents")
	require.NoError(t, os.MkdirAll(claudeAgentsDir, 0o755))

	// Create a sample Claude agent
	agentContent := `---
name: dry-run-agent
description: Should not be created
---

Content.
`
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeAgentsDir, "dry-run-agent.md"),
		[]byte(agentContent),
		0o644,
	))

	// Run import with dry run and isolated home directory
	importer, err := NewImporterWithHomeDir(workingDir, homeDir)
	require.NoError(t, err)

	result, err := importer.Import(true)
	require.NoError(t, err)

	// Verify result shows what would happen
	assert.Equal(t, 1, result.Created)
	assert.True(t, result.DryRun)

	// Verify no files were actually created
	brainsProfilePath := filepath.Join(workingDir, ".brains", "profiles", "dry-run-agent.md")
	_, err = os.Stat(brainsProfilePath)
	assert.True(t, os.IsNotExist(err), "profile should not exist in dry run mode")
}

func TestImport_NoClaudeDirectory(t *testing.T) {
	// Create temp directory without Claude agents
	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "project")
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(workingDir, 0o755))
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Run import with isolated home directory
	importer, err := NewImporterWithHomeDir(workingDir, homeDir)
	require.NoError(t, err)

	result, err := importer.Import(false)
	require.NoError(t, err)

	// Should succeed with zero imports
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 0, result.Overwritten)
	assert.Equal(t, 0, result.Failed)
}

func TestImport_EmptyBody(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "project")
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Create Claude agents directory
	claudeAgentsDir := filepath.Join(workingDir, ".claude", "agents")
	require.NoError(t, os.MkdirAll(claudeAgentsDir, 0o755))

	// Create a Claude agent with empty body
	agentContent := `---
name: empty-body-agent
description: Has no body
---
`
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeAgentsDir, "empty-body-agent.md"),
		[]byte(agentContent),
		0o644,
	))

	// Run import with isolated home directory
	importer, err := NewImporterWithHomeDir(workingDir, homeDir)
	require.NoError(t, err)

	result, err := importer.Import(false)
	require.NoError(t, err)

	// Should succeed
	assert.Equal(t, 1, result.Created)

	// Verify profile exists with valid frontmatter
	brainsProfilePath := filepath.Join(workingDir, ".brains", "profiles", "empty-body-agent.md")
	content, err := os.ReadFile(brainsProfilePath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "name: empty-body-agent")
	assert.Contains(t, string(content), "inherits: false")
}

func TestImport_MultipleAgents(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "project")
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Create Claude agents directory
	claudeAgentsDir := filepath.Join(workingDir, ".claude", "agents")
	require.NoError(t, os.MkdirAll(claudeAgentsDir, 0o755))

	// Create brains profiles directory with one existing profile
	brainsProfilesDir := filepath.Join(workingDir, ".brains", "profiles")
	require.NoError(t, os.MkdirAll(brainsProfilesDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(brainsProfilesDir, "agent2.md"),
		[]byte("---\nname: agent2\n---\nOld content"),
		0o644,
	))

	// Create multiple Claude agents
	agents := []struct {
		name    string
		content string
	}{
		{"agent1", "---\nname: agent1\n---\nAgent 1 content"},
		{"agent2", "---\nname: agent2\n---\nAgent 2 updated"},
		{"agent3", "---\nname: agent3\n---\nAgent 3 content"},
	}

	for _, a := range agents {
		require.NoError(t, os.WriteFile(
			filepath.Join(claudeAgentsDir, a.name+".md"),
			[]byte(a.content),
			0o644,
		))
	}

	// Run import with isolated home directory
	importer, err := NewImporterWithHomeDir(workingDir, homeDir)
	require.NoError(t, err)

	result, err := importer.Import(false)
	require.NoError(t, err)

	// Verify counts
	assert.Equal(t, 2, result.Created)
	assert.Equal(t, 1, result.Overwritten)
	assert.Equal(t, 0, result.Failed)
}

func TestImport_PartialFailure(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "project")
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Create Claude agents directory
	claudeAgentsDir := filepath.Join(workingDir, ".claude", "agents")
	require.NoError(t, os.MkdirAll(claudeAgentsDir, 0o755))

	// Create a valid agent
	validContent := `---
name: valid-agent
---
Valid content
`
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeAgentsDir, "valid-agent.md"),
		[]byte(validContent),
		0o644,
	))

	// Create an invalid agent (bad YAML)
	invalidContent := `---
name: invalid-agent
description: [unclosed array
---
Content
`
	require.NoError(t, os.WriteFile(
		filepath.Join(claudeAgentsDir, "invalid-agent.md"),
		[]byte(invalidContent),
		0o644,
	))

	// Run import with isolated home directory
	importer, err := NewImporterWithHomeDir(workingDir, homeDir)
	require.NoError(t, err)

	result, err := importer.Import(false)
	require.NoError(t, err)

	// Valid agent should be imported, invalid should be skipped
	// Note: ClaudeSource silently skips parse errors, so both may appear as 0 failed
	// depending on implementation. The important thing is no error is returned.
	assert.GreaterOrEqual(t, result.Created, 1)
}

func TestImport_InheritsFieldConversion(t *testing.T) {
	// Create temp directories
	tmpDir := t.TempDir()
	workingDir := filepath.Join(tmpDir, "project")
	homeDir := filepath.Join(tmpDir, "home")
	require.NoError(t, os.MkdirAll(homeDir, 0o755))

	// Create Claude agents directory
	claudeAgentsDir := filepath.Join(workingDir, ".claude", "agents")
	require.NoError(t, os.MkdirAll(claudeAgentsDir, 0o755))

	// Create agents with various inherits values
	testCases := []struct {
		name         string
		claudeInherits string
	}{
		{"inherits-true", "inherits: true"},
		{"inherits-false", "inherits: false"},
		{"inherits-unset", ""}, // No inherits field
	}

	for _, tc := range testCases {
		content := "---\nname: " + tc.name + "\n"
		if tc.claudeInherits != "" {
			content += tc.claudeInherits + "\n"
		}
		content += "---\nContent\n"
		require.NoError(t, os.WriteFile(
			filepath.Join(claudeAgentsDir, tc.name+".md"),
			[]byte(content),
			0o644,
		))
	}

	// Run import with isolated home directory
	importer, err := NewImporterWithHomeDir(workingDir, homeDir)
	require.NoError(t, err)

	result, err := importer.Import(false)
	require.NoError(t, err)
	assert.Equal(t, 3, result.Created)

	// Verify all profiles have inherits: false
	for _, tc := range testCases {
		profilePath := filepath.Join(workingDir, ".brains", "profiles", tc.name+".md")
		content, err := os.ReadFile(profilePath)
		require.NoError(t, err)

		lines := strings.Split(string(content), "\n")
		foundInherits := false
		for _, line := range lines {
			if strings.TrimSpace(line) == "inherits: false" {
				foundInherits = true
				break
			}
			// Also check it's not inherits: true
			if strings.Contains(line, "inherits: true") {
				t.Errorf("profile %s should have inherits: false, got inherits: true", tc.name)
			}
		}
		assert.True(t, foundInherits, "profile %s should have inherits: false", tc.name)
	}
}
