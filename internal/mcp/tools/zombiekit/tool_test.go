package zombiekit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTool(t *testing.T) {
	tool := NewTool()
	if tool == nil {
		t.Fatal("NewTool returned nil")
	}
}

func TestToolDefinition(t *testing.T) {
	tool := NewTool()
	def := tool.Definition()

	if def.Name != "feature" {
		t.Errorf("expected name 'feature', got '%s'", def.Name)
	}

	if def.Description == "" {
		t.Error("expected non-empty description")
	}

	if def.InputSchema == nil {
		t.Error("expected non-nil InputSchema")
	}

	// Verify InputSchema structure
	schemaType, ok := def.InputSchema["type"].(string)
	if !ok || schemaType != "object" {
		t.Error("expected InputSchema type to be 'object'")
	}

	required, ok := def.InputSchema["required"].([]string)
	if !ok || len(required) != 0 {
		t.Error("expected empty required array")
	}
}

func TestExecuteSuccess(t *testing.T) {
	// Set up a temporary home directory with the template file
	tempDir := t.TempDir()
	templateDir := filepath.Join(tempDir, ".brains", "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template dir: %v", err)
	}

	expectedContent := "# Step Feature Template\n\n## Test Content"
	templateFile := filepath.Join(templateDir, "step.feature.md")
	if err := os.WriteFile(templateFile, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	// Override HOME for the test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	tool := NewTool()
	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result != expectedContent {
		t.Errorf("expected content %q, got %q", expectedContent, result)
	}
}

func TestExecuteFileNotFound(t *testing.T) {
	// Set up a temporary home directory without the template file
	tempDir := t.TempDir()

	// Override HOME for the test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	tool := NewTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for missing file")
	}

	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' in error, got: %v", err)
	}

	// Verify the path is included in the error
	expectedPath := filepath.Join(tempDir, TemplateFilePath)
	if !strings.Contains(err.Error(), expectedPath) {
		t.Errorf("expected path '%s' in error, got: %v", expectedPath, err)
	}
}

func TestExecutePermissionDenied(t *testing.T) {
	// Skip on Windows as permission handling is different
	if os.Getenv("GOOS") == "windows" || filepath.Separator == '\\' {
		t.Skip("skipping permission test on Windows")
	}

	// Set up a temporary home directory with an unreadable file
	tempDir := t.TempDir()
	templateDir := filepath.Join(tempDir, ".brains", "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template dir: %v", err)
	}

	templateFile := filepath.Join(templateDir, "step.feature.md")
	if err := os.WriteFile(templateFile, []byte("content"), 0000); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}
	defer os.Chmod(templateFile, 0644) // Restore permissions for cleanup

	// Override HOME for the test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	tool := NewTool()
	_, err := tool.Execute(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for unreadable file")
	}

	if !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("expected 'permission denied' in error, got: %v", err)
	}
}

func TestExecuteEmptyFile(t *testing.T) {
	// Set up a temporary home directory with an empty template file
	tempDir := t.TempDir()
	templateDir := filepath.Join(tempDir, ".brains", "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template dir: %v", err)
	}

	templateFile := filepath.Join(templateDir, "step.feature.md")
	if err := os.WriteFile(templateFile, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	// Override HOME for the test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	tool := NewTool()
	result, err := tool.Execute(context.Background(), nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestExecuteWithArgs(t *testing.T) {
	// Test that Execute ignores any arguments passed
	tempDir := t.TempDir()
	templateDir := filepath.Join(tempDir, ".brains", "templates")
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		t.Fatalf("failed to create template dir: %v", err)
	}

	expectedContent := "test content"
	templateFile := filepath.Join(templateDir, "step.feature.md")
	if err := os.WriteFile(templateFile, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("failed to write template file: %v", err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", originalHome)

	tool := NewTool()

	// Test with various argument values (all should be ignored)
	testCases := []map[string]interface{}{
		nil,
		{},
		{"unknown": "value"},
		{"path": "override/attempt"},
	}

	for _, args := range testCases {
		result, err := tool.Execute(context.Background(), args)
		if err != nil {
			t.Errorf("Execute failed with args %v: %v", args, err)
			continue
		}
		if result != expectedContent {
			t.Errorf("expected content %q with args %v, got %q", expectedContent, args, result)
		}
	}
}
