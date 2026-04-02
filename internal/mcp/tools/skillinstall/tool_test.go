package skillinstall

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteInvalidName(t *testing.T) {
	tool := NewTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"name":  "../evil",
		"scope": "local",
	})
	if err == nil {
		t.Error("expected error for invalid skill name, got nil")
	}
	if !strings.Contains(err.Error(), "invalid skill name") {
		t.Errorf("expected 'invalid skill name' in error, got: %v", err)
	}
}

func TestExecuteMissingName(t *testing.T) {
	tool := NewTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"scope": "local",
	})
	if err == nil {
		t.Error("expected error when name is missing, got nil")
	}
}

func TestExecuteMissingScope(t *testing.T) {
	tool := NewTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"name": "my-skill",
	})
	if err == nil {
		t.Error("expected error when scope is missing, got nil")
	}
}

func TestExecuteUnknownProfile(t *testing.T) {
	dir := t.TempDir()
	tool := NewTool()
	_, err := tool.Execute(context.Background(), map[string]any{
		"name":              "definitely-not-a-real-profile-xyzzy",
		"scope":             "local",
		"working_directory": dir,
	})
	if err == nil {
		t.Error("expected error for unknown profile, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestExecuteLocalInstall(t *testing.T) {
	dir := t.TempDir()
	tool := NewTool()

	result, err := tool.Execute(context.Background(), map[string]any{
		"name":              "create-pr",
		"scope":             "local",
		"working_directory": dir,
	})
	if err != nil && strings.Contains(err.Error(), "not found") {
		t.Skip("profile 'create-pr' not available in test environment")
	}
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "create-pr", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Errorf("expected SKILL.md at %s: %v", skillPath, err)
	}
	if !strings.Contains(result, skillPath) {
		t.Errorf("result %q does not contain path %q", result, skillPath)
	}
}

func TestExecuteIdempotent(t *testing.T) {
	dir := t.TempDir()
	tool := NewTool()
	args := map[string]any{
		"name":              "create-pr",
		"scope":             "local",
		"working_directory": dir,
	}

	_, err := tool.Execute(context.Background(), args)
	if err != nil && strings.Contains(err.Error(), "not found") {
		t.Skip("profile 'create-pr' not available in test environment")
	}
	if err != nil {
		t.Fatalf("first Execute: %v", err)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "create-pr", "SKILL.md")
	first, _ := os.ReadFile(skillPath)

	if _, err := tool.Execute(context.Background(), args); err != nil {
		t.Fatalf("second Execute: %v", err)
	}

	second, _ := os.ReadFile(skillPath)
	if string(first) != string(second) {
		t.Error("second install produced different content (not idempotent)")
	}
}
