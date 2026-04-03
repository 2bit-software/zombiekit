package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportSkill(t *testing.T) {
	srcDir := t.TempDir()
	destBase := t.TempDir()

	// Create source skill with supporting script
	skillDir := filepath.Join(srcDir, "my-skill")
	mustMkdirAll(t, filepath.Join(skillDir, "scripts"))
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), `---
name: my-skill
description: Does things
allowed-tools: Bash(*)
---

Skill body here.
`)
	mustWriteFile(t, filepath.Join(skillDir, "scripts", "run.sh"), "#!/bin/bash\necho hello")

	item := DiscoverableItem{
		Name:       "my-skill",
		Type:       "skill",
		SourcePath: filepath.Join(skillDir, "SKILL.md"),
	}

	err := importSkill(item, filepath.Join(destBase, "my-skill"))
	if err != nil {
		t.Fatalf("importSkill error: %v", err)
	}

	// Verify SKILL.md was written with transformed frontmatter
	content, err := os.ReadFile(filepath.Join(destBase, "my-skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("reading dest SKILL.md: %v", err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "allowed-tools") {
		t.Error("allowed-tools should have been stripped")
	}
	if !strings.Contains(contentStr, "my-skill") {
		t.Error("name should be preserved")
	}
	if !strings.Contains(contentStr, "Skill body here.") {
		t.Error("body should be preserved verbatim")
	}

	// Verify supporting files were copied
	scriptContent, err := os.ReadFile(filepath.Join(destBase, "my-skill", "scripts", "run.sh"))
	if err != nil {
		t.Fatalf("reading copied script: %v", err)
	}
	if string(scriptContent) != "#!/bin/bash\necho hello" {
		t.Errorf("script content mismatch: %q", string(scriptContent))
	}
}

func TestImportAgent(t *testing.T) {
	srcDir := t.TempDir()
	destBase := t.TempDir()

	mustWriteFile(t, filepath.Join(srcDir, "my-agent.md"), `---
name: my-agent
description: Agent description
model: opus
color: cyan
memory: user
---

Agent system prompt.
`)

	item := DiscoverableItem{
		Name:       "my-agent",
		Type:       "agent",
		SourcePath: filepath.Join(srcDir, "my-agent.md"),
	}

	err := importAgent(item, filepath.Join(destBase, "my-agent"))
	if err != nil {
		t.Fatalf("importAgent error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(destBase, "my-agent", "SKILL.md"))
	if err != nil {
		t.Fatalf("reading dest SKILL.md: %v", err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "model:") {
		t.Error("model should have been stripped")
	}
	if strings.Contains(contentStr, "color:") {
		t.Error("color should have been stripped")
	}
	if strings.Contains(contentStr, "memory:") {
		t.Error("memory should have been stripped")
	}
	if !strings.Contains(contentStr, "my-agent") {
		t.Error("name should be preserved")
	}
	if !strings.Contains(contentStr, "Agent system prompt.") {
		t.Error("body should be preserved")
	}
}

func TestImportAgent_WithSkillsComment(t *testing.T) {
	srcDir := t.TempDir()
	destBase := t.TempDir()

	mustWriteFile(t, filepath.Join(srcDir, "multi-agent.md"), `---
name: multi-agent
description: Uses skills
model: sonnet
skills: skill1,skill2
---

Agent body.
`)

	item := DiscoverableItem{
		Name:       "multi-agent",
		Type:       "agent",
		SourcePath: filepath.Join(srcDir, "multi-agent.md"),
	}

	err := importAgent(item, filepath.Join(destBase, "multi-agent"))
	if err != nil {
		t.Fatalf("importAgent error: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(destBase, "multi-agent", "SKILL.md"))
	if err != nil {
		t.Fatalf("reading dest SKILL.md: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "<!-- Referenced skills: skill1,skill2") {
		t.Error("expected HTML comment noting referenced skills")
	}
	if !strings.Contains(contentStr, "Agent body.") {
		t.Error("body should be preserved after comment")
	}
}

func TestImport_SkipsMissingItem(t *testing.T) {
	items := []DiscoverableItem{
		{Name: "exists", Type: "skill", SourcePath: "/nonexistent"},
	}

	opts := ImportOptions{
		Names:      []string{"does-not-exist"},
		Scope:      "local",
		WorkingDir: t.TempDir(),
	}

	result, err := Import(opts, items)
	if err != nil {
		t.Fatalf("Import error: %v", err)
	}

	if len(result.Skipped) != 1 {
		t.Fatalf("expected 1 skipped, got %d", len(result.Skipped))
	}
	if result.Skipped[0].Name != "does-not-exist" {
		t.Errorf("expected skipped name 'does-not-exist', got %q", result.Skipped[0].Name)
	}
}

func TestImport_CollisionError(t *testing.T) {
	srcDir := t.TempDir()
	destBase := t.TempDir()

	// Create source
	skillDir := filepath.Join(srcDir, "collide")
	mustMkdirAll(t, skillDir)
	mustWriteFile(t, filepath.Join(skillDir, "SKILL.md"), `---
name: collide
description: Test
---

Body.
`)

	// Pre-create destination to trigger collision
	mustMkdirAll(t, filepath.Join(destBase, ".brains", "profiles", "collide"))

	items := []DiscoverableItem{
		{Name: "collide", Type: "skill", SourcePath: filepath.Join(skillDir, "SKILL.md")},
	}

	opts := ImportOptions{
		Names:      []string{"collide"},
		Scope:      "local",
		WorkingDir: destBase,
	}

	_, err := Import(opts, items)
	if err == nil {
		t.Error("expected collision error")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestWriteAgentShim(t *testing.T) {
	srcDir := t.TempDir()

	agentPath := filepath.Join(srcDir, "my-agent.md")
	mustWriteFile(t, agentPath, `---
name: my-agent
description: Original agent
model: opus
color: cyan
skills: skill1
memory: user
---

Original agent body.
`)

	item := DiscoverableItem{
		Name:        "my-agent",
		Type:        "agent",
		Description: "Original agent",
		SourcePath:  agentPath,
	}

	shimPath, err := writeAgentShim(item)
	if err != nil {
		t.Fatalf("writeAgentShim error: %v", err)
	}

	if shimPath != agentPath {
		t.Errorf("expected shim at %s, got %s", agentPath, shimPath)
	}

	content, err := os.ReadFile(shimPath)
	if err != nil {
		t.Fatalf("reading shim: %v", err)
	}

	contentStr := string(content)
	// Should preserve original frontmatter fields
	if !strings.Contains(contentStr, "model:") {
		t.Error("shim should preserve model")
	}
	if !strings.Contains(contentStr, "color:") {
		t.Error("shim should preserve color")
	}
	// Should have allowed-tools
	if !strings.Contains(contentStr, "mcp__zombiekit__profile-compose") {
		t.Error("shim should have profile-compose reference")
	}
	// Should NOT have original body
	if strings.Contains(contentStr, "Original agent body.") {
		t.Error("shim should replace body, not keep original")
	}
	// Should have delegation body
	if !strings.Contains(contentStr, `profiles: ["my-agent"]`) {
		t.Error("shim should have delegation call")
	}
}

func TestCopyDirContents(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()

	mustWriteFile(t, filepath.Join(src, "SKILL.md"), "skip me")
	mustWriteFile(t, filepath.Join(src, "keep.txt"), "keep me")
	mustMkdirAll(t, filepath.Join(src, "scripts"))
	mustWriteFile(t, filepath.Join(src, "scripts", "run.sh"), "script content")

	err := copyDirContents(src, dst, []string{"SKILL.md"})
	if err != nil {
		t.Fatalf("copyDirContents error: %v", err)
	}

	// SKILL.md should be excluded
	if _, err := os.Stat(filepath.Join(dst, "SKILL.md")); err == nil {
		t.Error("SKILL.md should have been excluded")
	}

	// keep.txt should exist
	content, err := os.ReadFile(filepath.Join(dst, "keep.txt"))
	if err != nil {
		t.Fatal("keep.txt not copied")
	}
	if string(content) != "keep me" {
		t.Errorf("keep.txt content mismatch: %q", string(content))
	}

	// Nested script should exist
	content, err = os.ReadFile(filepath.Join(dst, "scripts", "run.sh"))
	if err != nil {
		t.Fatal("scripts/run.sh not copied")
	}
	if string(content) != "script content" {
		t.Errorf("script content mismatch: %q", string(content))
	}
}
