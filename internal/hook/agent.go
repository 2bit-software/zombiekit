package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// DetectAgent determines which AI coding agent is running by checking
// environment variables. Checks CLAUDE_CODE_ENTRYPOINT first, then GEMINI_SESSION_ID.
func DetectAgent() Agent {
	if os.Getenv("CLAUDE_CODE_ENTRYPOINT") != "" {
		return AgentClaude
	}
	if os.Getenv("GEMINI_SESSION_ID") != "" {
		return AgentGemini
	}
	return AgentGemini // default to plain markdown output
}

// FormatOutput wraps the concatenated rules content in the appropriate format
// for the detected agent. Claude gets <system-reminder> tags; Gemini gets
// plain markdown. Used for SessionStart where plain stdout is injected.
func FormatOutput(agent Agent, bodies []string) string {
	if len(bodies) == 0 {
		return ""
	}

	content := strings.Join(bodies, "\n\n")

	switch agent {
	case AgentClaude:
		return fmt.Sprintf("<system-reminder>\n%s\n</system-reminder>", content)
	default:
		return content
	}
}

// hookResponse is the JSON envelope Claude Code expects from PreToolUse hooks.
type hookResponse struct {
	HookSpecificOutput hookSpecificOutput `json:"hookSpecificOutput"`
}

type hookSpecificOutput struct {
	HookEventName      string `json:"hookEventName"`
	PermissionDecision string `json:"permissionDecision"`
	AdditionalContext  string `json:"additionalContext,omitempty"`
}

// FormatPreToolOutput returns the rules as structured JSON for PreToolUse hooks.
// Claude Code requires this envelope to inject additionalContext before tool
// execution. For non-Claude agents, falls back to plain markdown.
func FormatPreToolOutput(agent Agent, bodies []string) string {
	if len(bodies) == 0 {
		return ""
	}

	content := strings.Join(bodies, "\n\n")

	if agent != AgentClaude {
		return content
	}

	resp := hookResponse{
		HookSpecificOutput: hookSpecificOutput{
			HookEventName:      "PreToolUse",
			PermissionDecision: "allow",
			AdditionalContext:  content,
		},
	}

	out, err := json.Marshal(resp)
	if err != nil {
		return content
	}
	return string(out)
}
