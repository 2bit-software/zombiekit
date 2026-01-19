# Reproduction Steps

## Environment

- Platform: darwin (macOS)
- Task runner: task v3
- Shell: zsh/bash

## Prerequisites

1. Docker running (for PostgreSQL)
2. Ollama running with nomic-embed-text model
3. Built binary at `./bin/brains`

## Steps to Reproduce

1. Run `task up`
2. Wait for stack to start (preflight, build, db:up, db:migrate)
3. Observe the importer starts in background
4. Press Ctrl+C
5. Check for lock file: `ls -la ~/.claude/.zombiekit-import.lock`

## Expected Behavior

- Lock file should be removed on shutdown
- Clean exit with "Stopping importer..." message

## Actual Behavior

- Lock file persists after Ctrl+C
- Subsequent runs may encounter lock issues

## Analysis

Looking at the code flow:

1. `Taskfile.yml:up` runs importer in background with `&`
2. Taskfile sets trap: `trap "echo 'Stopping importer...'; kill $IMPORTER_PID 2>/dev/null" EXIT INT TERM`
3. `kill $IMPORTER_PID` sends SIGTERM to the Go process
4. In `recallWatchClaudeAction` (recall.go:351-375), signal handling listens for SIGTERM/SIGINT
5. BUT: The lock is acquired in `importClaudeHistory` (recall.go:425-429), which is called inside the ticker loop
6. When import completes, `defer lock.Release()` runs, releasing the lock
7. HOWEVER: When signal arrives during idle wait (between imports), there's no lock held

Wait - the lock is acquired per-import call, not for the entire watch session. Let me re-examine...

The actual issue: When `task up` trap sends `kill $IMPORTER_PID`, it sends SIGTERM. The Go process catches it in the select statement (line 371), prints "Shutting down..." and returns. The return triggers all defers including `storage.Close()` but NOT `lock.Release()` because the lock is acquired inside `importClaudeHistory` which may not be running at that moment.

The lock is scoped to each import operation, not the watch loop. If signal arrives while waiting on ticker (not during import), there's no lock to release - which is correct.

The REAL issue might be different: Does the lock file get deleted on Release()? Let me check...

Looking at lock.go:52-64, `Release()` only unlocks and closes the file - it does NOT delete the file. The comment says "The lock is automatically released when the process terminates" (line 27) - this is true for flock(), but the file remains.

The bug: The lock FILE persists because it's never deleted - only the flock is released.
