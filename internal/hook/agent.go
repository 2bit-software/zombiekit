package hook

import (
	"fmt"
	"os"
	"strings"
)

// DetectAgent determines which AI coding agent is running by checking
// environment variables. Checks CLAUDE_SESSION_ID first, then GEMINI_SESSION_ID.
func DetectAgent() Agent {
	if os.Getenv("CLAUDE_SESSION_ID") != "" {
		return AgentClaude
	}
	if os.Getenv("GEMINI_SESSION_ID") != "" {
		return AgentGemini
	}
	return AgentGemini // default to plain markdown output
}

// FormatOutput wraps the concatenated rules content in the appropriate format
// for the detected agent. Claude gets <system-reminder> tags; Gemini gets
// plain markdown.
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
