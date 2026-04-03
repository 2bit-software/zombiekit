package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsShim(t *testing.T) {
	if !IsShim("Call mcp__zombiekit__profile-compose with profiles") {
		t.Error("expected shim body to be detected")
	}
	if IsShim("This is a regular skill body with instructions.") {
		t.Error("expected non-shim body to not be detected")
	}
	if IsShim("") {
		t.Error("expected empty body to not be detected")
	}
}

func TestDiscoverSkills(t *testing.T) {
	// Create a temp directory structure mimicking ~/.claude/skills/
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".claude", "skills")

	// Regular skill
	realSkillDir := filepath.Join(skillsDir, "my-tool")
	mustMkdirAll(t, realSkillDir)
	mustWriteFile(t, filepath.Join(realSkillDir, "SKILL.md"), `---
name: my-tool
description: A real tool
allowed-tools: Bash(*)
---

Do stuff here.
`)

	// Skill with supporting scripts
	scriptSkillDir := filepath.Join(skillsDir, "with-scripts")
	mustMkdirAll(t, filepath.Join(scriptSkillDir, "scripts"))
	mustWriteFile(t, filepath.Join(scriptSkillDir, "SKILL.md"), `---
name: with-scripts
description: Has scripts
---

Run scripts.
`)
	mustWriteFile(t, filepath.Join(scriptSkillDir, "scripts", "run.sh"), "#!/bin/bash\necho hi")

	// Shim skill (should be excluded)
	shimSkillDir := filepath.Join(skillsDir, "shim-skill")
	mustMkdirAll(t, shimSkillDir)
	mustWriteFile(t, filepath.Join(shimSkillDir, "SKILL.md"), `---
name: shim-skill
description: A shim
allowed-tools: mcp__zombiekit__profile-compose
---

Call `+"`mcp__zombiekit__profile-compose`"+` with `+"`profiles: [\"shim-skill\"]`"+` and follow the returned instructions exactly.
`)

	// Discover using the local .claude/skills/ path
	items, err := discoverSkillsInDirs([]string{skillsDir})
	if err != nil {
		t.Fatalf("DiscoverSkills error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 items (shim excluded), got %d: %+v", len(items), items)
	}

	names := map[string]bool{}
	for _, item := range items {
		names[item.Name] = true
		if item.Type != "skill" {
			t.Errorf("expected type 'skill', got %q for %s", item.Type, item.Name)
		}
	}

	if !names["my-tool"] {
		t.Error("missing my-tool")
	}
	if !names["with-scripts"] {
		t.Error("missing with-scripts")
	}
	if names["shim-skill"] {
		t.Error("shim-skill should have been excluded")
	}
}

func TestDiscoverAgents(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, ".claude", "agents")
	mustMkdirAll(t, agentsDir)

	mustWriteFile(t, filepath.Join(agentsDir, "my-agent.md"), `---
name: my-agent
description: Does agent things
model: opus
color: cyan
---

You are an agent.
`)

	mustWriteFile(t, filepath.Join(agentsDir, "another.md"), `---
name: another
description: Another agent
model: sonnet
skills: skill1,skill2
---

Another agent prompt.
`)

	// Non-md file should be ignored
	mustWriteFile(t, filepath.Join(agentsDir, "readme.txt"), "not an agent")

	items, err := discoverAgentsInDirs([]string{agentsDir})
	if err != nil {
		t.Fatalf("DiscoverAgents error: %v", err)
	}

	if len(items) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(items))
	}

	for _, item := range items {
		if item.Type != "agent" {
			t.Errorf("expected type 'agent', got %q for %s", item.Type, item.Name)
		}
	}
}

func TestDiscoverAll_NameCollision(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".claude", "skills")
	agentsDir := filepath.Join(dir, ".claude", "agents")

	// Skill named "foo"
	fooSkillDir := filepath.Join(skillsDir, "foo")
	mustMkdirAll(t, fooSkillDir)
	mustWriteFile(t, filepath.Join(fooSkillDir, "SKILL.md"), `---
name: foo
description: Foo skill
---

Foo skill body.
`)

	// Agent also named "foo"
	mustMkdirAll(t, agentsDir)
	mustWriteFile(t, filepath.Join(agentsDir, "foo.md"), `---
name: foo
description: Foo agent
model: opus
---

Foo agent body.
`)

	items, warnings, err := discoverAllInDirs([]string{skillsDir}, []string{agentsDir})
	if err != nil {
		t.Fatalf("DiscoverAll error: %v", err)
	}

	if len(warnings) == 0 {
		t.Error("expected name collision warning")
	}

	if len(items) != 2 {
		t.Errorf("expected both items returned, got %d", len(items))
	}
}

func TestDiscoverSkills_BrokenSymlink(t *testing.T) {
	dir := t.TempDir()
	skillsDir := filepath.Join(dir, ".claude", "skills")
	mustMkdirAll(t, skillsDir)

	// Create a broken symlink
	brokenDir := filepath.Join(skillsDir, "broken")
	mustMkdirAll(t, brokenDir)
	if err := os.Symlink("/nonexistent/path/SKILL.md", filepath.Join(brokenDir, "SKILL.md")); err != nil {
		t.Skip("symlinks not supported on this platform")
	}

	// Also a valid skill
	validDir := filepath.Join(skillsDir, "valid")
	mustMkdirAll(t, validDir)
	mustWriteFile(t, filepath.Join(validDir, "SKILL.md"), `---
name: valid
description: Valid skill
---

Valid body.
`)

	items, err := discoverSkillsInDirs([]string{skillsDir})
	if err != nil {
		t.Fatalf("error: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item (broken skipped), got %d", len(items))
	}
	if items[0].Name != "valid" {
		t.Errorf("expected 'valid', got %q", items[0].Name)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
