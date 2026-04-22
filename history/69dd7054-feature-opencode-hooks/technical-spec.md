# Technical Spec: OpenCode Hook Support

## Architecture

Two artifacts ship in this feature:

1. **`brains` Go changes** — a new `opencode` editor registered with the
   existing hook registry, plus a minimal handler tweak so a single
   `SessionStart` code path can serve both "full reset" (compact, resume,
   startup) and "idempotent inject" (per-turn) semantics.
2. **`brains.ts` shim** — a Bun/Node-compatible OpenCode plugin that
   subscribes to three hooks, normalizes their inputs into zombiekit's
   existing `HookEvent` JSON shape, spawns `brains hook --editor opencode`,
   and mutates OpenCode's hook output with whatever `additionalContext`
   `brains` returns. Committed to the repo at
   `embed/integrations/opencode/brains.ts` and embedded at build time.

The shared rule matcher, session dedup, audit sink, and CLI entrypoint are
reused unchanged. The handler gains one new branch (`Source == "inject"`
skips dedup reset). Session state gains one new field
(`GraphiteInjected bool`). Nothing else in the core changes.

## Event Mapping

Canonical `--event` values on the CLI side. OpenCode uses three of them:

| Shim hook | `--event` | Canonical HookEventName | Source | Handler behavior |
|-----------|-----------|-------------------------|--------|------------------|
| `tool.execute.after` (file ops) | `post-tool-use` | `PostToolUse` | — | Existing `handlePostToolUse` — dedup-gated file-glob injection |
| `experimental.chat.system.transform` | `session-inject` | `SessionStart` | `inject` | `handleSessionStart` branch: **skip** `ResetInjectedRules`; inject unconditional rules with per-session dedup |
| `experimental.session.compacting` | `compact` | `SessionStart` | `compact` | Existing path: `ResetInjectedRules`, re-inject unconditional, return bodies for the shim to push onto `output.context` |

`pre-tool-use` and `session-end` are not wired for OpenCode in this feature
(pre-tool-use is explicitly out of scope; session-end has no OpenCode hook
equivalent — state files eventually age out of `/tmp`).

### Handler change

In `internal/hook/handler.go::handleSessionStart`, change the top of the
function from:

```go
state := LoadState(event.SessionID, h.agent)
ResetInjectedRules(state)
if event.Source != "compact" { state.CompactionCount-- ... }
```

to:

```go
state := LoadState(event.SessionID, h.agent)
if event.Source != "inject" {
    ResetInjectedRules(state)
    if event.Source != "compact" { state.CompactionCount-- ... }
}
```

Everything below (unconditional rule resolution, dedup-gated marking,
graphite append, save) stays. The dedup-gated loop already handles the
`inject` case correctly: on the first `session-inject` call of a session,
no rules are marked, so all unconditional rules fire; on the second call,
they're all marked, so nothing fires.

### Graphite dedup

Currently `handleSessionStart` appends `DetectGraphiteStatus(cwd)` to
`bodies` on every call. Under `session-inject` this would re-append on
every turn, defeating the dedup. Add a `GraphiteInjected bool` field to
`rules.SessionState` (JSON key `graphite_injected`), and guard the append:

```go
if !state.GraphiteInjected {
    if g := DetectGraphiteStatus(event.CWD); g != "" {
        bodies = append(bodies, g)
        state.GraphiteInjected = true
    }
}
```

On reset (`Source != "inject"`), the field is cleared along with the map
inside a revised `ResetInjectedRules`:

```go
func ResetInjectedRules(state *rules.SessionState) {
    state.InjectedRules = make(map[string]time.Time)
    state.CompactionCount++
    state.GraphiteInjected = false
}
```

Behavioral impact on Claude/Gemini: graphite status now fires once per
session-reset instead of once per SessionStart call. In practice
SessionStart fires once per reset anyway (Claude and Gemini each call
it at startup/resume/compact only), so this is a no-op for them.

## CLI Changes (`internal/cli/hook.go`)

Extend the `--event` flag usage and switch statement:

```go
Usage: "Hook event type: session-start, session-inject, pre-tool-use, post-tool-use, session-end, compact",
```

```go
switch eventType {
case "session-start":
    event.HookEventName = "SessionStart"
case "session-inject":
    event.HookEventName = "SessionStart"
    event.Source = "inject"
case "pre-tool-use":
    event.HookEventName = "PreToolUse"
case "post-tool-use":
    event.HookEventName = "PostToolUse"
case "session-end":
    event.HookEventName = "SessionEnd"
case "compact":
    event.HookEventName = "SessionStart"
    event.Source = "compact"
default:
    return fmt.Errorf("unknown event type: %s", eventType)
}
```

`session-inject` and `compact` forcibly set `event.Source`, overriding
whatever the shim wrote on the wire. The shim isn't expected to populate
`source` at all for these two events — the CLI flag is authoritative.

## `internal/hook/types.go`

Add `AgentOpenCode`:

```go
const (
    AgentClaude   Agent = "claude"
    AgentGemini   Agent = "gemini"
    AgentOpenCode Agent = "opencode"
)
```

## `internal/hook/editor_opencode.go` (new)

```go
package hook

import (
    "encoding/json"
    "strings"
)

type opencodeFormatter struct{}

func init() {
    RegisterEditor(AgentOpenCode, opencodeFormatter{})
}

// All four Format* methods emit the same envelope shape the shim parses.
// SessionStart covers both session-inject and compact (which the CLI
// rewrites to SessionStart). PostToolUse covers file-edit injection.
// PreToolUse and SessionEnd are no-ops for OpenCode.

func (opencodeFormatter) FormatSessionStart(bodies []string) string {
    return marshalOpencodeEnvelope(bodies)
}
func (opencodeFormatter) FormatPreToolUse([]string) string     { return "" }
func (opencodeFormatter) FormatPostToolUse(bodies []string) string {
    return marshalOpencodeEnvelope(bodies)
}
func (opencodeFormatter) FormatSessionEnd([]string) string     { return "" }

// ExtractFilePaths recognizes OpenCode's native file-editing tool names.
// The shim passes input.tool through verbatim; no name translation.
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
        // OpenCode's multi-edit applies many edits to a single file, so
        // ExtractFilePaths returns at most one path. If the shim forwards
        // filePath on ToolInput, use it; otherwise walk edits.
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

// IsShellTool reports whether toolName is OpenCode's shell tool.
// Shell-command rule triggers are out of scope for this iteration, so
// the handler's Bash path will not be exercised — but return the correct
// tool name for forward compatibility.
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
```

Empty bodies produce `{}` (same invariant as Gemini — always valid JSON,
shim can always `JSON.parse` stdout). The shim detects "no rules" by
checking whether `hookSpecificOutput` is present.

## The Shim: `embed/integrations/opencode/brains.ts`

Single TypeScript file, Bun-compatible. Structure:

```ts
import type { Plugin } from "@opencode-ai/plugin"

const BRAINS_BIN = process.env.BRAINS_BIN ?? "brains"

type Envelope = {
  hookSpecificOutput?: { additionalContext?: string }
}

async function callBrains(
  event: string,
  payload: Record<string, unknown>,
): Promise<string> {
  try {
    const proc = Bun.spawn([BRAINS_BIN, "hook", "--editor", "opencode", "--event", event], {
      stdin: "pipe",
      stdout: "pipe",
      stderr: "inherit",
    })
    proc.stdin.write(JSON.stringify(payload))
    proc.stdin.end()
    const out = await new Response(proc.stdout).text()
    const exit = await proc.exited
    if (exit !== 0 || !out.trim()) return ""
    const env = JSON.parse(out) as Envelope
    return env.hookSpecificOutput?.additionalContext ?? ""
  } catch (err) {
    console.error(`[brains/opencode] ${event} failed:`, err)
    return ""
  }
}

function extractFilePath(tool: string, args: any): string | undefined {
  if (tool === "write" || tool === "edit" || tool === "multi-edit") {
    return args?.filePath ?? args?.file_path
  }
  return undefined
}

export const server: Plugin = async ({ directory }) => ({
  "experimental.chat.system.transform": async (input, output) => {
    const ctx = await callBrains("session-inject", {
      session_id: input.sessionID,
      hook_event_name: "SessionStart",
      cwd: directory,
    })
    if (ctx) output.system.push(ctx) // append-only, never touch index 0
  },

  "experimental.session.compacting": async (input, output) => {
    const ctx = await callBrains("compact", {
      session_id: input.sessionID,
      hook_event_name: "SessionStart",
      cwd: directory,
    })
    if (ctx) output.context.push(ctx)
  },

  "tool.execute.after": async (input, output) => {
    const filePath = extractFilePath(input.tool, output.args)
    if (!filePath) return
    const ctx = await callBrains("post-tool-use", {
      session_id: input.sessionID,
      hook_event_name: "PostToolUse",
      cwd: directory,
      tool_name: input.tool,
      tool_input: { file_path: filePath },
    })
    if (ctx) output.output = `${output.output}\n\n${ctx}`
  },
})
```

Key points:

- **Binary path** from `BRAINS_BIN`, default `brains`. User sets
  `BRAINS_BIN=brains-test` in the OpenCode environment during manual
  testing, restarts OpenCode.
- **Silent failure.** Every `callBrains` is try/caught; errors go to
  stderr (which OpenCode captures to its own logs), hook still returns
  cleanly, OpenCode pipeline never blocks.
- **Append-only on `output.system`.** Never reassigns or touches index 0.
  Preserves OpenCode's two-part caching collapse.
- **`cwd` comes from `input.directory`** (supplied by OpenCode per
  `plugin/index.ts:138-153`), not `process.cwd()`, so multi-workspace
  sessions resolve rules from the correct project.
- **`tool_name` and `tool_input.file_path` use zombiekit's existing
  `HookEvent` schema**, so the opencode editor's `ExtractFilePaths`
  reads them with no additional parsing.

## Embedding

Add to `embed.go`:

```go
//go:embed embed/integrations/opencode/brains.ts
var embeddedOpencodeShim embed.FS

var EmbeddedOpencodeShim fs.FS
```

and in `init()`:

```go
EmbeddedOpencodeShim, err = fs.Sub(embeddedOpencodeShim, "embed")
if err != nil {
    panic("embed: opencode shim: " + err.Error())
}
```

A future `brains opencode install` subcommand can materialize this into
`.opencode/plugins/brains.ts`. Not in scope for this feature; the embed
is there so we're ready to ship it and the user always has one canonical
copy.

## Testing

### Unit tests (Go)

`internal/hook/editor_opencode_test.go` — parallel to
`editor_gemini_test.go`:

- `FormatSessionStart` with bodies → envelope with
  `hookSpecificOutput.additionalContext`; without bodies → `{}`.
- `FormatPostToolUse` same.
- `FormatPreToolUse` / `FormatSessionEnd` return `""`.
- `ExtractFilePaths` for `write`, `edit`, `multi-edit` returns the
  path; for unknown tool returns nil.
- `IsShellTool("bash")` true, other names false.

`internal/hook/handler_test.go` — add OpenCode coverage:

- `session-inject` (SessionStart + Source=inject) fires unconditional
  rules on first call, returns empty on second (dedup).
- `compact` (SessionStart + Source=compact) resets dedup and re-fires
  unconditional rules after a previous `session-inject` call. This is
  the regression test for the issue the user flagged.
- Graphite status fires once across multiple `session-inject` calls,
  fires again after `compact`.

`internal/hook/agent_test.go` — add `AgentOpenCode` round-trip via
`ResolveEditor("opencode")`.

`internal/cli/hook_test.go` (if it exists, otherwise skip) — `--event
session-inject` and `--event compact` wire through to the handler with
correct Source.

### Unit test (shim)

Not in scope. The shim is short enough that unit-testing it inside a
Bun harness costs more than it delivers. Coverage comes from:

1. The Go side (tested).
2. The `brains` CLI contract (integration-tested via
   `internal/hook/handler_test.go`).
3. Manual end-to-end in real OpenCode (see below).

### Manual E2E

1. Build and install a test binary under a distinct name:
   `go build -o $GOPATH/bin/brains-test ./cmd/brains`.
2. Copy `embed/integrations/opencode/brains.ts` into the test project's
   `.opencode/plugins/brains.ts`.
3. Stop OpenCode if it's running.
4. Start OpenCode with `BRAINS_BIN=brains-test` in its environment.
5. Drop a `.brains/rules/test-unconditional.md` with no `paths` field and
   a recognizable body. Start a new session. Verify the body appears in
   the model's system prompt (the first turn's request).
6. Drop a `.brains/rules/test-glob.md` with `paths: ["**/*.go"]`. Have
   the agent edit a `.go` file. Verify the body appears appended to the
   tool result the model reads next.
7. Trigger compaction. Verify unconditional rules still appear in the
   compacted state's context (this is the regression check).
8. Edit a non-matching file. Verify no injection (negative).

The user performs this test manually and reports results back. The
development loop for iteration: stop OpenCode → rebuild `brains-test` →
restart OpenCode → re-run.

## Open Items Deferred to Tasks

- Exact JSON field names the shim reads off `output.args` for each tool
  — research said `filePath` (camelCase) for `write`/`edit`/`multi-edit`
  but it's worth a quick verification against the OpenCode source on the
  specific commit we're targeting. If field names differ, the shim's
  `extractFilePath` is the only place that needs updating.
- Whether `input.directory` is always populated or sometimes undefined.
  Guard with a fallback to `process.cwd()` if needed; do not block rule
  resolution on its absence.

## Risk Assessment

| Risk | Impact | Mitigation |
|------|--------|------------|
| `system.transform` blocks stream start on first turn | User-visible latency | `brains` cold-call is fast (<50ms in practice); shim's try/catch cuts off runaway failures |
| OpenCode renames `experimental.*` hooks | Shim breaks silently | README warns; add a runtime log the first time a hook fires, so users can confirm registration |
| Shim receives an unknown tool name | Silent no-injection | Intentional — unknown tools shouldn't trigger rules |
| User forgets `BRAINS_BIN` when swapping binaries | Tests run against stale binary | Shim logs the binary it's about to spawn, visible in OpenCode's plugin log |
| `output.system.push` on wrong index | Upstream cache invalidation | Code comment explicitly flags this; append-only enforced at the call site |
