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

// FormatPostToolUse is a no-op for Claude — the protocol typically handles
// rule injection during PreToolUse permissions.
func (claudeFormatter) FormatPostToolUse([]string) string { return "" }

// FormatSessionEnd is a no-op for Claude — the protocol does not consume
// output from SessionEnd hooks.
func (claudeFormatter) FormatSessionEnd([]string) string { return "" }

// ExtractFilePaths returns the file paths referenced by a Claude Code tool
// event. Recognizes the Read, Write, Edit, and MultiEdit tool names; other
// tool names (including any Gemini-style tool names) return nil.
func (claudeFormatter) ExtractFilePaths(event *HookEvent) []string {
	if event.ToolInput == nil {
		return nil
	}

	switch event.ToolName {
	case "Read":
		if path := event.ToolInput.GetFilePath(); path != "" {
			return []string{path}
		}
	case "Write", "Edit":
		if path := event.ToolInput.GetFilePath(); path != "" {
			return []string{path}
		}
		if event.ToolResponse != nil && event.ToolResponse.FilePath != "" {
			return []string{event.ToolResponse.FilePath}
		}
	case "MultiEdit":
		var paths []string
		for _, edit := range event.ToolInput.Edits {
			if path := edit.GetFilePath(); path != "" {
				paths = append(paths, path)
			}
		}
		return paths
	}

	return nil
}

// IsShellTool reports whether toolName is Claude Code's Bash tool.
func (claudeFormatter) IsShellTool(toolName string) bool {
	return toolName == "Bash"
}

type claudeHookResponse struct {
	HookSpecificOutput claudeHookSpecificOutput `json:"hookSpecificOutput"`
}

type claudeHookSpecificOutput struct {
	HookEventName      string `json:"hookEventName"`
	PermissionDecision string `json:"permissionDecision"`
	AdditionalContext  string `json:"additionalContext,omitempty"`
}
