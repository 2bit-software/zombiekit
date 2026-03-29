package cmux

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type workspaceEntry struct {
	ref      string
	name     string
	selected bool
}

// parseNewWorkspace extracts workspace ref from "OK workspace:N".
func parseNewWorkspace(stdout string) (string, error) {
	parts := strings.Fields(stdout)
	if len(parts) != 2 || parts[0] != "OK" || !strings.HasPrefix(parts[1], "workspace:") {
		return "", fmt.Errorf("unexpected new-workspace output: %q", stdout)
	}
	return parts[1], nil
}

// parseListWorkspaces parses cmux list-workspaces plain text output.
//
// Expected format per line:
//
//	[*] workspace:N  name  [selected]
//
// Returns error if non-empty input produces zero valid entries (format change detection).
func parseListWorkspaces(stdout string) ([]workspaceEntry, error) {
	var entries []workspaceEntry
	var nonEmptyLines int

	for _, line := range strings.Split(stdout, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		nonEmptyLines++

		selected := false
		if strings.HasPrefix(line, "* ") {
			selected = true
			line = line[2:]
		} else {
			line = strings.TrimPrefix(line, "  ")
		}

		parts := strings.SplitN(line, "  ", 2)
		if len(parts) < 2 {
			continue
		}

		ref := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		name = strings.TrimSuffix(name, "[selected]")
		name = strings.TrimSpace(name)

		entries = append(entries, workspaceEntry{
			ref:      ref,
			name:     name,
			selected: selected,
		})
	}

	if nonEmptyLines > 0 && len(entries) == 0 {
		return nil, fmt.Errorf(
			"failed to parse list-workspaces output (%d lines, 0 entries): format may have changed",
			nonEmptyLines,
		)
	}

	return entries, nil
}

// findByTicketID searches workspace entries for a name starting with "{ticketID}: ".
func findByTicketID(entries []workspaceEntry, ticketID string) *workspaceEntry {
	prefix := ticketID + ": "
	for i := range entries {
		if strings.HasPrefix(entries[i].name, prefix) {
			return &entries[i]
		}
	}
	return nil
}

var validEnvKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// buildCommand constructs the shell command string with exported env vars.
// Values are single-quote escaped. Empty env produces just the command.
func buildCommand(env map[string]string, cmd string) (string, error) {
	if len(env) == 0 {
		return cmd, nil
	}

	var exports []string
	for k, v := range env {
		if !validEnvKey.MatchString(k) {
			return "", newErrorf(ErrInvalidEnvKey, nil, "invalid env key: %q", k)
		}
		escaped := "'" + strings.ReplaceAll(v, "'", "'\\''") + "'"
		exports = append(exports, k+"="+escaped)
	}

	sort.Strings(exports)

	return "export " + strings.Join(exports, " ") + " && " + cmd, nil
}
