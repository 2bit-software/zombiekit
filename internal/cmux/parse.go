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

// bashQuote wraps a string in bash single quotes, escaping embedded single
// quotes with the '\'' idiom.
func bashQuote(s string) string {
	escaped := strings.ReplaceAll(s, `'`, `'\''`)
	return `'` + escaped + `'`
}

// buildCommand constructs a bash -c command string with exported env vars and
// an optional prompt appended as a positional argument.
//
// The inner command uses bash single-quote escaping for values. The outer layer
// uses double-quote escaping so the command string is valid in any outer shell
// (bash, zsh, nushell, fish). The '\" single-quote idiom is bash-specific and
// breaks in nushell; double-quote wrapping is portable.
func buildCommand(env map[string]string, cmd, prompt string) (string, error) {
	effective := cmd
	if len(prompt) > 0 {
		effective = cmd + ` ` + bashQuote(prompt)
	}

	if len(env) == 0 {
		return effective, nil
	}

	var exports []string
	for k, v := range env {
		if !validEnvKey.MatchString(k) {
			return "", newErrorf(ErrInvalidEnvKey, nil, "invalid env key: %q", k)
		}
		exports = append(exports, k+"="+bashQuote(v))
	}

	sort.Strings(exports)

	inner := "export " + strings.Join(exports, " ") + " && " + effective
	outer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`).Replace(inner)
	return `bash -c "` + outer + `"`, nil
}
