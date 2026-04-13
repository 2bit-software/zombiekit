# Research Summary

## Hook Architecture (current state)

- Entrypoint: `internal/cli/hook.go:31` reads `HookEvent` from stdin,
  dispatches via `Handler.Handle`.
- Dispatch: `internal/hook/handler.go:33` switches on event name —
  `SessionStart`, `PreToolUse`, `SessionEnd`.
- PreToolUse today only handles file-path tools (Read/Write/Edit/
  MultiEdit) via `HookEvent.ExtractFilePaths` at
  `internal/hook/types.go:44`. **Bash is not handled** — the handler
  returns `HandleResult{}` when no file paths are present
  (`handler.go:90`).
- `HookEvent.ToolInput` (`types.go:17`) only exposes `FilePath` and
  `Edits` today. We must add a `Command string` field.
- Output is wrapped by `FormatPreToolOutput` (`internal/hook/agent.go:51`)
  into the Claude Code JSON envelope with
  `permissionDecision: "allow"`, hardcoded — warnings are already
  structurally non-blocking, no changes needed there.

## Rules System (current state)

- Rules live in `.brains/rules/*.md` (project, parent, global) with YAML
  frontmatter. `RuleFrontmatter` is defined in
  `internal/rules/types.go:37`.
- Today the only matcher field is `paths:` (string or list, normalized).
- `rules.Service` exposes `ResolveUnconditional()` and `ResolveForFiles()`
  (`internal/rules/service.go:31,48`). Matching uses `doublestar` globs
  via `matchesAnyPattern` (`internal/rules/matcher.go:26`).
- A rule is "unconditional" iff `Paths == nil` and is injected at
  SessionStart. We must keep this invariant when adding `commands:` — a
  rule with only `commands:` is conditional, not unconditional.
- Session-level dedup via `SessionState.InjectedRules`
  (`internal/hook/types.go:78`) is keyed by rule ID, already works for
  any new rule path.

## Uncommitted audit work (in flight)

- `internal/hook/audit.go` (AuditRecord + AuditSink interface) and
  `internal/hook/filesink.go` (JSONL writer to `~/.zombiekit/logs/
  hooks.jsonl`) are complete but not wired into `handler.go` yet.
- `internal/cli/hook_log.go` is a working `hook log` subcommand.
- Our feature must call the same sink so command-matched rules show up
  in `hook log` alongside file-matched rules.

## Design Decisions

### Extend existing rules vs separate config
**Decision**: Extend `RuleFrontmatter` with `commands:` and
`requires_files:`. One rule system keeps the authoring surface small.

### Matching semantics
**Decision**: Prefix tokens against each command segment.
- Split command on `&&`, `||`, `;`, `|` (surface-level, no shell
  parsing).
- Strip leading `VAR=value` env assignments from each segment.
- For each matcher `m`, segment matches iff `segment == m` or
  `strings.HasPrefix(segment, m+" ")`.

No regex, no glob, no real shell parser. Covers the 80% case; upgrade
later if needed.

### Taskfile conditional gate
**Decision**: Add symmetrical `requires_files:` and
`requires_files_absent:` fields. ALL-semantics on both sides. `fs.Stat`
per file, resolved by walking from `event.CWD` up to repo root. Both
fields may coexist on a rule. Gates evaluate before the command
matcher.

Use case: two rules, same `commands:` list. One gated on
`requires_files: [Taskfile.yml]` says "use task dev -- test". The
other gated on `requires_files_absent: [Taskfile.yml]` says
"consider adding a Taskfile". Mutually exclusive by gate.

### Per-trigger dedup
**Decision**: `SessionState.InjectedRules` keys become
`ruleID|trigger`. File-rule injections use empty trigger (`ruleID|""`),
preserving existing behavior. Command-rule injections use the matched
command prefix as the trigger, so `go test` and `go run` triggering
the same rule body fire independently, while `go test` twice in a
session dedups.

### Default rules
**Decision**: Ship no defaults. Document an example in
INFRASTRUCTURE.md. Projects author their own under `.brains/rules/`.

## Weakest Links

- **Command parsing drift**: naive segment split will miss heredocs,
  `bash -c "..."`, backticks. Document the limits; fire silently
  rather than falsely.
- **requires_files resolution**: walk up to repo root so subdirectory
  invocations still find the Taskfile.
- **Dedup vs repeated nudging**: rule fires once per session. If user
  ignores the first warning, subsequent runs go quiet. Parity with
  existing rules > nagging.

## Files That Will Change

- `internal/hook/types.go` — add `Command` to `ToolInput`.
- `internal/hook/handler.go` — branch on `tool_name == "Bash"`.
- `internal/rules/types.go` — add `Commands`, `RequiresFiles` fields.
- `internal/rules/matcher.go` — add `MatchRulesByCommand`.
- `internal/rules/service.go` — add `ResolveForCommand(cmd, cwd)`.
- `internal/hook/handler_test.go` — new Bash-path tests.
- `internal/rules/matcher_test.go` (new) — unit tests for command
  tokenizer and prefix matcher.
- `INFRASTRUCTURE.md` — document authoring a command rule.
