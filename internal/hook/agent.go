package hook

import (
	"fmt"
	"os"
	"strings"
)

// ResolveEditor determines which editor a hook invocation targets. If
// flagValue is non-empty it is validated against the editor registry and
// returned as the authoritative choice. Otherwise ResolveEditor falls back
// to environment detection (CLAUDE_CODE_ENTRYPOINT) and finally to the
// compiled-in default (Claude). The returned EditorSource records which
// path was taken so downstream audit consumers can distinguish explicit
// flag use from inferred defaults.
func ResolveEditor(flagValue string) (Agent, EditorSource, error) {
	if flagValue != "" {
		id := Agent(flagValue)
		if _, ok := LookupEditor(id); !ok {
			return "", "", fmt.Errorf("unknown editor: %s (valid: %s)", flagValue, strings.Join(KnownEditors(), ", "))
		}
		return id, EditorSourceFlag, nil
	}

	if os.Getenv("CLAUDE_CODE_ENTRYPOINT") != "" {
		return AgentClaude, EditorSourceEnv, nil
	}

	return AgentClaude, EditorSourceDefault, nil
}
