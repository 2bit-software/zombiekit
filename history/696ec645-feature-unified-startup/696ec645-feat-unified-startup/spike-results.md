---
status: validated
date: 2026-01-19
---

# Spike Results: Signal Handling Pattern

## Objective

Validate a robust pattern for:
1. Capturing SIGINT (Ctrl+C) and SIGTERM
2. Propagating shutdown to multiple services via context
3. 10-second timeout before force exit
4. Double Ctrl+C for immediate force exit

## Validated Pattern

```go
// 1. Create cancellable context
ctx, cancel := context.WithCancel(context.Background())

// 2. Buffered signal channel (capacity 2 for double-signal)
sigCh := make(chan os.Signal, 2)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

// 3. errgroup for service coordination
g, ctx := errgroup.WithContext(ctx)

// 4. Launch services (each respects ctx.Done())
g.Go(func() error { return runService(ctx, "gui") })
g.Go(func() error { return runService(ctx, "recall") })

// 5. Signal handler as goroutine in errgroup
g.Go(func() error {
    select {
    case <-sigCh:
        cancel()  // Propagate to all services
        return nil
    case <-ctx.Done():
        return nil
    }
})

// 6. Wait with timeout and second-signal handling
done := make(chan error, 1)
go func() { done <- g.Wait() }()

select {
case err := <-done:
    // Clean shutdown
case <-time.After(shutdownTimeout):
    os.Exit(1)  // Timeout
case <-sigCh:
    os.Exit(1)  // Force exit (second signal)
}
```

## Test Results

All 6 tests pass:

| Test | Description | Result |
|------|-------------|--------|
| `TestGracefulShutdownWithTimeout` | Context propagates, services complete within timeout | PASS |
| `TestSlowServiceHitsTimeout` | Timeout fires when service exceeds limit | PASS |
| `TestDoubleSignalForceExit` | Second Ctrl+C triggers force exit path | PASS |
| `TestContextPropagation` | Cancellation reaches all nested goroutines | PASS |
| `TestSignalBufferCapacity` | Channel buffers multiple signals without loss | PASS |
| `TestErrgroupErrorPropagation` | Service error cancels all other services | PASS |

## Key Findings

1. **Buffered channel capacity 2 is essential** - prevents signal loss during the window between first and second signal handling

2. **errgroup.WithContext provides automatic error propagation** - if any service errors during startup, all others receive context cancellation

3. **Signal handler must be part of errgroup** - otherwise it can outlive the services and cause coordination issues

4. **Each service must check ctx.Done() in its main loop** - no exceptions, or shutdown hangs

5. **Timeout context for shutdown is separate** - don't reuse the canceled signal context for cleanup operations (e.g., `http.Server.Shutdown`)

## Gaps Addressed

The research identified these concerns, which the spike validates:

| Concern | How Addressed |
|---------|--------------|
| Signal loss during initialization | Buffered channel capacity 2 |
| Multiple services coordination | errgroup.WithContext |
| Timeout enforcement | select with time.After |
| Double Ctrl+C | Second read from sigCh in final select |
| Context propagation | errgroup's derived context |

## Spike Files

- `spike/signal-handling/main.go` - runnable demo
- `spike/signal-handling/main_test.go` - automated validation

## Recommendation

Adopt this pattern for the unified startup command. The pattern is:
- Battle-tested (Kubernetes, Consul use similar)
- Well-tested in our spike
- Integrates cleanly with errgroup for multi-service orchestration
