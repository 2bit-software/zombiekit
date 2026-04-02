package rules

import (
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

// MatchRules returns all rules whose path patterns match the given file path.
// Unconditional rules (no paths) are not included — use IsUnconditional for those.
func MatchRules(rules []*Rule, filePath string) []*Rule {
	normalized := filepath.ToSlash(filePath)

	var matched []*Rule
	for _, rule := range rules {
		if rule.IsUnconditional() {
			continue
		}
		if matchesAnyPattern(rule.Paths, normalized) {
			matched = append(matched, rule)
		}
	}
	return matched
}

func matchesAnyPattern(patterns []string, filePath string) bool {
	for _, pattern := range patterns {
		matched, err := doublestar.Match(pattern, filePath)
		if err == nil && matched {
			return true
		}
	}
	return false
}
