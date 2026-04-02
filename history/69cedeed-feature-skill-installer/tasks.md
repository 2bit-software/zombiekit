# Tasks: Skill Installer

**Initiative**: 69cedeed-feature-skill-installer
**Complexity**: Medium (7 files, ~375 lines)
**Total tasks**: 9

## Dependency Graph

```
T001 (core logic)
├── T002 (core tests)
├── T003 (CLI command)  ──┐
└── T004 (MCP tool)    ──┤
                          ├── T005 (register CLI in root.go)
                          ├── T006 (add to KnownTools)
                          └── T007 (wire MCP in server.go)
                               ├── T008 (CLI integration test)
                               └── T009 (MCP integration test)
```

T003 and T004 are parallel (both depend only on T001).
T008 and T009 are parallel (both depend on T005+T006+T007).

## Critical Path

T001 → T003 → T005 → T007 → T008

---

## Task List

- [ ] T001 [US1] Create core install logic — `internal/skill/install.go`
  - New file. Package `skill`. Module: `github.com/2bit-software/zombiekit/internal/skill`
  - Implement `ValidateName(name string) error` — regex `^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`
  - Implement `TargetDir(global bool, workingDir string) (string, error)` — global: `~/.claude/skills/`, local: `{workingDir}/.claude/skills/` (calls `os.Getwd()` when workingDir is empty)
  - Implement `GenerateContent(name, description string) string` — exact SKILL.md template from FR-002; fallback description when empty
  - Implement `WriteSkill(targetDir, name, content string) (string, error)` — `os.MkdirAll`, `os.WriteFile`, plain-file collision check, returns full path
  - **Acceptance**: Package compiles. Template output matches FR-002 exactly. Path traversal is impossible — ValidateName rejects `.`, `/`, `\`, uppercase, underscores.

- [ ] T002 [P] [US1] Write unit tests for core install logic — `internal/skill/install_test.go`
  - Depends on: T001
  - `TestValidateName`: valid cases (`my-skill`, `a`, `abc-def-123`); invalid cases (`../evil`, `./bad`, `Bad`, `my_skill`, `-lead`, `trail-`, `""`)
  - `TestGenerateContent`: profile with description → uses it; empty description → uses fallback; output matches exact template byte-for-byte
  - `TestWriteSkill`: creates dirs when missing; idempotent overwrite (run twice, same content); other files in dir left untouched; plain-file collision returns expected error
  - Use `t.TempDir()` for all filesystem tests. No mocking.
  - **Acceptance**: `go test ./internal/skill/...` passes.

- [ ] T003 [P] [US1] Create CLI skill command — `internal/cli/skill.go`
  - Depends on: T001
  - New file in package `cli`
  - `newSkillCommand()` returns `*cli.Command{Name: "skill", Usage: "Manage Claude Code skills", Subcommands: [newSkillInstallCommand()]}`
  - `newSkillInstallCommand()`: ArgsUsage `<profile-name>`, `--global` BoolFlag
  - Action flow: validate arg present → `skill.ValidateName` → `profile.NewServiceWithSource(profile.SourceTypeBrains, "")` → `svc.Show(name, false)` → extract `result.Description` → `skill.TargetDir(c.Bool("global"), "")` → `skill.GenerateContent` → `skill.WriteSkill` → `fmt.Printf("Installed skill '%s' to %s\n", name, fullPath)`
  - Error path: on Show failure, call `svc.List()` and return `"profile %q not found. Available profiles:\n  - name1\n  - name2"` via helper `handleSkillProfileError`
  - Imports: `github.com/urfave/cli/v2`, `github.com/2bit-software/zombiekit/internal/profile`, `github.com/2bit-software/zombiekit/internal/skill`
  - **Acceptance**: File compiles. `brains skill install --help` shows usage.

- [ ] T004 [P] [US3] Create MCP skill-install tool — `internal/mcp/tools/skillinstall/tool.go`
  - Depends on: T001
  - New file. Package `skillinstall`
  - `Tool` struct (empty), `NewTool() *Tool`
  - `Execute(ctx context.Context, args map[string]any) (string, error)`: extract name/scope/working_directory → `skill.ValidateName` → `profile.NewService(workingDir)` → `svc.Show(name, false)` → extract `result.Description` → `skill.TargetDir(scope == "global", workingDir)` → `skill.GenerateContent` → `skill.WriteSkill` → return success string
  - Error path: same profile-not-found helper pattern as CLI (can share via `internal/skill` or duplicate — prefer duplicate to keep packages independent)
  - Imports: `context`, `fmt`, `strings`, `github.com/2bit-software/zombiekit/internal/profile`, `github.com/2bit-software/zombiekit/internal/skill`
  - **Acceptance**: File compiles. Execute returns success string on valid input, error on invalid profile name.

- [ ] T005 [US1] Register skill command in CLI root — `internal/cli/root.go`
  - Depends on: T003
  - Add `newSkillCommand()` to the `Commands` slice in `NewApp()` — one line insertion after `newInitCommand()`
  - **Acceptance**: `go build ./cmd/brains/...` succeeds. `brains skill --help` works.

- [ ] T006 [US3] Add skill-install to KnownTools — `internal/config/tools.go`
  - Depends on: T004
  - Add `"skill-install"` to the `KnownTools` var slice
  - **Acceptance**: `go build ./...` succeeds. `"skill-install"` appears in KnownTools.

- [ ] T007 [US3] Wire skill-install MCP tool into server — `internal/mcp/server.go`
  - Depends on: T004, T006
  - Add field to `Server` struct: `skillInstallTool *skillinstalltool.Tool`
  - In `NewServer()`: initialize `skillInstallTool: skillinstalltool.NewTool()`
  - In `registerTools()`: add `if s.config.IsToolEnabled("skill-install") { ... }` block with `mcp.NewTool("skill-install", ...)` and `s.mcpServer.AddTool(t, s.handleSkillInstall)`
  - Add handler method `handleSkillInstall(ctx, req)` following the stickymemory pattern (lines 191-203 in server.go)
  - MCP tool parameters: `name` (string, required), `scope` (string, required, Enum("local","global")), `working_directory` (string, optional)
  - Import: `skillinstalltool "github.com/2bit-software/zombiekit/internal/mcp/tools/skillinstall"`
  - **Acceptance**: `go build ./...` succeeds. MCP server starts without error.

- [ ] T008 [P] [US1] [US2] CLI integration test — `internal/cli/skill_test.go`
  - Depends on: T003, T005
  - Test `brains skill install <name>` (local) and `brains skill install <name> --global` using temp dirs
  - Verify SKILL.md created at correct path with correct content
  - Verify idempotent reinstall (exit 0 both times)
  - Verify error on unknown profile name includes available profiles
  - Verify no install when name fails validation
  - **Acceptance**: `go test ./internal/cli/...` passes.

- [ ] T009 [P] [US3] MCP tool integration test — `internal/mcp/tools/skillinstall/tool_test.go`
  - Depends on: T004
  - Test Execute with scope=local+working_directory, scope=global, scope=local without working_directory
  - Verify SKILL.md file content and path
  - Verify error returned (not panic) for unknown profile name
  - Use `t.TempDir()` for all paths
  - **Acceptance**: `go test ./internal/mcp/tools/skillinstall/...` passes.

---

## Execution Order

**Wave 1** (sequential): T001
**Wave 2** (parallel): T002, T003, T004
**Wave 3** (sequential): T005, T006 (tiny — can be done together)
**Wave 4** (sequential): T007
**Wave 5** (parallel): T008, T009

## FR Traceability

| FR | Tasks |
|----|-------|
| FR-001 | T001 (WriteSkill) |
| FR-002 | T001 (GenerateContent), T002 |
| FR-003 | T001 (GenerateContent) |
| FR-004 | T001 (TargetDir), T003 (--global flag) |
| FR-005 | T001 (MkdirAll) |
| FR-006 | T003, T004 (error handling) |
| FR-007 | T001 (ValidateName), T002 |
| FR-008 | T004, T007 |
| FR-009 | T001 (WriteSkill idempotency) |
| FR-010 | T001 (TargetDir), T003 |
| FR-011 | T003 (fmt.Printf) |
