package hook

import "sort"

// Formatter renders rule bodies for a specific editor's hook response.
// Implementations are stateless and safe for concurrent use.
type Formatter interface {
	FormatSessionStart(bodies []string) string
	FormatPreToolUse(bodies []string) string
	FormatSessionEnd(bodies []string) string
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

var editors = map[Agent]Formatter{}

// RegisterEditor adds a formatter to the editor registry. It panics if id is
// already registered, since duplicate registration is a programmer error.
func RegisterEditor(id Agent, f Formatter) {
	if _, exists := editors[id]; exists {
		panic("hook: editor already registered: " + string(id))
	}
	editors[id] = f
}

// LookupEditor returns the formatter for id and reports whether it was found.
func LookupEditor(id Agent) (Formatter, bool) {
	f, ok := editors[id]
	return f, ok
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
