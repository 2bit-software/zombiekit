package step

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitService_IsGitAvailable(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewGitService(tmpDir)

	// Git should be available in most development environments
	// This test might fail in environments without git
	if _, err := exec.LookPath("git"); err == nil {
		assert.True(t, svc.isGitAvailable())
	}
}

func TestGitService_IsGitRepository(t *testing.T) {
	t.Run("returns false for non-git directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc := NewGitService(tmpDir)

		assert.False(t, svc.isGitRepository())
	})

	t.Run("returns true for git repository", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Initialize git repository
		cmd := exec.Command("git", "init")
		cmd.Dir = tmpDir
		require.NoError(t, cmd.Run())

		svc := NewGitService(tmpDir)
		assert.True(t, svc.isGitRepository())
	})
}

func TestGitService_FormatBranchName(t *testing.T) {
	tmpDir := t.TempDir()
	svc := NewGitService(tmpDir)

	tests := []struct {
		name      string
		initType  string
		input     string
		expected  string
		shouldErr bool
	}{
		{"feature type", "feature", "user-auth", "feat/user-auth", false},
		{"bug type", "bug", "login-crash", "fix/login-crash", false},
		{"refactor type", "refactor", "extract-service", "ref/extract-service", false},
		{"with spaces", "feature", "User Auth", "feat/user-auth", false},
		{"uppercase", "feature", "MyFeature", "feat/myfeature", false},
		{"special chars", "feature", "feature_name!", "feat/feature-name", false},
		{"unknown type", "unknown", "name", "", true},
		{"empty name", "feature", "   ", "", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := svc.formatBranchName(tc.initType, tc.input)
			if tc.shouldErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestGitService_EnsureBranch_GracefulDegradation(t *testing.T) {
	t.Run("returns nil when not a git repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		svc := NewGitService(tmpDir)

		// Should not error - graceful degradation
		err := svc.EnsureBranch("feature", "test")
		assert.NoError(t, err)
	})
}

func TestGitService_EnsureBranch_CreatesBranch(t *testing.T) {
	tmpDir := t.TempDir()

	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Create initial commit so we can create branches
	dummyFile := filepath.Join(tmpDir, "dummy.txt")
	require.NoError(t, os.WriteFile(dummyFile, []byte("dummy"), 0644))

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	svc := NewGitService(tmpDir)

	// Create a feature branch
	err := svc.EnsureBranch("feature", "test-feature")
	require.NoError(t, err)

	// Verify branch exists
	assert.True(t, svc.branchExists("feat/test-feature"))
}

func TestGitService_EnsureBranch_SwitchesExistingBranch(t *testing.T) {
	tmpDir := t.TempDir()

	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	// Create initial commit
	dummyFile := filepath.Join(tmpDir, "dummy.txt")
	require.NoError(t, os.WriteFile(dummyFile, []byte("dummy"), 0644))

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	svc := NewGitService(tmpDir)

	// Create a branch
	err := svc.EnsureBranch("feature", "existing-feature")
	require.NoError(t, err)

	// Switch back to main
	cmd = exec.Command("git", "checkout", "master")
	cmd.Dir = tmpDir
	if cmd.Run() != nil {
		// Try main instead
		cmd = exec.Command("git", "checkout", "main")
		cmd.Dir = tmpDir
		cmd.Run()
	}

	// Should switch to existing branch without error
	err = svc.EnsureBranch("feature", "existing-feature")
	assert.NoError(t, err)
}
