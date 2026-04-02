package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	valid := []string{"a", "my-skill", "abc-def-123", "x1", "foo"}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("ValidateName(%q) returned unexpected error: %v", name, err)
		}
	}

	invalid := []string{"", "../evil", "./bad", "Bad", "my_skill", "-lead", "trail-", "A", "has space", "dot.dot"}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Errorf("ValidateName(%q) should have returned an error", name)
		}
	}
}

func TestGenerateContent(t *testing.T) {
	t.Run("uses provided description", func(t *testing.T) {
		content := GenerateContent("my-skill", "Does something useful.")
		if !strings.Contains(content, "name: my-skill") {
			t.Error("missing name field")
		}
		if !strings.Contains(content, "Does something useful.") {
			t.Error("missing description")
		}
		if !strings.Contains(content, `profiles: ["my-skill"]`) {
			t.Error("missing profile-compose call with correct profile name")
		}
		if !strings.Contains(content, "allowed-tools: mcp__zombiekit__profile-compose") {
			t.Error("missing allowed-tools field")
		}
	})

	t.Run("uses fallback when description is empty", func(t *testing.T) {
		content := GenerateContent("my-skill", "")
		if !strings.Contains(content, "Delegates to the my-skill profile via profile-compose.") {
			t.Error("missing fallback description")
		}
	})

	t.Run("exact format: frontmatter then blank line then body", func(t *testing.T) {
		content := GenerateContent("test", "A test skill.")
		expected := "---\nname: test\ndescription: >\n  A test skill.\nallowed-tools: mcp__zombiekit__profile-compose\n---\n\nCall `mcp__zombiekit__profile-compose` with `profiles: [\"test\"]` and follow the returned instructions exactly.\n"
		if content != expected {
			t.Errorf("content does not match exact template.\ngot:\n%s\nwant:\n%s", content, expected)
		}
	})
}

func TestWriteSkill(t *testing.T) {
	t.Run("creates dirs and writes file", func(t *testing.T) {
		dir := t.TempDir()
		content := "test content"
		path, err := WriteSkill(dir, "my-skill", content)
		if err != nil {
			t.Fatalf("WriteSkill returned error: %v", err)
		}
		expected := filepath.Join(dir, "my-skill", "SKILL.md")
		if path != expected {
			t.Errorf("path = %q, want %q", path, expected)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading written file: %v", err)
		}
		if string(got) != content {
			t.Errorf("file content = %q, want %q", string(got), content)
		}
	})

	t.Run("idempotent overwrite", func(t *testing.T) {
		dir := t.TempDir()
		if _, err := WriteSkill(dir, "my-skill", "first"); err != nil {
			t.Fatalf("first write: %v", err)
		}
		if _, err := WriteSkill(dir, "my-skill", "second"); err != nil {
			t.Fatalf("second write: %v", err)
		}
		got, _ := os.ReadFile(filepath.Join(dir, "my-skill", "SKILL.md"))
		if string(got) != "second" {
			t.Errorf("expected second write to overwrite, got %q", string(got))
		}
	})

	t.Run("leaves other files untouched", func(t *testing.T) {
		dir := t.TempDir()
		skillDir := filepath.Join(dir, "my-skill")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatal(err)
		}
		otherPath := filepath.Join(skillDir, "other.txt")
		if err := os.WriteFile(otherPath, []byte("keep me"), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := WriteSkill(dir, "my-skill", "new content"); err != nil {
			t.Fatalf("WriteSkill: %v", err)
		}
		got, err := os.ReadFile(otherPath)
		if err != nil {
			t.Fatalf("other.txt was deleted: %v", err)
		}
		if string(got) != "keep me" {
			t.Errorf("other.txt was modified, got %q", string(got))
		}
	})

	t.Run("errors when name exists as plain file", func(t *testing.T) {
		dir := t.TempDir()
		// Create a plain file where the skill dir would go
		if err := os.WriteFile(filepath.Join(dir, "my-skill"), []byte("i am a file"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := WriteSkill(dir, "my-skill", "content")
		if err == nil {
			t.Error("expected error when name exists as plain file, got nil")
		}
	})
}
