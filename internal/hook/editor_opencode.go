package hook

import (
	"encoding/json"
	"strings"
)

type opencodeFormatter struct{}

func init() {
	RegisterEditor(AgentOpenCode, opencodeFormatter{})
}

// FormatSessionStart emits the OpenCode envelope used for both
// session-inject (per-turn idempotent unconditional injection) and
// compact (post-compaction re-injection). The shim pushes
// additionalContext onto output.system or output.context respectively.
func (opencodeFormatter) FormatSessionStart(bodies []string) string {
	return marshalOpencodeEnvelope(bodies)
}

// FormatPreToolUse is a no-op for OpenCode; the shim does not subscribe
// to tool.execute.before in this iteration.
// TODO(opencode): implement once tool.execute.before is wired in brains.ts.
func (opencodeFormatter) FormatPreToolUse([]string) string { return "" }

// FormatPostToolUse emits the OpenCode envelope for file-edit rule
// injection. The shim appends additionalContext to the tool result
// string the model reads next.
func (opencodeFormatter) FormatPostToolUse(bodies []string) string {
	return marshalOpencodeEnvelope(bodies)
}

// FormatSessionEnd is a no-op; OpenCode has no session-end hook.
// TODO(opencode): if OpenCode adds a session lifecycle teardown hook,
// wire it here to clean up per-session state files.
func (opencodeFormatter) FormatSessionEnd([]string) string { return "" }

// ExtractFilePaths returns the file paths referenced by an OpenCode
// tool event. Recognizes OpenCode's native file-editing tool names
// (write, edit, multi-edit) passed through verbatim by the shim.
func (opencodeFormatter) ExtractFilePaths(event *HookEvent) []string {
	if event.ToolInput == nil {
		return nil
	}
	switch event.ToolName {
	case "write", "edit":
		if p := event.ToolInput.GetFilePath(); p != "" {
			return []string{p}
		}
	case "multi-edit":
		if p := event.ToolInput.GetFilePath(); p != "" {
			return []string{p}
		}
		for _, e := range event.ToolInput.Edits {
			if p := e.GetFilePath(); p != "" {
				return []string{p}
			}
		}
	}
	return nil
}

// IsShellTool reports whether toolName is OpenCode's shell execution tool.
func (opencodeFormatter) IsShellTool(toolName string) bool {
	return toolName == "bash"
}

func marshalOpencodeEnvelope(bodies []string) string {
	if len(bodies) == 0 {
		return "{}"
	}
	env := opencodeEnvelope{
		HookSpecificOutput: &opencodeHookOutput{
			AdditionalContext: strings.Join(bodies, "\n\n"),
		},
	}
	out, err := json.Marshal(env)
	if err != nil {
		return "{}"
	}
	return string(out)
}

type opencodeEnvelope struct {
	HookSpecificOutput *opencodeHookOutput `json:"hookSpecificOutput,omitempty"`
}

type opencodeHookOutput struct {
	AdditionalContext string `json:"additionalContext"`
}
