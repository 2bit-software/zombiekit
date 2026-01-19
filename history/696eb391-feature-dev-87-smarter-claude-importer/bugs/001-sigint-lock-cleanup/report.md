# Bug Report: SIGINT Lock File Cleanup

## Summary

When running `task up` and pressing Ctrl+C, the import watcher does not clean up its lock file.

## Symptoms

- Lock file persists after Ctrl+C
- Subsequent runs may fail or behave unexpectedly due to stale lock

## Context

- Command: `task up`
- Signal: SIGINT (Ctrl+C)
- Expected: Lock file should be removed on graceful shutdown
- Actual: Lock file remains

## Environment

- Platform: darwin
- Task runner invokes `./bin/brains recall watch claude --verbose` in background
- Taskfile trap: `trap "echo 'Stopping importer...'; kill $IMPORTER_PID 2>/dev/null" EXIT INT TERM`

## Related Code

- Watch command implementation
- Lock file management in recall importer
