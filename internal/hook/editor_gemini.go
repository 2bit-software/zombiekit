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

// FormatSessionEnd is a no-op; Gemini CLI does not consume SessionEnd output.
func (geminiFormatter) FormatSessionEnd([]string) string { return "" }

func marshalGeminiEnvelope(bodies []string) string {
	var env geminiEnvelope
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
	HookSpecificOutput *geminiHookOutput `json:"hookSpecificOutput,omitempty"`
}

type geminiHookOutput struct {
	AdditionalContext string `json:"additionalContext"`
}
