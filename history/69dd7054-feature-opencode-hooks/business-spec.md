# Business Spec: OpenCode Hook Support

## Summary

Extend zombiekit's hook system so OpenCode users get the same automatic rule
injection that Claude Code and Gemini CLI users already have: when the agent
edits a file, matching rules (`.brains/rules/**.md`) are injected into the
next thing the model reads, gated by path globs and per-session deduplication.

## Motivation

Rule injection is zombiekit's core developer-experience feature and currently
works for Claude Code (via `PreToolUse` hooks) and Gemini CLI (via
`BeforeTool` / `AfterTool` hooks). OpenCode is the third AI coding CLI we want
to cover. Its plugin model is fundamentally different — in-process JS/TS, not
subprocess — so a direct `--editor opencode` flag is insufficient; a bridge
plugin is also required.

## Functional Requirements

### FR-1: OpenCode editor in `brains hook`

`brains hook --editor opencode --event <event>` accepts a normalized JSON
payload on stdin and emits a JSON response consumable by the OpenCode plugin
shim. The editor integrates with the existing editor registry (`Editor`
interface in `internal/hook/editors.go`); it does not modify the shared
handler, rule matcher, or session dedup.

### FR-2: Normalized payload contract

The OpenCode editor defines its own stdin payload schema — NOT OpenCode's
native event shape. The shim is responsible for translating
`input.tool` + `output.args` into this schema before spawning `brains`. The
schema must carry at minimum:

- `tool`: string (e.g. `"write"`, `"edit"`, `"multi-edit"`)
- `file_path` or `file_paths`: the edited file(s), absolute paths
- `session_id`: to key the existing session-dedup state file
- Event source metadata sufficient for audit logging

The schema is versioned (a `version` field or equivalent) so the shim and
editor can evolve together without silent breakage.

### FR-3: Response shape consumable by the shim

`brains hook --editor opencode` stdout is a JSON object mirroring the
convergent field name used by the Claude and Gemini editors:

```json
{"hookSpecificOutput": {"additionalContext": "<rule bodies>"}}
```

Empty bodies produce `{}` — always valid JSON, same invariant as the
Gemini editor. The shim detects "no rules" by absence of
`hookSpecificOutput` and performs no mutation in that case. The shape has
no `decision` or `permissionDecision` field because neither
`tool.execute.after` nor `experimental.chat.system.transform` has a
corresponding `output.decision` semantic in OpenCode.

### FR-4: Path extraction and rule matching

The OpenCode editor's `ExtractFilePaths` reads paths from the normalized
payload — it does not need to know OpenCode tool-arg shapes, because the
shim has already normalized them. All path globbing, rule resolution, and
dedup continue to use the shared handler path; no duplication.

### FR-5: Shim plugin script

A `.ts` file is committed to the zombiekit repo at a `//go:embed`-friendly
path (working assumption: `internal/hook/opencode/brains.ts` alongside a
`.go` file that embeds it — final location decided during planning). For
this iteration the user hand-copies the file into
`.opencode/plugins/brains.ts`. A future `brains opencode install`
subcommand will materialize the embedded asset automatically; that command
is out of scope for this feature.

The shim registers three hooks:

- **`tool.execute.after`** — file-edit injection. Filters to file-editing
  tools (`write`, `edit`, `multi-edit`), reads the file path from
  `output.args` per tool, spawns
  `brains hook --editor opencode --event post-tool-use` with a normalized
  payload on stdin, and on non-empty rule output appends the rule text to
  `output.output` so the model sees it as part of the tool result.
- **`experimental.chat.system.transform`** — unconditional (no-path-glob)
  rule injection at session start, equivalent to Claude/Gemini SessionStart.
  Fires on every assistant turn; zombiekit's existing
  `/tmp/zk-session-<id>.json` per-session dedup makes second-and-later
  calls a no-op. Spawns
  `brains hook --editor opencode --event session-start` with `session_id`
  from `input.sessionID`, and on non-empty output **appends** a new entry
  to `output.system`. The shim must **never modify `output.system[0]`** —
  OpenCode's LLM layer collapses `system[1..n]` into a single block to
  preserve byte-identical `system[0]` for upstream provider prompt caching
  (`session/llm.ts:136-141`).
- **`experimental.session.compacting`** — unconditional rule re-injection
  across compaction boundaries. Compaction runs on its own path and does
  not re-enter `LLM.stream`, so `system.transform` does not fire during
  compaction; without explicit handling, unconditional rules would be lost
  from the compacted state. The shim spawns
  `brains hook --editor opencode --event compact --session-id <id>`. Two
  things happen:
  1. `brains` resets the session dedup state so that subsequent
     `system.transform` calls behave correctly after compaction (mirroring
     Claude's existing `Source: "compact"` SessionStart reset).
  2. `brains` returns the unconditional rule bodies, and the shim pushes
     them onto `output.context` so the compacted conversation is born
     with the rules already present — no reliance on next-turn timing.

The shim's `brains` binary path is configurable via a `BRAINS_BIN`
environment variable, defaulting to `brains` on PATH. This is required for
the manual test workflow: the user runs a side-installed binary (e.g.
`brains-test`) without touching the main installation. OpenCode must be
stopped, the test binary installed, and OpenCode restarted with
`BRAINS_BIN=brains-test` for the swap to take effect — hot-swap is not
required.

The shim fails silently (logs to stderr) on `brains` errors — never blocks
the OpenCode tool pipeline or the stream start.

### FR-6: Test binary workflow

The implementation plan must support swapping the `brains` binary the shim
calls without restarting the agent session that wrote the shim. Concretely:

- The shim reads the binary name/path from an env var or editable constant.
- The development workflow documents how to install `brains` under an
  alternate name (e.g. `go build -o $GOPATH/bin/brains-test ./cmd/brains`)
  and point the shim at it.
- Before the user runs `go install` of the test binary, they will stop
  OpenCode, install, then restart — the spec does not need to support hot
  swap, only the name split.

### FR-7: Documentation

README gets an OpenCode section parallel to the existing Claude Code and
Gemini CLI sections, covering:

- Where to drop the shim file.
- How to register it in `opencode.json` if auto-discovery is not in play.
- The binary-name environment variable for development/swap workflows.
- The caveat that `OPENCODE_PURE=1` disables all plugins.

## Acceptance Criteria

- [ ] `brains hook --editor opencode --event post-tool-use` reads a
      normalized JSON payload from stdin and writes a JSON response to
      stdout without error.
- [ ] Given a `.brains/rules/*.md` rule with a path glob that matches the
      payload's file path, the response contains the rule body.
- [ ] Given no matching rule, the response is the documented "no-op" shape
      and the shim performs no mutation.
- [ ] Session dedup: the same rule is not injected twice within a single
      `session_id` for the same trigger.
- [ ] Compaction preserves unconditional rules: when
      `experimental.session.compacting` fires, `brains` resets session
      dedup AND returns the unconditional rule bodies; the shim pushes
      them onto `output.context` so they appear in the compacted state.
      Covered by a unit test on the `brains` side (compact event resets
      dedup and emits the expected bodies) and a documented manual test
      step on the shim side.
- [ ] A `.ts` shim in the repo, when copied to `.opencode/plugins/brains.ts`
      and run under OpenCode with a rule and a matching file edit, causes
      the injected rule text to appear in the model's next turn (manually
      verified by the user in real OpenCode).
- [ ] The shim reads its `brains` binary path from an env var (or edit
      point) so the user can target `brains-test` during iteration.
- [ ] Existing Claude and Gemini editor tests still pass; the OpenCode
      editor has its own unit tests covering format output for the
      match / no-match / empty-bodies cases (parallel to
      `editor_gemini_test.go`).
- [ ] README documents OpenCode setup.

## Out of Scope

- **An `install` / `setup` subcommand** that generates the shim file or
  edits `opencode.json`. Deferred — the user will hand-copy the `.ts` file
  for now.
- **An npm package** for the shim. Raw `.ts` file only.
- **`tool.execute.before` (PreToolUse-equivalent).** Deferred. File-edit
  rules fire on `tool.execute.after` only; we do not attempt to mimic
  Claude's pre-permission injection point in this cut.
- **Shell-command rule triggers.** OpenCode's shell tool is not covered in
  this iteration; only file-editing tools (`write`, `edit`, `multi-edit`).
- **Hot-swapping the binary while OpenCode is running.** The dev loop is
  stop-OpenCode, install, restart.
- **Upstreaming subprocess hook support to OpenCode.** Not our problem.

## Open Questions

1. **Event choice — only `tool.execute.after`, or also
   `experimental.chat.system.transform` for session-start rules?** The MVP
   covers file-edit injection only, but unconditional (no-path-glob) rules
   currently fire on SessionStart for Claude/Gemini and would silently not
   fire for OpenCode without a second hook. Decision: defer to planning step
   — note explicitly in the implementation plan.
2. **Response shape — reuse an existing editor's envelope (Gemini's
   `{decision, hookSpecificOutput.additionalContext}`) or define a new one
   specific to the shim contract?** Leaning toward a new, minimal shape
   because the shim is our own code and doesn't need to mimic any upstream
   protocol. Resolve during planning.
3. **Shim location in the repo.** `assets/opencode/`, `contrib/opencode/`,
   or somewhere else? Resolve during planning.
4. **Binary-path mechanism in the shim.** Env var (`BRAINS_BIN`) vs. an
   editable constant at the top of the file. Env var is more flexible for
   the swap workflow. Resolve during planning.
5. **`session_id` propagation.** OpenCode exposes `sessionID` on hook
   inputs; does the shim pass it through to `brains` verbatim, and does
   the existing `/tmp/zk-session-<id>.json` dedup work unchanged with
   OpenCode's ID format? Verify during planning — likely yes, since the
   dedup state is keyed by opaque string.
