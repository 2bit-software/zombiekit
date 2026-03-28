package worktree

import "strings"

// sanitizeTitle converts a human-readable title into a git-safe branch suffix.
//
// Rules applied in order:
//  1. Lowercase
//  2. Spaces to hyphens
//  3. Strip non-ASCII and non-alphanumeric (keep hyphens, underscores)
//  4. Collapse consecutive hyphens
//  5. Trim leading/trailing hyphens
//  6. Truncate to 40 characters
//  7. Trim trailing hyphens after truncation
//  8. Fallback to "untitled" if empty
func sanitizeTitle(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")

	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	s = b.String()

	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	s = strings.Trim(s, "-")

	if len(s) > 40 {
		s = s[:40]
	}
	s = strings.TrimRight(s, "-")

	if s == "" {
		return "untitled"
	}
	return s
}
