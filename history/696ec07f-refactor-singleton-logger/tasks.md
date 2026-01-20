# Task List: Singleton Logger

**Initiative**: 696ec07f-refactor-singleton-logger | **Total Tasks**: 16 | **Parallel Opportunities**: 4 groups

## Phase 1: Singleton Infrastructure

- [ ] T001 Add singleton variable and InitLogger function to `internal/logging/logger.go`
- [ ] T002 Add Logger() fail-fast getter to `internal/logging/logger.go`
- [ ] T003 Add ResetLogger() test helper to `internal/logging/logger.go`
- [ ] T004 Add unit tests for singleton functions in `internal/logging/logger_test.go`

## Phase 2: Entrypoint Migration

- [ ] T005 [P] Replace SetupLogger with InitLogger in `internal/cli/serve.go`
- [ ] T006 [P] Replace SetupLogger with InitLogger in `internal/cli/gui.go`

## Phase 3: Config Package Migration

- [ ] T007 Remove logger param from LoadConfig, LoadLocalConfig, LoadGlobalConfig, LoadStorageConfig in `internal/config/loader.go`
- [ ] T008 Remove logger param from LoadToolsConfig in `internal/config/tools.go`
- [ ] T009 Update config function calls in `internal/cli/serve.go` (remove logger args)
- [ ] T010 Update config function calls in `internal/cli/gui.go` (remove logger args)

## Phase 4: Web Package Migration

- [ ] T011 Remove logger field from Server struct and NewServer signature in `internal/web/server.go`
- [ ] T012 [P] Replace s.logger with logging.Logger() in `internal/web/middleware.go`
- [ ] T013 [P] Replace s.logger with logging.Logger() in `internal/web/search.go`
- [ ] T014 Remove logger from PluginRegistry and NewPluginRegistry in `internal/web/plugin.go`
- [ ] T015 Update NewServer and NewPluginRegistry calls in `internal/cli/gui.go`

## Phase 5: Helper Functions

- [ ] T016 Update LogToolCall and LogDBOperation signatures (remove logger param) in `internal/logging/logger.go`

## Dependency Graph

```
T001 → T002 → T003 → T004
                ↓
         T005, T006 (parallel)
                ↓
         T007, T008 (parallel)
                ↓
         T009, T010 (parallel)
                ↓
              T011
                ↓
       T012, T013, T014 (parallel)
                ↓
              T015
                ↓
              T016
```

## Verification Checkpoints

After T004: `go test ./internal/logging/...` passes
After T006: `brains serve` and `brains gui` start correctly
After T010: Config loading logs appear with `--log-level debug`
After T015: HTTP request logging works in GUI
After T016: All tests pass (`go test ./...`)

## Suggested Execution Order

1. T001-T004 (sequential, foundation)
2. T005-T006 (parallel, entrypoints)
3. T007-T008 (parallel, config signatures)
4. T009-T010 (parallel, config callers)
5. T011 (web server struct)
6. T012-T014 (parallel, web methods)
7. T015 (cli/gui.go web calls)
8. T016 (helper functions)

## Next Step

Run `/brains.implement` to begin execution.
