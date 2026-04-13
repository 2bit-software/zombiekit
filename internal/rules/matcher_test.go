package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMatchRules_GoPattern(t *testing.T) {
	rules := []*Rule{
		{FileName: "go.md", Paths: []string{"**/*.go"}, Body: "go rules"},
		{FileName: "py.md", Paths: []string{"**/*.py"}, Body: "python rules"},
	}

	matched := MatchRules(rules, "src/main.go")
	assert.Len(t, matched, 1)
	assert.Equal(t, "go.md", matched[0].FileName)
}

func TestMatchRules_BraceExpansion(t *testing.T) {
	rules := []*Rule{
		{FileName: "ts.md", Paths: []string{"**/*.{ts,tsx}"}, Body: "ts rules"},
	}

	assert.Len(t, MatchRules(rules, "src/App.tsx"), 1)
	assert.Len(t, MatchRules(rules, "src/index.ts"), 1)
	assert.Len(t, MatchRules(rules, "src/main.go"), 0)
}

func TestMatchRules_NoMatch(t *testing.T) {
	rules := []*Rule{
		{FileName: "go.md", Paths: []string{"**/*.go"}, Body: "go rules"},
	}

	assert.Empty(t, MatchRules(rules, "README.md"))
}

func TestMatchRules_SkipsUnconditional(t *testing.T) {
	rules := []*Rule{
		{FileName: "general.md", Body: "general rules"}, // no Paths = unconditional
		{FileName: "go.md", Paths: []string{"**/*.go"}, Body: "go rules"},
	}

	matched := MatchRules(rules, "main.go")
	assert.Len(t, matched, 1)
	assert.Equal(t, "go.md", matched[0].FileName)
}

func TestMatchRules_MultiplePatterns(t *testing.T) {
	rules := []*Rule{
		{FileName: "web.md", Paths: []string{"**/*.html", "**/*.css", "**/*.js"}, Body: "web rules"},
	}

	assert.Len(t, MatchRules(rules, "static/style.css"), 1)
	assert.Len(t, MatchRules(rules, "index.html"), 1)
	assert.Len(t, MatchRules(rules, "main.go"), 0)
}

func TestMatchRules_DirectoryPattern(t *testing.T) {
	rules := []*Rule{
		{FileName: "api.md", Paths: []string{"src/api/**"}, Body: "api rules"},
	}

	assert.Len(t, MatchRules(rules, "src/api/handler.go"), 1)
	assert.Len(t, MatchRules(rules, "src/cli/main.go"), 0)
}

func TestMatchRules_PathNormalization(t *testing.T) {
	rules := []*Rule{
		{FileName: "go.md", Paths: []string{"**/*.go"}, Body: "go rules"},
	}

	// Relative path
	assert.Len(t, MatchRules(rules, "internal/rules/service.go"), 1)

	// Absolute path
	assert.Len(t, MatchRules(rules, "/Users/morgan/Projects/personal/zombiekit/internal/rules/service.go"), 1)

	// Path with backslashes (Windows style, should be normalized)
	assert.Len(t, MatchRules(rules, "internal\\rules\\service.go"), 1)
}
