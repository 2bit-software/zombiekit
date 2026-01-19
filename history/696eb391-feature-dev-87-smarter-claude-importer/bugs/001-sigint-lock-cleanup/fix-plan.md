# Fix Plan

## Problem Summary

The `task up` command leaves orphaned importer processes because:
1. `trap "..." INT TERM` uses bash-style signal names
2. zsh (macOS default) doesn't recognize `INT`/`TERM` without `SIG` prefix
3. Background process isn't killed on Ctrl+C
4. Orphaned process holds flock, blocking subsequent runs

## Recommended Fix

Use portable signal syntax in the trap command.

### Changes Required

**File:** `Taskfile.yml`

**Before:**
```yaml
trap "echo 'Stopping importer...'; kill $IMPORTER_PID 2>/dev/null" EXIT INT TERM
```

**After:**
```yaml
trap 'echo "Stopping importer..."; kill $IMPORTER_PID 2>/dev/null' EXIT SIGINT SIGTERM
```

Note: Also changed double quotes to single quotes on outer trap to prevent premature variable expansion.

## Alternative: Swap foreground/background

Keep importer in foreground (receives Ctrl+C directly), run GUI in background:

```yaml
# Start GUI in background
./bin/brains gui --port ${WEBGUI_PORT:-9981} &
GUI_PID=$!

trap 'echo "Stopping..."; kill $GUI_PID 2>/dev/null' EXIT SIGINT SIGTERM

# Run importer in foreground (will catch Ctrl+C directly)
./bin/brains recall watch claude --verbose
```

This is cleaner because:
- Importer's Go signal handling works directly
- No need for shell-level signal forwarding
- Importer output streams to terminal naturally

## Testing

1. Apply fix
2. Run `task up`
3. Wait for stack to start
4. Press Ctrl+C
5. Verify: `ps aux | grep "brains recall"` shows no processes
6. Verify: `task up` runs again without lock error

## Verification Commands

```bash
# Check no orphaned processes
ps aux | grep "brains recall" | grep -v grep

# Check lock is acquirable
export $(grep -v '^#' .env | xargs) && ./bin/brains recall watch claude --once
```
