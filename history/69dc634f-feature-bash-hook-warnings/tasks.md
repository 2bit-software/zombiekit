# Tasks: Bash Hook Command Warnings

**Complexity**: Medium (9 files changed/created, ~600 LOC, 3 modules touched)
**Total tasks**: 17
**Critical path**: T001 → T002 → T006 → T012 → T014 → T017 (6 steps)

## Legend

- `[P]` parallelizable with sibling tasks at same dependency level
- `[US1]`..`[US4]` user story from spec.md

---

## Phase A — Pure additions (no behavior change yet)

- [ ] **T001** [P] Add `Commands`, `RequiresFiles`, `RequiresFilesAbsent` fields to `Rule` struct and `RuleFrontmatter` in `internal/rules/types.go`. Add `NormalizedCommands`, `NormalizedRequiresFiles`, `NormalizedRequiresFilesAbsent` methods mirroring `NormalizedPaths`. Update `IsUnconditional` to require `len(Commands) == 0` in addition to `len(Paths) == 0`. Existing tests MUST still pass.

- [ ] **T002** [P] [US1] Create `internal/rules/command_matcher.go` with pure functions `SplitSegments`, `StripEnvPrefix`, `isEnvAssignment`, `MatchCommandPrefix`, `MatchRulesByCommand`. Define `RuleMatch{Rule, Trigger}` struct. No I/O. No dependency on any other file in this batch.

- [ ] **T003** [P] [US1] Create `internal/rules/command_matcher_test.go` covering: literal prefix match, non-match (`gopher test` vs `go test`), env prefix strip (`CGO_ENABLED=0 go test`), chained commands (`cd x && go test`), multi-segment multiple matches, segment-split with `||`/`;`/`|`, empty command, empty matcher. Test iteration order is deterministic when multiple rules share a command.

- [ ] **T004** [P] [US3] Create `internal/rules/gate.go` with `GateResolver` type. Constructor walks cwd up to `.git` to find repo root (fall back to cwd). Inject `stat func(string)(os.FileInfo, error)` for tests. Implement `Passes(rule *Rule) bool` — checks all `RequiresFiles` exist AND all `RequiresFilesAbsent` do not exist, walking cwd→repoRoot for each file.

- [ ] **T005** [P] [US3] Create `internal/rules/gate_test.go` using a fake `stat` func. Cases: both fields nil (passes), `requires_files` all present (passes), one missing (fails), `requires_files_absent` all absent (passes), one present (fails), both set both satisfied (passes), walk-up finds file in parent dir, walk-up stops at repo root.

**Phase A gate**: `task dev -- test ./internal/rules/...` green. These five tasks touch disjoint files and run in parallel.

---

## Phase B — Wire matcher into service

- [ ] **T006** [US1][US2] Update `internal/rules/resolver.go` (and/or wherever `RuleFrontmatter` is parsed into `Rule`) to populate the new Commands/RequiresFiles/RequiresFilesAbsent fields from normalized frontmatter. **Depends on**: T001. Verify by adding an integration-style test fixture under a temp `.brains/rules/` directory with the three new fields and asserting the loaded `Rule` has them.

- [ ] **T007** Add `ResolveForCommand(cmd, cwd string) ([]RuleMatch, error)` to `internal/rules/service.go`. Loads all rules, filters to those with non-empty `Commands`, runs `GateResolver.Passes`, then `MatchRulesByCommand`, drops empty-body rules. **Depends on**: T001, T002, T004, T006.

- [ ] **T008** [P] Add `ResolveForCommand` integration test in `internal/rules/service_test.go` (or the existing file if present). Two rules with same commands — one gated on `requires_files: [Taskfile.yml]`, one on `requires_files_absent`. Assert correct rule fires based on temp-dir contents. **Depends on**: T007.

**Phase B gate**: `task dev -- test ./internal/rules/...` green with new tests.

---

## Phase C — Hook-side refactor (HandleResult shape + dedup key)

- [ ] **T009** [P] Add `Command string `json:"command,omitempty"`` to `ToolInput` in `internal/hook/types.go`. One-line change. **Depends on**: nothing. Trivial; parallelizable with T001-T005.

- [ ] **T010** Define `MatchedRule{ID, Trigger string}` in `internal/hook/types.go`. Change `HandleResult.MatchedRuleIDs []string` → `MatchedRules []MatchedRule`, and `SkippedRuleIDs` → `SkippedRules`. Update `internal/hook/handler.go` `handleSessionStart` and existing `handlePreToolUse` (file path) to populate with empty-trigger entries. **Depends on**: T009 (same file).

- [ ] **T011** Update `internal/cli/hook.go`, `internal/hook/audit.go`, `internal/hook/filesink.go`, and `internal/cli/hook_log.go` call sites to consume `[]MatchedRule` instead of `[]string`. `AuditRecord` fields gain a `Trigger` field where relevant so the JSONL log captures which trigger fired. **Depends on**: T010.

- [ ] **T012** Refactor dedup in `internal/hook/state.go` (or wherever `SessionState`/`IsRuleInjected`/`MarkRuleInjected` live). Add `IsRuleInjectedFor(state, ruleID, trigger)` and `MarkRuleInjectedFor(state, ruleID, trigger)` that use composite key `ruleID + "|" + trigger`. Add one-shot migration in `LoadState` that rewrites any bare key (no `|`) to `key+"|"`. Keep old `IsRuleInjected`/`MarkRuleInjected` as wrappers passing `trigger=""`. **Depends on**: T010.

- [ ] **T013** [P] Add unit test in `internal/hook/state_test.go` (create if missing) for the legacy-state migration: write a state file with bare-ruleID keys, `LoadState`, assert keys rewritten and `IsRuleInjectedFor(..., "")` returns true for migrated entries. **Depends on**: T012.

**Phase C gate**: full existing hook test suite green; call sites compile; legacy state migrates correctly.

---

## Phase D — Bash branch in the handler

- [ ] **T014** [US1] Add `handlePreBash` to `internal/hook/handler.go`. Body: load state, call `rules.ResolveForCommand(event.ToolInput.Command, event.CWD)`, iterate matches with per-trigger dedup, collect bodies, save state, wrap with `FormatPreToolOutput`. Insert `if event.ToolName == "Bash" { return h.handlePreBash(event) }` at the top of `handlePreToolUse`. **Depends on**: T007, T012.

- [ ] **T015** [P] [US1][US2][US3][US4] Add Bash-path integration tests in `internal/hook/handler_test.go`:
  1. Single rule fires on `go test ./...`, trigger recorded as `"go test"`.
  2. Same rule second call → skipped (dedup by trigger).
  3. Same rule, different trigger (`go run`) → fires again.
  4. Non-matching command (`ls`) → empty output, no state mutation.
  5. Taskfile-gated rule fires only when temp dir contains `Taskfile.yml`.
  6. `requires_files_absent` rule fires only when Taskfile missing.
  7. Rule with both `paths:` and `commands:` fires on file event and on bash event — independently. **Depends on**: T014.

---

## Phase E — Documentation

- [ ] **T016** [P] Update `INFRASTRUCTURE.md` with a "Bash command rules" subsection. Include the two-rule Taskfile example (present-body and absent-body), document matcher limits (no shell parsing, prefix-only, top-level separator split), and the gate resolution order. **Depends on**: nothing structural; write after T014 so docs match shipped behavior.

---

## Phase F — Final verification

- [ ] **T017** Run `task dev -- test ./...` (or `task check`) on the full repo. Fix any regressions in uncommitted audit/hook_log work that the `HandleResult` rename caused. Verify no new lint warnings. Confirm:
  - All spec FRs (FR-001..FR-008, FR-004b..e) have at least one test.
  - `hook log --follow` renders trigger info in the JSONL output.
  - Legacy state files still load.
  **Depends on**: T001–T016.

---

## Dependency Graph

```
T001 ─┬─> T006 ──> T007 ──> T008
T002 ─┤              │
T003  │              │
T004 ─┤              │
T005  │              │
T009 ─> T010 ─> T011 │
                 └─> T012 ─> T013
                        │
T007 ────────────────────┴───> T014 ──> T015
                                  └───> T016
All ───────────────────────────────────> T017
```

## Parallelization

- **Wave 1** (fully parallel): T001, T002, T003, T004, T005, T009
- **Wave 2**: T006, T010 (T010 depends on T009)
- **Wave 3**: T007, T011, T012
- **Wave 4**: T008, T013, T014
- **Wave 5**: T015, T016
- **Wave 6**: T017

6-wave critical path. Waves 1–3 each parallelize 3–6 tasks.

## FR Traceability

| FR | Tasks |
|---|---|
| FR-001 | T009, T014 |
| FR-002 | T001, T006, T007 |
| FR-003 | T002, T003 |
| FR-004 | T001, T004, T005 |
| FR-004b | T001, T004, T005 |
| FR-004c | T004, T007 |
| FR-004d | T012, T013 |
| FR-004e | T010, T011 |
| FR-005 | T001 |
| FR-006 | T014 |
| FR-007 | T012, T014, T015 |
| FR-008 | T011, T015 |

No FR is orphaned. No task is orphaned.

## Next

Run `/brains.next` to advance to `implement` and start executing the task list.
