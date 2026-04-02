package profile

import (
	"os"
	"path/filepath"
	"testing"
)

func makeSkillDir(t *testing.T, parentDir, skillName, skillMDContent string) string {
	t.Helper()
	dir := filepath.Join(parentDir, skillName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(skillMDContent), 0o644); err != nil {
		t.Fatalf("writing SKILL.md: %v", err)
	}
	return dir
}

func TestIsSkillDirectory(t *testing.T) {
	t.Run("directory with SKILL.md", func(t *testing.T) {
		dir := makeSkillDir(t, t.TempDir(), "my-skill", "# Skill")
		if !IsSkillDirectory(dir) {
			t.Error("expected IsSkillDirectory to return true")
		}
	})

	t.Run("directory without SKILL.md", func(t *testing.T) {
		dir := t.TempDir()
		if IsSkillDirectory(dir) {
			t.Error("expected IsSkillDirectory to return false for dir without SKILL.md")
		}
	})

	t.Run("nonexistent directory", func(t *testing.T) {
		if IsSkillDirectory("/does/not/exist/skill") {
			t.Error("expected IsSkillDirectory to return false for nonexistent path")
		}
	})
}

func TestLoadSkillProfile_WithFrontmatterName(t *testing.T) {
	dir := makeSkillDir(t, t.TempDir(), "my-skill", `---
name: custom-name
description: A test skill
---
Skill body here`)

	p, err := LoadSkillProfile(dir, SourceGlobal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Name != "custom-name" {
		t.Errorf("name: got %q, want %q", p.Name, "custom-name")
	}
	if p.Description != "A test skill" {
		t.Errorf("description: got %q, want %q", p.Description, "A test skill")
	}
	if p.Body != "Skill body here" {
		t.Errorf("body: got %q, want %q", p.Body, "Skill body here")
	}
	if !p.IsSkill {
		t.Error("expected IsSkill to be true")
	}
	if p.Source != SourceGlobal {
		t.Errorf("source: got %v, want %v", p.Source, SourceGlobal)
	}
	if filepath.Base(p.Path) != "SKILL.md" {
		t.Errorf("path should end in SKILL.md, got %q", p.Path)
	}
}

func TestLoadSkillProfile_NameDerivedFromDir(t *testing.T) {
	dir := makeSkillDir(t, t.TempDir(), "epic-planner", "# No frontmatter name\nBody here")

	p, err := LoadSkillProfile(dir, SourceLocal)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Name != "epic-planner" {
		t.Errorf("name: got %q, want %q", p.Name, "epic-planner")
	}
	if !p.IsSkill {
		t.Error("expected IsSkill to be true")
	}
}

func TestLoadSkillProfile_Nonexistent(t *testing.T) {
	_, err := LoadSkillProfile("/does/not/exist", SourceLocal)
	if err == nil {
		t.Error("expected error for nonexistent skill dir")
	}
}

func TestNormalizeSkillDirName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"epic-planner", "epic-planner"},
		{"My Cool Skill", "my-cool-skill"},
		{"Epic_Planner", "epic-planner"},
		{"UPPER CASE", "upper-case"},
		{"trailing-", "trailing"},
		{"-leading", "leading"},
		{"multi---hyphens", "multi-hyphens"},
		{"spec!@#ial", "special"},
	}

	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeSkillDirName(tc.input)
			if got != tc.want {
				t.Errorf("normalizeSkillDirName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestLoadSkillProfile_FlatProfileNotSkill(t *testing.T) {
	// A .md file is not a skill directory.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "profile.md"), []byte("body"), 0o644); err != nil {
		t.Fatal(err)
	}

	// The dir itself has no SKILL.md — IsSkillDirectory should return false.
	if IsSkillDirectory(dir) {
		t.Error("dir with only profile.md should not be a skill directory")
	}
}
