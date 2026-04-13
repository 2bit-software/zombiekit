# Rule Injection Verification: Ongoing Audit (Gemini CLI v0.37.1)

## Context
Verified that global and project-specific rules are correctly injected into the Gemini CLI context during file operations. The primary focus was ensuring that loading a `.go` file triggers the `PostToolUse` hook and provides rule-based guidance (e.g., from `go.md`).

## Verified Behavior
1.  **Workflow**: When the agent calls `read_file` on a `*.go` file, the `PostToolUse` hook fires.
2.  **Context Injection**: The `brains hook` command correctly extracts the file path from the `ToolResponse` (using the camelCase `filePath` sent by Gemini), matches it against patterns like `**/*.go`, and returns the rule body in `additionalContext`.
3.  **Deduplication**: Rules are injected exactly once per session. Subsequent reads of the same file (or other `.go` files) in the same session correctly skip injection to keep the context clean.
4.  **Path Compatibility**: The system handles both absolute and relative paths correctly. `doublestar.Match` works as expected for both types of paths as proven by `internal/rules/matcher_test.go`.
5.  **Trigger Dependency**: `PostToolUse` injection for Gemini CLI depends on `tool_response.success` being `true`. If `tool_response` is missing or `success` is `false`, the hook returns an empty decision.

## Instructions for Testing
To verify the injection in a live session:
1.  **Load a Go File**: Use `read_file` on any `.go` file (e.g., `internal/rules/service.go`).
2.  **Check Context**: After the tool execution, the agent should receive the contents of `go.md` (Go Development Standards) as part of the tool result's additional context.
3.  **Verify deduplication**: Try reading another `.go` file; the standards should not be repeated.

## Troubleshooting/Debug Info
- **Raw Input Log**: The current `bin/brains` is built with debug logic that saves the last raw hook JSON to `/tmp/zk-hook-input.json`.
- **Audit Log**: Hook executions are logged to `~/.zombiekit/logs/hooks.jsonl`.
- **Session State**: Injection state is tracked in `$TMPDIR/zk-session-<id>.json`.

## System Configuration
- **Built Logic**: The `internal/hook` package is fully updated to handle Gemini CLI's JSON schema (camelCase parameters) and extraction logic.
- **Global Settings**: This system assumes the user's global Gemini CLI settings are configured to execute the `brains hook` command for tool lifecycle events.
- **Rules Location**: Global rules are loaded from `~/.brains/rules/`, and project-specific rules from `.brains/rules/`.

## Current State
- `internal/rules/matcher_test.go` has been updated with `TestMatchRules_PathNormalization` to prove absolute path matching.
- Temporary session files and hook logs have been cleared.
- `internal/cli/hook.go` contains debug logic to capture raw stdin for inspection.
