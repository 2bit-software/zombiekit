package hook

import "sort"

// Formatter renders rule bodies for a specific editor's hook response.
// Implementations are stateless and safe for concurrent use.
type Formatter interface {
	FormatSessionStart(bodies []string) string
	FormatPreToolUse(bodies []string) string
	FormatPostToolUse(bodies []string) string
	FormatSessionEnd(bodies []string) string
}

// Editor describes an editor's full hook contract: the output formatter,
// the tool-name vocabulary for file operations, and the shell-tool
// identifier. Each supported editor registers one implementation via
// RegisterEditor during package init.
type Editor interface {
	Formatter
	// ExtractFilePaths returns the file paths referenced by a tool event
	// according to this editor's tool naming and tool_input schema.
	// Events whose ToolName is not a file-operation tool for this editor
	// return nil.
	ExtractFilePaths(event *HookEvent) []string
	// IsShellTool reports whether toolName is this editor's
	// shell-command execution tool.
	IsShellTool(toolName string) bool
}

// EditorSource describes how an editor was chosen for a hook invocation.
type EditorSource string

const (
	// EditorSourceFlag indicates the editor was selected via the --editor flag.
	EditorSourceFlag EditorSource = "flag"
	// EditorSourceEnv indicates the editor was inferred from an environment variable.
	EditorSourceEnv EditorSource = "env"
	// EditorSourceDefault indicates the editor is the compiled-in default.
	EditorSourceDefault EditorSource = "default"
)

var editors = map[Agent]Editor{}

// RegisterEditor adds an editor to the registry. It panics if id is
// already registered, since duplicate registration is a programmer error.
func RegisterEditor(id Agent, e Editor) {
	if _, exists := editors[id]; exists {
		panic("hook: editor already registered: " + string(id))
	}
	editors[id] = e
}

// LookupEditor returns the editor for id and reports whether it was found.
func LookupEditor(id Agent) (Editor, bool) {
	e, ok := editors[id]
	return e, ok
}

// KnownEditors returns the registered editor IDs in sorted order, for use in
// error messages that need to enumerate valid choices.
func KnownEditors() []string {
	ids := make([]string, 0, len(editors))
	for id := range editors {
		ids = append(ids, string(id))
	}
	sort.Strings(ids)
	return ids
}
