package rules

import "strings"

// RuleMatch pairs a rule with the specific trigger that matched. For
// file-glob rules the trigger is empty; for command rules the trigger is
// the command prefix string that caused the match.
type RuleMatch struct {
	Rule    *Rule
	Trigger string
}

// segmentSeparators are operators that delimit distinct commands in a single
// Bash invocation. We split on these at the raw-string level — we do not
// parse quoting, subshells, or heredocs.
var segmentSeparators = []string{"&&", "||", ";", "|"}

// SplitSegments breaks a raw bash command string into the distinct commands
// a user stacked together with top-level operators (`&&`, `||`, `;`, `|`).
// It is intentionally naive: commands inside quotes, subshells, or `bash -c`
// strings are not parsed and will be split along with the rest.
func SplitSegments(cmd string) []string {
	const sentinel = "\x00"
	normalized := cmd
	for _, op := range segmentSeparators {
		normalized = strings.ReplaceAll(normalized, op, sentinel)
	}
	raw := strings.Split(normalized, sentinel)
	out := make([]string, 0, len(raw))
	for _, s := range raw {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// StripEnvPrefix removes any leading `VAR=value` assignments from a command
// segment so matchers see the real program invocation.
func StripEnvPrefix(segment string) string {
	fields := strings.Fields(segment)
	idx := 0
	for idx < len(fields) && isEnvAssignment(fields[idx]) {
		idx++
	}
	return strings.Join(fields[idx:], " ")
}

// isEnvAssignment reports whether a token looks like a shell env assignment
// (`NAME=value` where NAME is an identifier).
func isEnvAssignment(token string) bool {
	eq := strings.Index(token, "=")
	if eq <= 0 {
		return false
	}
	name := token[:eq]
	for _, r := range name {
		if r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
}

// MatchCommandPrefix reports whether segment starts with matcher on a
// whole-token boundary — the matcher must be followed by end-of-string or a
// space, so `go test` matches `go test ./...` but not `go testbench`.
func MatchCommandPrefix(segment, matcher string) bool {
	if matcher == "" {
		return false
	}
	return segment == matcher || strings.HasPrefix(segment, matcher+" ")
}

// MatchRulesByCommand returns rule/trigger pairs for every rule whose
// declared command triggers match any top-level segment of cmd. Iteration
// order is stable: rules in input order, and each rule contributes at most
// one pair keyed on the first command trigger that matches.
func MatchRulesByCommand(rules []*Rule, cmd string) []RuleMatch {
	segments := tokenizeSegments(cmd)
	if len(segments) == 0 {
		return nil
	}

	var out []RuleMatch
	for _, rule := range rules {
		if len(rule.Commands) == 0 {
			continue
		}
		if trigger := firstMatchingCommand(rule.Commands, segments); trigger != "" {
			out = append(out, RuleMatch{Rule: rule, Trigger: trigger})
		}
	}
	return out
}

// tokenizeSegments splits a command string and strips env prefixes so each
// returned segment starts with the real program name.
func tokenizeSegments(cmd string) []string {
	raw := SplitSegments(cmd)
	out := make([]string, 0, len(raw))
	for _, seg := range raw {
		cleaned := StripEnvPrefix(seg)
		if cleaned != "" {
			out = append(out, cleaned)
		}
	}
	return out
}

// firstMatchingCommand returns the first trigger in triggers that matches
// any of segments, preserving author-declared order.
func firstMatchingCommand(triggers, segments []string) string {
	for _, trigger := range triggers {
		for _, seg := range segments {
			if MatchCommandPrefix(seg, trigger) {
				return trigger
			}
		}
	}
	return ""
}
