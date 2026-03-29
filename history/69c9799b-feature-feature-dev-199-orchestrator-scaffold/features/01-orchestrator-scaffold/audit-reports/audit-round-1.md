# Audit Report: Round 1

## Summary

Two parallel audits (completeness + AI-friendliness) identified 4 CRITICAL, 8 MAJOR, and 6 MINOR findings. All CRITICAL and MAJOR findings have been addressed in the updated spec.

## CRITICAL (Fixed)

1. **Callback server already exists** — Spec described a stub; `internal/callback/` has a full implementation with routes, event channel, and graceful shutdown. Fixed: spec now references existing implementation.

2. **Reconciliation already exists** — Spec described a stub; `internal/state/reconcile.go` has `ApplyReconciliation()` with real logic. Fixed: spec now calls existing function.

3. **Signal handling conflict** — `shutdown.Manager.Run()` handles SIGINT/SIGTERM internally. Following the zk-server pattern (`signal.NotifyContext`) would create competing handlers. Fixed: spec explicitly says not to use `signal.NotifyContext`.

4. **ctx parameter misleading** — `shutdown.Manager.Run()` creates its own context; a passed-in ctx has no effect on it. Fixed: spec documents context lifecycle — `ctx` is for pre-service steps only.

## MAJOR (Fixed)

5. **Constructor vs startup contradiction** — `NewOrchestrator(cfg, store)` implied store opened externally, but startup sequence said `Run` opens it. Fixed: eliminated `Orchestrator` struct entirely — `run()` in main.go wires the sequence directly.

6. **Watcher return value unspecified** — Returning `ctx.Err()` would cause errgroup to treat shutdown as failure. Fixed: spec mandates `return nil`.

7. **AC-7 port-in-use not truly fail-fast** — Port bind happens inside errgroup after watchers start. Fixed: relaxed AC-7 to say error "propagates through the shutdown manager."

8. **Error message format unspecified** — Fixed: spec gives example format and mandates collecting all errors.

9. **Worktrees root creation unspecified** — Fixed: created during config validation via `os.MkdirAll`.

10. **Run return value unspecified** — Fixed: added Return Values section under Graceful Shutdown.

11. **InitLogger panics on double-call** — Fixed: spec notes the constraint and test cleanup requirement.

12. **StateStore type qualification** — Fixed: constructor uses `state.StateStore` explicitly (moot now since no Orchestrator struct).

## MINOR (Fixed)

- Added `ORCH_` prefix rationale
- Added `Version: version.Get().Short()`
- Specified `cli.DurationFlag` for durations
- Specified slog key/value format for watcher log messages
- Noted event channel consumption deferred to future ticket
- Specified valid log levels with fail-fast on invalid

## Architectural Decision: Orchestrator Struct with DI (Approved)

After discussion, we opted for an `Orchestrator` struct in `internal/orchestrator/` that accepts `state.StateStore` (interface) and owns the lifecycle: reconciliation -> services -> shutdown. `main.go`'s `run()` is thin glue that owns resource acquisition (logger init, store open/close). This split gives unit-testable lifecycle logic while keeping `main.go` minimal.

Decision approved by user — testability was the deciding factor.
