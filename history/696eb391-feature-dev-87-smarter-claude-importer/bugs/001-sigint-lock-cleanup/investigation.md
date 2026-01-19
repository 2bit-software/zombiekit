# Investigation

## Actual Issue Discovered

The user reported:
```
trap: INT: invalid signal specification
Stopping importer...
import already in progress (another process holds the lock)
```

**Root cause:** `trap "..." EXIT INT TERM` is bash syntax. When Task runs with zsh (default macOS shell), `INT` is invalid - zsh uses `SIGINT`.

This causes:
1. Trap fails to register for SIGINT
2. Background importer process isn't killed on Ctrl+C
3. Orphaned process continues holding flock
4. Subsequent runs fail with lock error

## Evidence

```bash
$ ps aux | grep "brains recall"
19963 ./bin/brains recall watch claude --verbose  # Orphaned!
19704 ./bin/brains recall watch claude --verbose  # Another orphan!
```

Two orphaned watch processes were running, both holding the lock.

## Code Flow Analysis

`Taskfile.yml:55-59`:
```yaml
./bin/brains recall watch claude --verbose &
IMPORTER_PID=$!
trap "echo 'Stopping importer...'; kill $IMPORTER_PID 2>/dev/null" EXIT INT TERM
```

The trap syntax uses bash-style signal names (`INT`, `TERM`) which fail on zsh.

## Classification: **Implementation Error**

The Taskfile uses non-portable signal names in the trap command.

## Solutions

### Option A: Use signal numbers (portable)
```bash
trap "..." EXIT 2 15  # 2=SIGINT, 15=SIGTERM
```

### Option B: Use SIG prefix (works in both)
```bash
trap "..." EXIT SIGINT SIGTERM
```

### Option C: Force bash shell
```yaml
set: [errexit, pipefail]
shopt: [globstar]
```
But Task doesn't have a shell directive.

### Option D: Use job control differently
Start GUI in background, keep importer in foreground (catches signals directly).

### Option E: Let Go handle signals (recommended)
Remove bash trap entirely. The Go process already handles SIGINT/SIGTERM in `recallWatchClaudeAction`. Just need to ensure it receives the signal.

The issue is that when running in background with `&`, the subprocess doesn't receive SIGINT from Ctrl+C (only foreground process group does).
