# Research Summary

## OpenCode hook model

Source: `anomalyco/opencode@dev` (formerly `sst/opencode`; old repo 301-redirects).

OpenCode does not expose subprocess hooks. Its "hooks" are in-process JS/TS
functions inside a plugin module, loaded via dynamic `import()` at
`packages/opencode/src/plugin/loader.ts:102`. Plugins run in the OpenCode Bun
process; they receive live JS objects and mutate `output` in place. There is no
JSON wire format and no mechanism to register an external binary.

### Relevant hooks (`packages/plugin/src/index.ts:222-315`)

| Hook | Input | Mutable output | Use |
|------|-------|----------------|-----|
| `tool.execute.after` | `{tool, sessionID, callID, args}` | `{title, output, metadata}` | Append rule text to the tool result string the model will read next (closest analog to Gemini PostToolUse) |
| `tool.execute.before` | `{tool, sessionID, callID}` | `{args}` | Intercept tool call, cannot inject context directly |
| `experimental.chat.system.transform` | `{sessionID?, model}` | `{system: string[]}` | Push strings onto the system prompt (closest analog to Claude SessionStart) |
| `chat.message` | `{sessionID, agent?, model?, ...}` | `{message, parts}` | Mutate incoming user message parts |
| `event` (pub/sub) | `{event: Event}` | — | Fire-and-forget notifications, cannot block |

### Dispatch

`packages/opencode/src/plugin/index.ts:264-277` — trigger dispatch iterates
registered hook objects sequentially and awaits each. Return values are
discarded; only mutations to `output` take effect. `OPENCODE_PURE=1` disables
all external plugins.

### File path location

Unlike Claude/Gemini, the edited file path is not top-level on the event. It
lives inside `output.args` with a tool-dependent field name (`filePath` for
`write` / `edit` / `multi-edit`). Any bridge must switch on `input.tool` and
normalize before handing off to `brains`.

### Config

Plugins are registered in `opencode.json` via a `"plugin"` array of
module specs, or auto-loaded from `.opencode/plugins/*.ts`. Loader order:
global config → project config → `~/.config/opencode/plugins/` → `.opencode/plugins/`.

## zombiekit hook architecture (current)

Entry: `internal/cli/hook.go:36`. `--editor` flag selects an `Editor`
implementation from the registry (`internal/hook/editors.go:17-28`):

```go
type Editor interface {
    Formatter
    ExtractFilePaths(event *HookEvent) []string
    IsShellTool(toolName string) bool
}
```

Implementations register via `RegisterEditor(id, editor)` in `init()`. Handler
(`internal/hook/handler.go`) is editor-agnostic: it calls `ExtractFilePaths`,
resolves rules via `rules.ResolveForFiles`, deduplicates against session state,
and hands bodies to the editor's `Format*()` method.

Existing editors:

- `editor_claude.go` — Claude Code: `PreToolUse` JSON with
  `hookSpecificOutput.additionalContext` + `permissionDecision`; SessionStart
  plain `<system-reminder>` block; PostToolUse no-op.
- `editor_gemini.go` — Gemini CLI: uniform JSON envelope
  `{decision, hookSpecificOutput.additionalContext}` across SessionStart,
  PreToolUse, and PostToolUse; SessionEnd no-op.

Rule matching and dedup are shared in `internal/hook/handler.go` and
`internal/hook/session.go`; new editors do not touch them.

## Integration shape

Because OpenCode cannot spawn `brains` directly, the integration has two
pieces:

1. **`brains` side** — a new `opencode` editor that reads a normalized JSON
   payload on stdin (produced by the shim) and emits a JSON response the shim
   can consume. The payload must already carry the file path at a known field
   so `ExtractFilePaths` does not need to know about OpenCode tool-arg shapes.

2. **Shim side** — a single `.ts` plugin file the user drops into
   `.opencode/plugins/`. It:
   - Subscribes to `tool.execute.after` (and any other events we decide on).
   - Normalizes `input.tool` + `output.args` into a `{tool, file_path}` (or
     similar) JSON object.
   - Spawns `brains hook --editor opencode --event post-tool-use` via
     `Bun.spawn`, writes the payload to stdin, reads stdout.
   - Parses `{additionalContext}` (or whatever shape we land on) from stdout
     and mutates `output.output += ...` so the model sees it as part of the
     tool result.

Distribution: raw `.ts` script for now. An `install` subcommand can be added
later.
