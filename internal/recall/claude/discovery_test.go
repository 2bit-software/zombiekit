package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverHistoryFiles_FindsJSONL(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, "projects", "-Users-test-project")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("failed to create projects dir: %v", err)
	}

	// Create test .jsonl files
	files := []string{"session1.jsonl", "session2.jsonl"}
	for _, f := range files {
		path := filepath.Join(projectsDir, f)
		if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	// Discover files
	discovered, err := DiscoverHistoryFiles(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverHistoryFiles failed: %v", err)
	}

	if len(discovered) != 2 {
		t.Errorf("expected 2 files, got %d", len(discovered))
	}
}

func TestDiscoverHistoryFiles_IgnoresOtherFiles(t *testing.T) {
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, "projects", "-Users-test")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("failed to create projects dir: %v", err)
	}

	// Create various file types
	testFiles := []string{
		"session.jsonl",  // should be found
		"config.json",    // should be ignored
		"readme.txt",     // should be ignored
		"data.jsonl.bak", // should be ignored
	}
	for _, f := range testFiles {
		path := filepath.Join(projectsDir, f)
		if err := os.WriteFile(path, []byte(`{}`), 0644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}
	}

	discovered, err := DiscoverHistoryFiles(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverHistoryFiles failed: %v", err)
	}

	if len(discovered) != 1 {
		t.Errorf("expected 1 .jsonl file, got %d", len(discovered))
	}

	if len(discovered) > 0 && !contains(discovered, filepath.Join(projectsDir, "session.jsonl")) {
		t.Errorf("expected to find session.jsonl, got %v", discovered)
	}
}

func TestDiscoverHistoryFiles_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("failed to create projects dir: %v", err)
	}

	discovered, err := DiscoverHistoryFiles(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverHistoryFiles failed: %v", err)
	}

	if len(discovered) != 0 {
		t.Errorf("expected 0 files for empty dir, got %d", len(discovered))
	}
}

func TestDiscoverHistoryFiles_NoProjectsDir(t *testing.T) {
	tmpDir := t.TempDir()
	// Don't create projects directory

	discovered, err := DiscoverHistoryFiles(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverHistoryFiles should not fail when projects dir missing: %v", err)
	}

	if discovered != nil && len(discovered) != 0 {
		t.Errorf("expected empty/nil slice, got %v", discovered)
	}
}

func TestDiscoverProjectFiles_FiltersByProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two project directories
	project1 := filepath.Join(tmpDir, "projects", "-Users-alice-project1")
	project2 := filepath.Join(tmpDir, "projects", "-Users-bob-project2")
	if err := os.MkdirAll(project1, 0755); err != nil {
		t.Fatalf("failed to create project1 dir: %v", err)
	}
	if err := os.MkdirAll(project2, 0755); err != nil {
		t.Fatalf("failed to create project2 dir: %v", err)
	}

	// Create files in both
	if err := os.WriteFile(filepath.Join(project1, "sess1.jsonl"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project1, "sess2.jsonl"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project2, "other.jsonl"), []byte(`{}`), 0644); err != nil {
		t.Fatal(err)
	}

	// Filter to project1 only
	discovered, err := DiscoverProjectFiles(tmpDir, "/Users/alice/project1")
	if err != nil {
		t.Fatalf("DiscoverProjectFiles failed: %v", err)
	}

	if len(discovered) != 2 {
		t.Errorf("expected 2 files from project1, got %d", len(discovered))
	}
}

func TestDiscoverProjectFiles_NonexistentProject(t *testing.T) {
	tmpDir := t.TempDir()
	projectsDir := filepath.Join(tmpDir, "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatalf("failed to create projects dir: %v", err)
	}

	discovered, err := DiscoverProjectFiles(tmpDir, "/Users/nobody/missing")
	if err != nil {
		t.Fatalf("DiscoverProjectFiles should not fail for missing project: %v", err)
	}

	if discovered != nil && len(discovered) != 0 {
		t.Errorf("expected empty slice for nonexistent project, got %v", discovered)
	}
}

func TestEncodeProjectPath_Basic(t *testing.T) {
	result := EncodeProjectPath("/Users/foo/bar")
	expected := "-Users-foo-bar"

	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestEncodeProjectPath_RootPath(t *testing.T) {
	result := EncodeProjectPath("/")
	expected := "-"

	if result != expected {
		t.Errorf("expected %q for root path, got %q", expected, result)
	}
}

func TestEncodeProjectPath_DeepPath(t *testing.T) {
	result := EncodeProjectPath("/Users/morgan/Projects/personal/zombiekit")
	expected := "-Users-morgan-Projects-personal-zombiekit"

	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestDefaultClaudePath_ExpandsTilde(t *testing.T) {
	result := DefaultClaudePath()

	// Should not contain tilde
	if len(result) > 0 && result[0] == '~' {
		t.Errorf("DefaultClaudePath should expand tilde, got %q", result)
	}

	// Should end with .claude
	if !filepath.IsAbs(result) || !endsWith(result, ".claude") {
		t.Errorf("expected absolute path ending with .claude, got %q", result)
	}
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func endsWith(path, suffix string) bool {
	return filepath.Base(path) == suffix
}
