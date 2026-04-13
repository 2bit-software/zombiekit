package hook

import (
	"encoding/json"
	"fmt"
	"strings"
)

type claudeFormatter struct{}

func init() {
	RegisterEditor(AgentClaude, claudeFormatter{})
}

// FormatSessionStart wraps the concatenated bodies in a <system-reminder>
// block, which Claude Code interprets as additional context to inject into
// the conversation. Empty bodies produce empty output.
func (claudeFormatter) FormatSessionStart(bodies []string) string {
	if len(bodies) == 0 {
		return ""
	}
	return fmt.Sprintf("<system-reminder>\n%s\n</system-reminder>", strings.Join(bodies, "\n\n"))
}

// FormatPreToolUse returns the JSON envelope Claude Code expects from
// PreToolUse hooks, carrying the rule bodies as additionalContext.
func (claudeFormatter) FormatPreToolUse(bodies []string) string {
	if len(bodies) == 0 {
		return ""
	}

	resp := claudeHookResponse{
		HookSpecificOutput: claudeHookSpecificOutput{
			HookEventName:      "PreToolUse",
			PermissionDecision: "allow",
			AdditionalContext:  strings.Join(bodies, "\n\n"),
		},
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return strings.Join(bodies, "\n\n")
	}
	return string(out)
}

// FormatSessionEnd is a no-op for Claude — the protocol does not consume
// output from SessionEnd hooks.
func (claudeFormatter) FormatSessionEnd([]string) string { return "" }

type claudeHookResponse struct {
	HookSpecificOutput claudeHookSpecificOutput `json:"hookSpecificOutput"`
}

type claudeHookSpecificOutput struct {
	HookEventName      string `json:"hookEventName"`
	PermissionDecision string `json:"permissionDecision"`
	AdditionalContext  string `json:"additionalContext,omitempty"`
}
