package hook

import (
	"encoding/json"
	"strings"
)

type geminiFormatter struct{}

func init() {
	RegisterEditor(AgentGemini, geminiFormatter{})
}

// FormatSessionStart marshals the bodies into Gemini CLI's hook response
// envelope, placing the concatenated rule bodies at
// hookSpecificOutput.additionalContext. Empty bodies produce "{}" so stdout
// is always valid JSON, as Gemini requires.
func (geminiFormatter) FormatSessionStart(bodies []string) string {
	return marshalGeminiEnvelope(bodies)
}

// FormatPreToolUse uses the same envelope shape as FormatSessionStart.
// Gemini CLI's BeforeTool hook reads additionalContext identically.
func (geminiFormatter) FormatPreToolUse(bodies []string) string {
	return marshalGeminiEnvelope(bodies)
}

// FormatPostToolUse uses the same envelope shape as FormatSessionStart.
// Gemini CLI's AfterTool hook reads additionalContext to append to the tool
// result, making it the ideal spot for file-specific rule injection.
func (geminiFormatter) FormatPostToolUse(bodies []string) string {
	return marshalGeminiEnvelope(bodies)
}

// FormatSessionEnd is a no-op; Gemini CLI does not consume SessionEnd output.
func (geminiFormatter) FormatSessionEnd([]string) string { return "" }

// ExtractFilePaths returns the file paths referenced by a Gemini CLI tool
// event. Recognizes read_file, write_file, and replace — the three
// file-operation tools Gemini ships. Gemini has no multi-edit equivalent;
// multiple edits are serialized as N separate replace tool calls, so
// ExtractFilePaths returns at most one path per event.
func (geminiFormatter) ExtractFilePaths(event *HookEvent) []string {
	switch event.ToolName {
	case "read_file", "write_file", "replace":
		// Check ToolInput first
		if event.ToolInput != nil {
			if path := event.ToolInput.GetFilePath(); path != "" {
				return []string{path}
			}
		}
		// Check ToolResponse (especially for PostToolUse)
		if event.ToolResponse != nil && event.ToolResponse.FilePath != "" {
			return []string{event.ToolResponse.FilePath}
		}
	}

	return nil
}

// IsShellTool reports whether toolName is Gemini CLI's shell execution tool.
func (geminiFormatter) IsShellTool(toolName string) bool {
	return toolName == "run_shell_command"
}

func marshalGeminiEnvelope(bodies []string) string {
	var env geminiEnvelope
	env.Decision = "allow"
	if len(bodies) > 0 {
		env.HookSpecificOutput = &geminiHookOutput{
			AdditionalContext: strings.Join(bodies, "\n\n"),
		}
	}
	out, err := json.Marshal(env)
	if err != nil {
		return "{}"
	}
	return string(out)
}

type geminiEnvelope struct {
	Decision           string            `json:"decision"`
	HookSpecificOutput *geminiHookOutput `json:"hookSpecificOutput,omitempty"`
}

type geminiHookOutput struct {
	AdditionalContext string `json:"additionalContext"`
}
