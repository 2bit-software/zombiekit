package profile

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// makeSkillZIP creates a .skill ZIP in dir with the given entries.
// entries is a map of relative path → content ("" for directory entries).
func makeSkillZIP(t *testing.T, dir, name string, entries map[string]string) string {
	t.Helper()
	path := filepath.Join(dir, name+".skill")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("creating zip: %v", err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for entryPath, content := range entries {
		fw, err := w.Create(entryPath)
		if err != nil {
			t.Fatalf("creating zip entry %s: %v", entryPath, err)
		}
		if _, err := fw.Write([]byte(content)); err != nil {
			t.Fatalf("writing zip entry %s: %v", entryPath, err)
		}
	}
	return path
}

func TestExtractSkillFile_NestedPrefix(t *testing.T) {
	dir := t.TempDir()
	makeSkillZIP(t, dir, "epic-planner", map[string]string{
		"epic-planner/SKILL.md":         "# Epic Planner",
		"epic-planner/script.sh":        "#!/bin/bash",
		"epic-planner/templates/out.md": "# Template",
	})

	targetDir := filepath.Join(dir, "epic-planner")
	if err := ExtractSkillFile(filepath.Join(dir, "epic-planner.skill"), targetDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, want := range []string{"SKILL.md", "script.sh", "templates/out.md"} {
		if _, err := os.Stat(filepath.Join(targetDir, want)); err != nil {
			t.Errorf("expected file %s to exist after extraction", want)
		}
	}
}

func TestExtractSkillFile_FlatLayout(t *testing.T) {
	dir := t.TempDir()
	makeSkillZIP(t, dir, "flat-skill", map[string]string{
		"SKILL.md":  "# Flat Skill",
		"helper.sh": "#!/bin/bash",
	})

	targetDir := filepath.Join(dir, "flat-skill")
	if err := ExtractSkillFile(filepath.Join(dir, "flat-skill.skill"), targetDir); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(targetDir, "SKILL.md")); err != nil {
		t.Error("expected SKILL.md after extraction")
	}
}

func TestExtractSkillFile_NoSKILLMD(t *testing.T) {
	dir := t.TempDir()
	makeSkillZIP(t, dir, "bad-skill", map[string]string{
		"bad-skill/README.md": "no skill here",
	})

	targetDir := filepath.Join(dir, "bad-skill")
	err := ExtractSkillFile(filepath.Join(dir, "bad-skill.skill"), targetDir)
	if err == nil {
		t.Fatal("expected error for ZIP with no SKILL.md")
	}
	if !strings.Contains(err.Error(), "SKILL.md") {
		t.Errorf("error should mention SKILL.md, got: %v", err)
	}

	// Extracted dir must be cleaned up on error.
	if _, statErr := os.Stat(targetDir); !os.IsNotExist(statErr) {
		t.Error("extracted directory should have been removed after SKILL.md validation failure")
	}
}

func TestExtractSkillFile_CorruptZIP(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "corrupt.skill")
	if err := os.WriteFile(path, []byte("this is not a zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := ExtractSkillFile(path, filepath.Join(dir, "corrupt"))
	if err == nil {
		t.Fatal("expected error for corrupt ZIP")
	}
}

func TestExtractPendingSkills_Idempotent(t *testing.T) {
	dir := t.TempDir()
	makeSkillZIP(t, dir, "my-skill", map[string]string{
		"my-skill/SKILL.md": "# My Skill",
	})

	// First call: extracts.
	errs := ExtractPendingSkills(dir)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors on first extract: %v", errs)
	}
	if _, err := os.Stat(filepath.Join(dir, "my-skill", "SKILL.md")); err != nil {
		t.Fatal("expected SKILL.md after first extract")
	}

	// Second call: directory exists, should be a no-op.
	errs = ExtractPendingSkills(dir)
	if len(errs) > 0 {
		t.Errorf("unexpected errors on second extract (should be no-op): %v", errs)
	}
}

func TestExtractPendingSkills_BatchContinuesOnError(t *testing.T) {
	dir := t.TempDir()

	// Good skill.
	makeSkillZIP(t, dir, "good-skill", map[string]string{
		"good-skill/SKILL.md": "# Good",
	})
	// Bad skill (no SKILL.md).
	makeSkillZIP(t, dir, "bad-skill", map[string]string{
		"bad-skill/README.md": "nope",
	})

	errs := ExtractPendingSkills(dir)
	if len(errs) == 0 {
		t.Fatal("expected at least one error for bad-skill")
	}

	// Good skill should still have been extracted.
	if _, err := os.Stat(filepath.Join(dir, "good-skill", "SKILL.md")); err != nil {
		t.Error("good-skill should have been extracted despite bad-skill error")
	}
}

func TestExtractSkillFile_ZipSlipRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "evil.skill")

	f, _ := os.Create(path)
	w := zip.NewWriter(f)
	fw, _ := w.Create("../../escape.txt")
	fw.Write([]byte("evil"))
	w.Close()
	f.Close()

	err := ExtractSkillFile(path, filepath.Join(dir, "evil"))
	if err == nil {
		t.Fatal("expected error for zip-slip path")
	}

	// The escaped file must not exist.
	if _, statErr := os.Stat(filepath.Join(filepath.Dir(dir), "escape.txt")); !os.IsNotExist(statErr) {
		t.Error("zip-slip file should not have been created")
	}
}

func TestDetectTopLevelPrefix(t *testing.T) {
	cases := []struct {
		name    string
		entries []string
		want    string
	}{
		{"nested common prefix", []string{"skill/SKILL.md", "skill/script.sh"}, "skill/"},
		{"flat", []string{"SKILL.md", "script.sh"}, ""},
		{"mixed top-levels", []string{"a/SKILL.md", "b/script.sh"}, ""},
		{"single file nested", []string{"s/SKILL.md"}, "s/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var files []*zip.File
			for _, e := range tc.entries {
				f := &zip.File{}
				f.Name = e
				files = append(files, f)
			}
			got := detectTopLevelPrefix(files)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
