# Technical Specification: cmux Session Manager

## Package: `internal/cmux`

### Type Definitions

```go
// SessionManager manages cmux workspace lifecycles for agent sessions.
type SessionManager interface {
    SpawnSession(ctx context.Context, ticketID, title, worktreePath string, env map[string]string) (workspaceRef string, err error)
    KillSession(ctx context.Context, ticketID string) error
    SessionExists(ctx context.Context, ticketID string) (bool, error)
}

// CmuxManager implements SessionManager by shelling out to the cmux CLI.
// The mutex serializes all mutating operations (SpawnSession, KillSession).
// This is a throughput trade-off: concurrent spawns for different tickets
// block each other, but spawn operations are infrequent (~seconds apart)
// and correctness requires atomicity across the check-create-rename sequence.
type CmuxManager struct {
    cmuxBin  string
    command  string // launch command, default "claude"
    mu       sync.Mutex
    sessions map[string]sessionEntry // ticketID -> entry
}

type sessionEntry struct {
    ref  string // e.g. "workspace:9"
    name string // e.g. "DEV-186: implement session manager"
}

// Option configures a CmuxManager.
type Option func(*CmuxManager)

// WithCommand overrides the default launch command (default: "claude").
func WithCommand(cmd string) Option {
    return func(m *CmuxManager) {
        m.command = cmd
    }
}
```

### Error Types

```go
type ErrorKind int

const (
    ErrSessionExists   ErrorKind = iota + 1
    ErrSessionNotFound
    ErrCmuxUnavailable
    ErrBinaryNotFound
    ErrCommandFailed
    ErrInvalidEnvKey
)

type Error struct {
    Kind    ErrorKind
    Message string
    Err     error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }

// classifyError maps cmux stderr to ErrorKind.
func classifyError(stderr string) ErrorKind {
    switch {
    case strings.Contains(stderr, "not_found"):
        return ErrSessionNotFound
    case strings.Contains(stderr, "connection refused"),
         strings.Contains(stderr, "No such file"),
         strings.Contains(stderr, "could not connect"):
        return ErrCmuxUnavailable
    default:
        return ErrCommandFailed
    }
}
```

### Constructor

```go
func New(opts ...Option) (*CmuxManager, error) {
    cmuxBin, err := exec.LookPath("cmux")
    if err != nil {
        return nil, newError(ErrBinaryNotFound, "cmux not found on PATH", err)
    }

    m := &CmuxManager{
        cmuxBin:  cmuxBin,
        command:  "claude",
        sessions: make(map[string]sessionEntry),
    }

    for _, opt := range opts {
        opt(m)
    }

    // Validate cmux is running
    if _, err := m.run(context.Background(), "ping"); err != nil {
        return nil, newError(ErrCmuxUnavailable, "cmux is not running or unreachable", err)
    }

    return m, nil
}
```

### Command Execution

```go
func (_m *CmuxManager) run(ctx context.Context, args ...string) (string, error) {
    cmd := exec.CommandContext(ctx, _m.cmuxBin, args...)

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        stderrStr := strings.TrimSpace(stderr.String())
        // cmux writes errors to stderr (verified via spike)
        if stderrStr == "" {
            stderrStr = strings.TrimSpace(stdout.String()) // fallback
        }
        kind := classifyError(stderrStr)
        return "", newError(kind, fmt.Sprintf("cmux %s: %s", args[0], stderrStr), err)
    }

    return strings.TrimSpace(stdout.String()), nil
}
```

### Output Parsing

```go
// parseNewWorkspace extracts workspace ref from "OK workspace:N".
func parseNewWorkspace(stdout string) (string, error) {
    // Expected: "OK workspace:9"
    parts := strings.Fields(stdout)
    if len(parts) != 2 || parts[0] != "OK" || !strings.HasPrefix(parts[1], "workspace:") {
        return "", fmt.Errorf("unexpected new-workspace output: %q", stdout)
    }
    return parts[1], nil
}

type workspaceEntry struct {
    ref      string
    name     string
    selected bool
}

// parseListWorkspaces parses cmux list-workspaces plain text output.
// Format per line: "[*] workspace:N  name  [selected]"
// Returns error if non-empty input produces zero valid entries (format change detection).
func parseListWorkspaces(stdout string) ([]workspaceEntry, error) {
    var entries []workspaceEntry
    var nonEmptyLines int
    for _, line := range strings.Split(stdout, "\n") {
        line = strings.TrimSpace(line)
        if line == "" {
            continue
        }
        nonEmptyLines++

        selected := false
        if strings.HasPrefix(line, "* ") {
            selected = true
            line = line[2:]
        } else {
            line = strings.TrimPrefix(line, "  ")
        }

        // Split on first double-space to separate ref from name
        parts := strings.SplitN(line, "  ", 2)
        if len(parts) < 2 {
            continue
        }

        ref := strings.TrimSpace(parts[0])
        name := strings.TrimSpace(parts[1])
        name = strings.TrimSuffix(name, "[selected]")
        name = strings.TrimSpace(name)

        entries = append(entries, workspaceEntry{
            ref:      ref,
            name:     name,
            selected: selected,
        })
    }

    // Detect format changes: non-empty input but zero parsed entries
    if nonEmptyLines > 0 && len(entries) == 0 {
        return nil, fmt.Errorf("failed to parse list-workspaces output (%d lines, 0 entries): format may have changed", nonEmptyLines)
    }

    return entries, nil
}

// findByTicketID searches workspace entries for a name starting with "{ticketID}: ".
func findByTicketID(entries []workspaceEntry, ticketID string) *workspaceEntry {
    prefix := ticketID + ": "
    for i := range entries {
        if strings.HasPrefix(entries[i].name, prefix) {
            return &entries[i]
        }
    }
    return nil
}
```

### Shell Escaping

```go
var validEnvKey = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// buildCommand constructs the shell command string with exported env vars.
// Values are single-quote escaped. Empty env produces just the command.
func buildCommand(env map[string]string, cmd string) (string, error) {
    if len(env) == 0 {
        return cmd, nil
    }

    var exports []string
    for k, v := range env {
        if !validEnvKey.MatchString(k) {
            return "", newError(ErrInvalidEnvKey,
                fmt.Sprintf("invalid env key: %q", k), nil)
        }
        escaped := "'" + strings.ReplaceAll(v, "'", "'\\''") + "'"
        exports = append(exports, k+"="+escaped)
    }

    // Sort for deterministic output (testability)
    sort.Strings(exports)

    return "export " + strings.Join(exports, " ") + " && " + cmd, nil
}
```

### SpawnSession Flow

```go
func (_m *CmuxManager) SpawnSession(ctx context.Context, ticketID, title, worktreePath string, env map[string]string) (string, error) {
    _m.mu.Lock()
    defer _m.mu.Unlock()

    // Check internal tracking
    if _, exists := _m.sessions[ticketID]; exists {
        return "", newError(ErrSessionExists,
            fmt.Sprintf("session already tracked for %s", ticketID), nil)
    }

    // Check live cmux state
    listOut, err := _m.run(ctx, "list-workspaces")
    if err != nil {
        return "", err
    }
    entries, err := parseListWorkspaces(listOut)
    if err != nil {
        return "", newError(ErrCommandFailed, err.Error(), err)
    }
    if found := findByTicketID(entries, ticketID); found != nil {
        return "", newError(ErrSessionExists,
            fmt.Sprintf("cmux workspace already exists for %s: %s", ticketID, found.ref), nil)
    }

    // Build command string
    cmdStr, err := buildCommand(env, _m.command)
    if err != nil {
        return "", err
    }

    // Create workspace
    createOut, err := _m.run(ctx, "new-workspace", "--cwd", worktreePath, "--command", cmdStr)
    if err != nil {
        return "", err
    }
    ref, err := parseNewWorkspace(createOut)
    if err != nil {
        return "", newError(ErrCommandFailed, err.Error(), err)
    }

    // Set display name
    name := ticketID + ": " + title
    if _, err := _m.run(ctx, "rename-workspace", "--workspace", ref, name); err != nil {
        // Workspace was created but rename failed -- close it to avoid orphan
        _ = _m.run(ctx, "close-workspace", "--workspace", ref) //nolint:errcheck
        return "", err
    }

    // Track
    _m.sessions[ticketID] = sessionEntry{ref: ref, name: name}
    return ref, nil
}
```

### KillSession Flow

```go
func (_m *CmuxManager) KillSession(ctx context.Context, ticketID string) error {
    _m.mu.Lock()
    defer _m.mu.Unlock()

    entry, exists := _m.sessions[ticketID]
    if !exists {
        return newError(ErrSessionNotFound,
            fmt.Sprintf("no tracked session for %s", ticketID), nil)
    }

    if _, err := _m.run(ctx, "close-workspace", "--workspace", entry.ref); err != nil {
        return err
    }

    delete(_m.sessions, ticketID)
    return nil
}
```

### SessionExists Flow

```go
func (_m *CmuxManager) SessionExists(ctx context.Context, ticketID string) (bool, error) {
    // Check live cmux state (authoritative)
    listOut, err := _m.run(ctx, "list-workspaces")
    if err != nil {
        return false, err // Do NOT return false, nil on cmux failure
    }

    entries, err := parseListWorkspaces(listOut)
    if err != nil {
        return false, err // Unparseable output is an error, not "doesn't exist"
    }

    if findByTicketID(entries, ticketID) != nil {
        return true, nil
    }

    // Reconcile: if we were tracking it but cmux says it's gone, clean up
    _m.mu.Lock()
    if _, tracked := _m.sessions[ticketID]; tracked {
        delete(_m.sessions, ticketID)
    }
    _m.mu.Unlock()

    return false, nil
}
```

## File Layout

```
internal/cmux/
  doc.go           — package docs, usage example
  types.go         — SessionManager interface, CmuxManager, Option, sessionEntry
  errors.go        — ErrorKind, Error, classifyError, Is* helpers, newError
  manager.go       — New, SpawnSession, KillSession, SessionExists, run
  parse.go         — parseNewWorkspace, parseListWorkspaces, findByTicketID, buildCommand
  manager_test.go  — integration tests (skip without cmux)
  parse_test.go    — unit tests for parsing and command building
```
