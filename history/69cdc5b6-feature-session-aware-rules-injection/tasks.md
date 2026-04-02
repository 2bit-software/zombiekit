# Tasks: Session-Aware Rules Injection

**Complexity**: Medium (12 source files, 8 test files, 2 packages)
**Critical Path**: T001 → T002 → T004 → T005 → T008 → T010 → T011 → T012

## Dependency Graph

```
T001 (dep + types) ──┬──→ T002 (frontmatter) ──→ T005 (service)
                     ├──→ T003 (resolver)     ──→ T005
                     └──→ T004 (matcher)      ──→ T005
                                                    │
T006 (hook types) ──→ T007 (session state) ────┐    │
                                               ├──→ T010 (handler)
T008 (agent detection) ───────────────────────┘    │
                                                    │
T009 (empty-body filter in service) ← T005 ────────┘
                                                    │
T010 ──→ T011 (CLI integration) ──→ T012 (hook registration JSON)
                                                    │
T013-T020 (tests) ← after corresponding source tasks
```

**Parallel groups:**
- Group A: T002, T003, T004 (after T001)
- Group B: T006, T008 (independent of Group A)
- Group C: T013-T020 (tests, after their source tasks)

## Tasks

### Foundation

- [ ] T001 [FR-009] Add `doublestar/v4` dependency and create rules types
  - `go get github.com/bmatcuk/doublestar/v4`
  - Create `internal/rules/types.go`: `Rule`, `RuleSource`, `RuleFrontmatter` structs, `Rule.ID()`, `Rule.IsUnconditional()`
  - **Accept**: Types compile. `Rule.ID()` returns `"project:go.md"` format. `IsUnconditional()` returns true when Paths is nil/empty.

### Rules Package (Group A — parallelizable after T001)

- [ ] T002 [P] [FR-009] Implement rules frontmatter parsing
  - Create `internal/rules/frontmatter.go`: `ParseRule(content []byte, name, filePath string, source RuleSource) (*Rule, error)`
  - Use `adrg/frontmatter` (already in go.mod) to parse `RuleFrontmatter`
  - Implement `NormalizedPaths()`: handle nil, string, []string, []interface{} — no comma splitting
  - **Accept**: Parses `paths: ["**/*.go"]` (array), `paths: "**/*.go"` (string), no frontmatter (nil paths). Body extracted correctly.

- [ ] T003 [P] [FR-007] Implement rules directory resolver
  - Create `internal/rules/resolver.go`: `Resolver` struct, `FindRulesDirs()`, `LoadRules()`
  - Ancestor walk from CWD to git root collecting `.brains/rules/` directories at each level
  - Global directory at `~/.brains/rules/`
  - Follow `profile/resolver.go` pattern (git root detection, ancestor walk, global fallback)
  - Skip missing directories silently
  - **Accept**: Finds `.brains/rules/` at CWD and parent dirs up to git root. Finds `~/.brains/rules/`. Returns `[]*Rule` with correct Source (project vs global).

- [ ] T004 [P] [FR-005] Implement glob pattern matcher
  - Create `internal/rules/matcher.go`: `MatchRules(rules []*Rule, filePath string) []*Rule`
  - Use `doublestar.Match()` with forward-slash normalized paths
  - Return all rules whose any pattern matches the file path
  - **Accept**: `**/*.go` matches `src/main.go`. `*.{ts,tsx}` matches `component.tsx`. Non-matching paths return empty slice. Unconditional rules not included (handled separately).

### Rules Service

- [ ] T005 [FR-001, FR-002, FR-006, FR-013] Implement rules service
  - Create `internal/rules/service.go`: `Service` struct, `NewService(workingDir, homeDir string)`
  - `ResolveForFile(filePath string) ([]*Rule, error)` — load rules, match against path
  - `ResolveUnconditional() ([]*Rule, error)` — return rules with no paths
  - `ResolveForFiles(filePaths []string) ([]*Rule, error)` — batch for MultiEdit, deduplicated
  - Rules loaded from disk on each call (hook binary is short-lived, no caching needed)
  - Filter out rules with empty Body (frontmatter only, no content to inject)
  - **Accept**: Returns matching rules for a given file path. Returns unconditional rules. MultiEdit batch returns deduplicated union. Empty-body rules excluded.

### Hook Package (Group B — parallel with Group A)

- [ ] T006 [P] Create hook event types
  - Create `internal/hook/types.go`: `HookEvent`, `ToolInput`, `EditEntry`, `ToolResponse`, `SessionState`, `Agent` const
  - JSON tags match exact Claude Code/Gemini CLI wire format (snake_case for input, camelCase for response)
  - **Accept**: `json.Unmarshal` correctly deserializes all 5 stdin JSON examples from spec's Interface Contract.

- [ ] T007 [FR-003, FR-011, FR-012] Implement session state management
  - Create `internal/hook/session.go`: `LoadState()`, `SaveState()`, `DeleteState()`, `IsRuleInjected()`, `MarkRuleInjected()`, `ResetInjectedRules()`
  - State file at `/tmp/zk-session-{SESSION_ID}.json`
  - Atomic writes (temp file + os.Rename)
  - Missing/corrupt file → return fresh state (never error)
  - **Accept**: Creates state file on first save. Reads back correctly. Reset clears injected map and increments compaction count. Corrupt JSON → fresh state. Delete removes file.

- [ ] T008 [P] [FR-008] Implement agent detection and output formatting
  - Create `internal/hook/agent.go`: `DetectAgent() Agent`, `FormatOutput(agent Agent, bodies []string) string`
  - Check `CLAUDE_SESSION_ID` first, then `GEMINI_SESSION_ID`
  - Claude: single `<system-reminder>` wrapper around all rules concatenated with blank line
  - Gemini: plain markdown, rules concatenated with blank line
  - **Accept**: Claude detection wraps in `<system-reminder>`. Gemini outputs plain markdown. Both set when both envs present → Claude wins.

- [ ] T009 Add `SessionID(event *HookEvent) string` helper
  - In `internal/hook/agent.go` or `types.go`: extract session ID from event's `session_id` field (authoritative source for state file path)
  - **Accept**: Returns event.SessionID. Used by handler for state file path.

### Handler (merges Group A + B)

- [ ] T010 [FR-001, FR-002, FR-004, FR-006, FR-013, FR-014] Implement hook event handler
  - Create `internal/hook/handler.go`: `Handler` struct, `NewHandler()`, `Handle(event *HookEvent) (string, error)`
  - `handleSessionStart()`: reset tracking for all sources (startup/resume/compact), inject unconditional rules, save state
  - `handlePostToolUse()`: extract file paths per tool type (Read/Write/Edit/MultiEdit), resolve matching rules, filter already-injected, mark injected, save state
  - `handleSessionEnd()`: delete state file
  - File path extraction: `ToolInput.FilePath` for Read/Write/Edit, `ToolInput.Edits[].FilePath` for MultiEdit, fallback to `ToolResponse.FilePath` for Write/Edit
  - **Accept**: SessionStart outputs unconditional rules. PostToolUse Read outputs matching rules (first time only). PostToolUse Write skips if rules already injected. MultiEdit handles multiple paths. SessionEnd deletes state. Compaction resets tracking.

### CLI Integration

- [ ] T011 [FR-014] Wire up `brains hook` CLI command
  - Create `internal/cli/hook.go`: `newHookCommand()` with `--event` flag (required, values: session-start, post-tool-use, session-end)
  - Action: read stdin JSON → parse HookEvent → detect agent → create Handler → call Handle → print to stdout
  - Exit 0 on success, exit 1 on error (stderr only, never stdout)
  - Add `newHookCommand()` to Commands slice in `internal/cli/root.go`
  - **Accept**: `echo '{"session_id":"test","hook_event_name":"SessionStart","cwd":"/tmp","source":"startup"}' | brains hook --event session-start` produces output (or empty if no rules). Exit 0.

- [ ] T012 Document hook registration JSON for settings.json
  - Create/update documentation or example showing the hooks config block for Claude Code and Gemini CLI settings.json
  - Include exact `matcher` and `command` values
  - **Accept**: JSON block ready to paste into settings.json.

### Tests (Group C — after corresponding source tasks)

- [ ] T013 [P] Unit tests for rules frontmatter parsing
  - Create `internal/rules/frontmatter_test.go`
  - Test: array paths, string path, no frontmatter, empty paths, brace expansion passthrough, body extraction
  - **Accept**: All cases pass.

- [ ] T014 [P] Unit tests for glob matcher
  - Create `internal/rules/matcher_test.go`
  - Test: `**/*.go` matches nested paths, `*.{ts,tsx}` matches both extensions, non-match returns empty, forward-slash normalization
  - **Accept**: All cases pass.

- [ ] T015 [P] Unit tests for agent detection
  - Create `internal/hook/agent_test.go`
  - Test: Claude env → Claude, Gemini env → Gemini, both → Claude, neither → Gemini (default)
  - Test: FormatOutput wraps correctly for each agent
  - **Accept**: All cases pass.

- [ ] T016 [P] Integration test for session state lifecycle
  - Create `internal/hook/session_test.go`
  - Test: create → read → mark injected → reset → read (empty) → delete → read (fresh)
  - Test: corrupt JSON → fresh state
  - **Accept**: All lifecycle transitions pass. Uses temp directory.

- [ ] T017 Integration test for rules resolver
  - Create `internal/rules/resolver_test.go`
  - Test: project rules found, global rules found, both composed, missing dir skipped
  - **Accept**: All cases pass. Uses temp directory structure.

- [ ] T018 Integration test for handler — SessionStart
  - Create `internal/hook/handler_test.go` (part 1)
  - Test: startup injects unconditional rules, compact resets + re-injects, resume resets + re-injects
  - **Accept**: All SessionStart variants produce correct output and state.

- [ ] T019 Integration test for handler — PostToolUse
  - `internal/hook/handler_test.go` (part 2)
  - Test: Read injects matching rules, second Read skips (dedup), Write injects if not seen, MultiEdit injects for multiple types
  - **Accept**: All PostToolUse variants produce correct output and state.

- [ ] T020 Integration test for handler — SessionEnd + edge cases
  - `internal/hook/handler_test.go` (part 3)
  - Test: SessionEnd deletes state file, missing state file → fresh session, empty-body rules skipped
  - **Accept**: All edge cases pass.

## FR Traceability

| FR | Tasks |
|----|-------|
| FR-001 | T004, T005, T010, T019 |
| FR-002 | T005, T010, T019 |
| FR-003 | T007, T016 |
| FR-004 | T007, T010, T018 |
| FR-005 | T004, T014 |
| FR-006 | T005, T010, T018 |
| FR-007 | T003, T017 |
| FR-008 | T008, T015 |
| FR-009 | T001, T002, T013 |
| FR-010 | Architectural (Go binary startup + file I/O within budget) |
| FR-011 | T007, T016, T020 |
| FR-012 | T007, T020 |
| FR-013 | T005, T010, T019 |
| FR-014 | T011 |

## Execution Order

**Phase 1** (parallel): T001
**Phase 2** (parallel): T002, T003, T004, T006, T008
**Phase 3** (parallel): T005, T007, T009
**Phase 4**: T010
**Phase 5**: T011, T012
**Phase 6** (parallel): T013, T014, T015, T016, T017, T018, T019, T020
