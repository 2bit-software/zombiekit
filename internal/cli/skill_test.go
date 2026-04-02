package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func runSkillInstall(args []string) error {
	app := &cli.App{
		Commands: []*cli.Command{
			newSkillCommand(),
		},
	}
	return app.Run(append([]string{"brains"}, args...))
}

func TestSkillInstallLocal(t *testing.T) {
	dir := t.TempDir()

	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(orig); err != nil {
			t.Errorf("restoring working dir: %v", err)
		}
	})

	// profile "create-pr" is expected to exist in the global brains profiles.
	// If it doesn't exist in the test environment, skip rather than fail.
	err = runSkillInstall([]string{"skill", "install", "create-pr"})
	if err != nil && strings.Contains(err.Error(), "not found") {
		t.Skip("profile 'create-pr' not available in test environment")
	}
	if err != nil {
		t.Fatalf("skill install: %v", err)
	}

	skillPath := filepath.Join(dir, ".claude", "skills", "create-pr", "SKILL.md")
	if _, err := os.Stat(skillPath); err != nil {
		t.Errorf("expected SKILL.md at %s, got: %v", skillPath, err)
	}
}

func TestSkillInstallInvalidName(t *testing.T) {
	err := runSkillInstall([]string{"skill", "install", "../evil"})
	if err == nil {
		t.Error("expected error for invalid skill name, got nil")
	}
	if !strings.Contains(err.Error(), "invalid skill name") {
		t.Errorf("expected 'invalid skill name' in error, got: %v", err)
	}
}

func TestSkillInstallMissingArg(t *testing.T) {
	err := runSkillInstall([]string{"skill", "install"})
	if err == nil {
		t.Error("expected error when no profile name given, got nil")
	}
}

func TestSkillInstallUnknownProfile(t *testing.T) {
	err := runSkillInstall([]string{"skill", "install", "definitely-not-a-real-profile-xyzzy"})
	if err == nil {
		t.Error("expected error for unknown profile, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}
