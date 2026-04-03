## Tasks

- [ ] T001: Add `ensureHooks(settingsPath)` function to `init.go` that idempotently adds brains hook entries — `internal/cli/init.go`
- [ ] T002: Call `ensureHooks` from both `initLocal` and `initGlobal` when `claude` is true — `internal/cli/init.go`
- [ ] T003: Add tests for hook installation (fresh, idempotent, preserves existing) — `internal/cli/init_test.go`
- [ ] T004: Verify tests pass — `go test`
