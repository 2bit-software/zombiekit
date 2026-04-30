package sandbox

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/2bit-software/zombiekit/internal/cmux"
)

// EnvSandboxName is the env key used to pass the sandbox name through the
// cmux CommandBuilder. The orchestrator sets this before calling SpawnSession;
// the CommandBuilder reads it and strips it from the env passed to the sandbox.
const EnvSandboxName = "_ZK_SANDBOX_NAME"

// NewCommandBuilder returns a cmux.CommandBuilder that wraps agent sessions
// in Docker Sandbox execution. The returned builder:
//
//  1. Reads the sandbox name from env[EnvSandboxName] (set by the caller).
//  2. Rewrites localhost callback URLs to cfg.CallbackHost.
//  3. Strips the sandbox name key from env so it is not leaked to the agent.
//  4. Builds an "sbx exec -it" command with env vars as -e flags.
//
// The worktreePath is used as the cmux cwd so that cmux displays the correct
// context, even though the actual work happens inside the sandbox.
func NewCommandBuilder(cfg Config) cmux.CommandBuilder {
	return func(worktreePath string, env map[string]string, baseCmd, prompt string) (cmd, cwd string, err error) {
		sandboxName := env[EnvSandboxName]
		if sandboxName == "" {
			return "", "", fmt.Errorf("env %s is required for sandbox command builder", EnvSandboxName)
		}

		// Filter out the sandbox name key and rewrite callback URLs.
		filtered := make(map[string]string, len(env))
		for k, v := range env {
			if k != EnvSandboxName {
				filtered[k] = v
			}
		}
		rewritten := RewriteCallbackHost(filtered, cfg.CallbackHost)

		cmd, err = buildSbxExecCommand(sandboxName, rewritten, baseCmd, prompt)
		if err != nil {
			return "", "", err
		}

		return cmd, worktreePath, nil
	}
}

var validEnvKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// buildSbxExecCommand constructs an "sbx exec -it" command string suitable for
// passing to cmux --command. The result is wrapped in bash -c "..." so it
// works regardless of cmux's shell (nushell, fish, etc.).
//
// Environment variables become -e flags (not bash exports), and the prompt is
// appended as a -p argument to the inner command.
//
// Example output:
//
//	bash -c "sbx exec -it -e WORK_CALLBACK_URL='http://host.docker.internal:8666/DEV-123' zk-dev-123 claude --dangerously-skip-permissions -p 'Read .ai/ticket.md'"
func buildSbxExecCommand(sandboxName string, env map[string]string, innerCmd, prompt string) (string, error) {
	var parts []string
	parts = append(parts, "sbx", "exec", "-it")

	// Env vars as -e flags, sorted for determinism.
	if len(env) > 0 {
		var keys []string
		for k := range env {
			if !validEnvKey.MatchString(k) {
				return "", fmt.Errorf("invalid env key: %q", k)
			}
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			parts = append(parts, "-e", k+"="+cmux.BashQuote(env[k]))
		}
	}

	parts = append(parts, sandboxName)

	// Inner command (e.g., "claude --dangerously-skip-permissions").
	parts = append(parts, strings.Fields(innerCmd)...)

	if prompt != "" {
		parts = append(parts, "-p", cmux.BashQuote(prompt))
	}

	// Wrap in bash -c "..." so the command works in any outer shell (nushell, fish).
	// Escape \, ", and $ in the inner string for the double-quote layer.
	inner := strings.Join(parts, " ")
	escaped := strings.NewReplacer(`\`, `\\`, `"`, `\"`, `$`, `\$`).Replace(inner)
	return `bash -c "` + escaped + `"`, nil
}
