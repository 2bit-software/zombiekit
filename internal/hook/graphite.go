package hook

import (
	"os"
	"os/exec"
	"path/filepath"
)

// DetectGraphiteStatus checks graphite CLI availability, repo initialization,
// and current branch tracking status. Returns a human-readable status line
// for injection into the session start output.
func DetectGraphiteStatus(workDir string) string {
	if !isGraphiteAvailable() {
		return "graphite: not available"
	}

	if !isGraphiteInitialized(workDir) {
		return "graphite: available, not initialized"
	}

	if isGraphiteTracked(workDir) {
		return "graphite: available, initialized, stacked"
	}

	return "graphite: available, initialized"
}

func isGraphiteAvailable() bool {
	_, err := exec.LookPath("gt")
	return err == nil
}

func isGraphiteInitialized(workDir string) bool {
	info, err := os.Stat(filepath.Join(workDir, ".graphite"))
	return err == nil && info.IsDir()
}

func isGraphiteTracked(workDir string) bool {
	cmd := exec.Command("gt", "info", "--no-interactive")
	cmd.Dir = workDir
	return cmd.Run() == nil
}
