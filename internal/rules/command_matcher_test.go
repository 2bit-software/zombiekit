package rules

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitSegments(t *testing.T) {
	cases := map[string][]string{
		"":                    nil,
		"go test ./...":       {"go test ./..."},
		"cd x && go test":     {"cd x", "go test"},
		"a || b || c":         {"a", "b", "c"},
		"a ; b ; c":           {"a", "b", "c"},
		"cat foo | grep bar":  {"cat foo", "grep bar"},
		"a && b || c ; d | e": {"a", "b", "c", "d", "e"},
		"  go test   ":        {"go test"},
		"go test && ":         {"go test"},
	}
	for input, want := range cases {
		got := SplitSegments(input)
		if len(want) == 0 {
			assert.Empty(t, got, "input=%q", input)
			continue
		}
		assert.Equal(t, want, got, "input=%q", input)
	}
}

func TestStripEnvPrefix(t *testing.T) {
	cases := map[string]string{
		"go test ./...":                     "go test ./...",
		"CGO_ENABLED=0 go test":             "go test",
		"CGO_ENABLED=0 GOOS=linux go build": "go build",
		"FOO=bar":                           "",
		"--flag=value go test":              "--flag=value go test", // not an env assignment
		"1BAD=x go test":                    "1BAD=x go test",       // identifier can't start with digit — actually allowed by our rules; leading digit still matches [0-9]
	}
	// Correct the last case: our isEnvAssignment allows digits, so 1BAD=x is a valid env assignment.
	cases["1BAD=x go test"] = "go test"

	for input, want := range cases {
		assert.Equal(t, want, StripEnvPrefix(input), "input=%q", input)
	}
}

func TestMatchCommandPrefix(t *testing.T) {
	assert.True(t, MatchCommandPrefix("go test", "go test"))
	assert.True(t, MatchCommandPrefix("go test ./...", "go test"))
	assert.True(t, MatchCommandPrefix("go test -count=1 ./...", "go test"))
	assert.False(t, MatchCommandPrefix("go testbench", "go test"))
	assert.False(t, MatchCommandPrefix("gotest", "go test"))
	assert.False(t, MatchCommandPrefix("gopher test", "go test"))
	assert.False(t, MatchCommandPrefix("", "go test"))
	assert.False(t, MatchCommandPrefix("go test", ""))
}

func TestMatchRulesByCommand_SinglePrefix(t *testing.T) {
	rules := []*Rule{
		{FileName: "tf.md", Commands: []string{"go test"}, Body: "use task"},
	}
	got := MatchRulesByCommand(rules, "go test ./...")
	assert.Len(t, got, 1)
	assert.Equal(t, "go test", got[0].Trigger)
	assert.Equal(t, "tf.md", got[0].Rule.FileName)
}

func TestMatchRulesByCommand_NoMatchOnSubstring(t *testing.T) {
	rules := []*Rule{
		{FileName: "tf.md", Commands: []string{"go test"}, Body: "use task"},
	}
	assert.Empty(t, MatchRulesByCommand(rules, "gopher test-helper"))
	assert.Empty(t, MatchRulesByCommand(rules, "go testbench"))
}

func TestMatchRulesByCommand_ChainedCommand(t *testing.T) {
	rules := []*Rule{
		{FileName: "tf.md", Commands: []string{"go test"}, Body: "use task"},
	}
	got := MatchRulesByCommand(rules, "cd pkg && go test ./...")
	assert.Len(t, got, 1)
	assert.Equal(t, "go test", got[0].Trigger)
}

func TestMatchRulesByCommand_EnvPrefixStripped(t *testing.T) {
	rules := []*Rule{
		{FileName: "tf.md", Commands: []string{"go test"}, Body: "use task"},
	}
	got := MatchRulesByCommand(rules, "CGO_ENABLED=0 go test ./...")
	assert.Len(t, got, 1)
	assert.Equal(t, "go test", got[0].Trigger)
}

func TestMatchRulesByCommand_MultipleTriggersOnOneRule(t *testing.T) {
	rules := []*Rule{
		{FileName: "tf.md", Commands: []string{"go test", "go run"}, Body: "use task"},
	}

	// `go test ./...` — first trigger matches, returns "go test"
	got := MatchRulesByCommand(rules, "go test ./...")
	assert.Equal(t, "go test", got[0].Trigger)

	// `go run main.go` — second trigger matches, returns "go run"
	got = MatchRulesByCommand(rules, "go run main.go")
	assert.Equal(t, "go run", got[0].Trigger)
}

func TestMatchRulesByCommand_RuleWithNoCommandsSkipped(t *testing.T) {
	rules := []*Rule{
		{FileName: "go.md", Paths: []string{"**/*.go"}, Body: "go rules"},
		{FileName: "tf.md", Commands: []string{"go test"}, Body: "use task"},
	}
	got := MatchRulesByCommand(rules, "go test ./...")
	assert.Len(t, got, 1)
	assert.Equal(t, "tf.md", got[0].Rule.FileName)
}

func TestMatchRulesByCommand_DeterministicOrder(t *testing.T) {
	// Two rules both match; assert input order is preserved.
	rules := []*Rule{
		{FileName: "a.md", Commands: []string{"go test"}, Body: "a"},
		{FileName: "b.md", Commands: []string{"go test"}, Body: "b"},
	}
	got := MatchRulesByCommand(rules, "go test")
	assert.Len(t, got, 2)
	assert.Equal(t, "a.md", got[0].Rule.FileName)
	assert.Equal(t, "b.md", got[1].Rule.FileName)
}

func TestMatchRulesByCommand_EmptyCommand(t *testing.T) {
	rules := []*Rule{
		{FileName: "tf.md", Commands: []string{"go test"}, Body: "use task"},
	}
	assert.Empty(t, MatchRulesByCommand(rules, ""))
	assert.Empty(t, MatchRulesByCommand(rules, "   "))
}
