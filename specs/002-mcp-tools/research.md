# Research: MCP Tools - Code Reasoning & Sticky Memory

**Date**: 2025-12-21
**Branch**: `002-mcp-tools`
**Compatibility**: mcp-genie (`telegraph/ai/tools/mcp-genie`)

## Summary

This document captures research findings for implementing the code-reasoning and sticky-memory MCP tools. All NEEDS CLARIFICATION items from the Technical Context have been resolved.

**Key Decision**: SQLite implementation MUST be compatible with mcp-genie's stickymemory patterns to enable code sharing and consistent behavior.

---

## 1. MCP Server Implementation (mark3labs/mcp-go)

### Decision: Use mark3labs/mcp-go for MCP protocol implementation

**Rationale**: Only mature Go MCP library, recommended in MASTER-DESIGN.md, minimal boilerplate.

**Alternatives Considered**:
- Hand-roll MCP protocol: Too much effort, error-prone
- Different language: Not compatible with existing Go codebase

### Key Patterns

#### Server Creation

```go
s := server.NewMCPServer(
    "brains",   // Server name
    "1.0.0",    // Version
)
```

#### Tool Definition

```go
tool := mcp.NewTool("stickymemory",
    mcp.WithDescription("Persistent key-value storage for AI assistants"),
    mcp.WithString("operation",
        mcp.Required(),
        mcp.Enum("get", "set", "list", "delete", "search", "clear"),
    ),
    mcp.WithString("name"),    // Optional for list/clear
    mcp.WithString("content"), // Required for set
    mcp.WithNumber("limit"),   // Optional for list/search
)

s.AddTool(tool, handleStickyMemory)
```

#### Tool Handler Pattern

```go
func handleStickyMemory(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    op, err := req.RequireString("operation")
    if err != nil {
        return nil, err
    }

    switch op {
    case "get":
        name, err := req.RequireString("name")
        if err != nil {
            return nil, err
        }
        // ... implementation
        return mcp.NewToolResultText(content), nil
    case "set":
        // ...
    }
}
```

#### Transport Modes

```go
// stdio (default, for Claude Desktop)
server.ServeStdio(s)

// HTTP with Streamable HTTP (default for web)
server.ServeHTTP(s, ":8080")

// SSE (legacy compatibility)
server.ServeSSE(s, ":8080")
```

### Error Handling

- Return `error` from handler for MCP protocol errors
- Use `mcp.NewToolResultError(msg)` for tool-level errors that should be shown to the user
- Validation errors (RequireString, etc.) are handled by the library

### Testing Pattern

```go
func TestStickyMemoryTool(t *testing.T) {
    // Create mock repository
    repo := memory.NewMockRepository()

    // Create tool with injected dependency
    tool := stickymemory.NewTool(repo)

    // Call handler directly
    result, err := tool.Handle(ctx, mcp.CallToolRequest{
        Params: map[string]any{
            "operation": "set",
            "name": "test",
            "content": "value",
        },
    })

    assert.NoError(t, err)
    assert.Contains(t, result.Content, "success")
}
```

---

## 2. PostgreSQL with pgx/v5

### Decision: Use pgxpool for connection management

**Rationale**: Native PostgreSQL driver, better performance than database/sql, built-in connection pooling.

**Alternatives Considered**:
- database/sql: Less performant, requires separate pool management
- sqlx: Adds dependency without significant benefit for our use case

### Connection Pool Configuration

```go
config, err := pgxpool.ParseConfig(os.Getenv("DATABASE_URL"))
if err != nil {
    return nil, fmt.Errorf("parse database URL: %w", err)
}

// Conservative defaults for local tool
config.MaxConns = 10
config.MinConns = 1
config.MaxConnLifetime = time.Hour
config.MaxConnIdleTime = 30 * time.Minute

pool, err := pgxpool.NewWithConfig(ctx, config)
if err != nil {
    return nil, fmt.Errorf("create pool: %w", err)
}

// Verify connectivity immediately (fail-fast per clarification)
if err := pool.Ping(ctx); err != nil {
    return nil, fmt.Errorf("database connection failed: %w", err)
}
```

### Transaction Pattern (BeginFunc)

```go
err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
    // All operations in this block are in a transaction
    _, err := tx.Exec(ctx, `
        INSERT INTO memories (name, content, version)
        VALUES ($1, $2, 1)
        ON CONFLICT (name) DO UPDATE
        SET content = $2, version = memories.version + 1, updated_at = NOW()
    `, name, content)
    return err
})
// Transaction auto-commits on nil, auto-rollbacks on error
```

### Error Handling

```go
import "github.com/jackc/pgx/v5/pgconn"

func handleDBError(err error) error {
    var pgErr *pgconn.PgError
    if errors.As(err, &pgErr) {
        switch pgErr.Code {
        case "23505": // unique_violation
            return ErrDuplicateName
        case "23503": // foreign_key_violation
            return ErrInvalidReference
        }
    }

    // Connection errors
    if errors.Is(err, context.DeadlineExceeded) {
        return ErrTimeout
    }

    return fmt.Errorf("database error: %w", err)
}
```

### Migration Pattern

```go
//go:embed migrations/*.sql
var migrationsFS embed.FS

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
    entries, _ := migrationsFS.ReadDir("migrations")

    // Ensure migrations table exists
    _, err := pool.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version INTEGER PRIMARY KEY,
            applied_at TIMESTAMPTZ DEFAULT NOW()
        )
    `)
    if err != nil {
        return err
    }

    for _, entry := range entries {
        version := extractVersion(entry.Name())

        // Check if already applied
        var exists bool
        pool.QueryRow(ctx,
            "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)",
            version,
        ).Scan(&exists)

        if exists {
            continue
        }

        // Apply migration in transaction
        sql, _ := migrationsFS.ReadFile("migrations/" + entry.Name())
        err := pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
            _, err := tx.Exec(ctx, string(sql))
            if err != nil {
                return err
            }
            _, err = tx.Exec(ctx,
                "INSERT INTO schema_migrations (version) VALUES ($1)",
                version,
            )
            return err
        })
        if err != nil {
            return fmt.Errorf("migration %d: %w", version, err)
        }
    }
    return nil
}
```

---

## 3. Structured Logging with slog

### Decision: Use Go 1.22+ slog with JSON handler

**Rationale**: Built into stdlib, structured by default, configurable levels, no external dependency.

### Setup Pattern

```go
var logLevel = new(slog.LevelVar) // allows runtime changes

func SetupLogger(level string, jsonOutput bool) *slog.Logger {
    // Parse level
    switch strings.ToLower(level) {
    case "debug":
        logLevel.Set(slog.LevelDebug)
    case "info":
        logLevel.Set(slog.LevelInfo)
    case "warn":
        logLevel.Set(slog.LevelWarn)
    case "error":
        logLevel.Set(slog.LevelError)
    default:
        logLevel.Set(slog.LevelInfo)
    }

    opts := &slog.HandlerOptions{
        Level: logLevel,
    }

    var handler slog.Handler
    if jsonOutput {
        handler = slog.NewJSONHandler(os.Stderr, opts)
    } else {
        handler = slog.NewTextHandler(os.Stderr, opts)
    }

    return slog.New(handler)
}
```

### Request Logging Pattern

```go
func logToolCall(logger *slog.Logger, toolName string, start time.Time, err error) {
    duration := time.Since(start)

    attrs := []slog.Attr{
        slog.String("tool", toolName),
        slog.Duration("duration", duration),
    }

    if err != nil {
        attrs = append(attrs, slog.String("error", err.Error()))
        logger.LogAttrs(context.Background(), slog.LevelError, "tool call failed", attrs...)
    } else {
        logger.LogAttrs(context.Background(), slog.LevelInfo, "tool call completed", attrs...)
    }
}
```

### Context Integration

```go
// Add logger to context for request-scoped logging
type ctxKey struct{}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
    return context.WithValue(ctx, ctxKey{}, logger)
}

func LoggerFrom(ctx context.Context) *slog.Logger {
    if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
        return l
    }
    return slog.Default()
}
```

---

## 4. Testing with testcontainers-go

### Decision: Use testcontainers-go for PostgreSQL integration tests

**Rationale**: Provides real PostgreSQL in tests, no mocking required, reproducible environments.

### Setup Pattern

```go
func TestWithPostgres(t *testing.T) {
    ctx := context.Background()

    container, err := postgres.Run(ctx,
        "postgres:16-alpine",
        postgres.WithDatabase("test"),
        postgres.WithUsername("test"),
        postgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(30*time.Second),
        ),
    )
    require.NoError(t, err)
    defer container.Terminate(ctx)

    connStr, err := container.ConnectionString(ctx, "sslmode=disable")
    require.NoError(t, err)

    // Create pool
    pool, err := pgxpool.New(ctx, connStr)
    require.NoError(t, err)
    defer pool.Close()

    // Run migrations
    err = database.RunMigrations(ctx, pool)
    require.NoError(t, err)

    // Run tests
    t.Run("memory operations", func(t *testing.T) {
        repo := memory.NewPostgresRepository(pool)
        // ... test cases
    })
}
```

### Parallel Test Isolation

```go
func TestMemoryRepository(t *testing.T) {
    pool := setupTestDB(t) // shared container

    t.Run("parallel tests", func(t *testing.T) {
        t.Run("set and get", func(t *testing.T) {
            t.Parallel()
            repo := memory.NewPostgresRepository(pool)

            // Use unique name per test to avoid conflicts
            name := fmt.Sprintf("test-%s", t.Name())
            // ...
        })

        t.Run("list", func(t *testing.T) {
            t.Parallel()
            // ...
        })
    })
}
```

---

## 5. CLI Structure (urfave/cli/v2)

### Decision: Use existing urfave/cli/v2 from project

**Rationale**: Already in use, lightweight, consistent with codebase.

### Command Structure

```go
func newServeCommand() *cli.Command {
    return &cli.Command{
        Name:  "serve",
        Usage: "Start the MCP server",
        Flags: []cli.Flag{
            &cli.StringFlag{
                Name:    "mode",
                Value:   "http",
                Usage:   "Transport mode: http, sse, stdio",
                EnvVars: []string{"BRAINS_MCP_MODE"},
            },
            &cli.IntFlag{
                Name:    "port",
                Value:   8080,
                Usage:   "Port for HTTP-based transports",
                EnvVars: []string{"BRAINS_MCP_PORT"},
            },
            &cli.StringFlag{
                Name:    "log-level",
                Value:   "info",
                Usage:   "Log level: debug, info, warn, error",
                EnvVars: []string{"BRAINS_LOG_LEVEL"},
            },
        },
        Action: runServe,
    }
}

func newMemoryCommand() *cli.Command {
    return &cli.Command{
        Name:  "memory",
        Usage: "Manage sticky memories",
        Subcommands: []*cli.Command{
            {Name: "list", Action: memoryList},
            {Name: "get", Action: memoryGet},
            {Name: "set", Action: memorySet},
            {Name: "delete", Action: memoryDelete},
            {Name: "search", Action: memorySearch},
            {Name: "clear", Action: memoryClear},
        },
    }
}
```

---

## 6. Code Reasoning Implementation

### Decision: In-memory session state (no persistence)

**Rationale**: Per spec assumption - "Code reasoning state is session-scoped (in-memory) and does not persist across server restarts."

### Session Design

```go
type Session struct {
    mu           sync.RWMutex
    thoughts     []Thought
    branches     map[string][]Thought
    totalThoughts int
    completed    bool
}

type Thought struct {
    Number      int
    Content     string
    IsRevision  bool
    RevisesNum  int
    BranchID    string
    CreatedAt   time.Time
}

// Thread-safe operations
func (s *Session) AddThought(thought Thought) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if s.completed {
        return ErrSessionCompleted
    }

    // Validate thought number
    if thought.Number != len(s.thoughts)+1 {
        return fmt.Errorf("expected thought %d, got %d", len(s.thoughts)+1, thought.Number)
    }

    // Handle branching
    if thought.BranchID != "" {
        if thought.IsRevision {
            return ErrCannotReviseAndBranch
        }
        s.branches[thought.BranchID] = append(s.branches[thought.BranchID], thought)
        return nil
    }

    // Handle revision
    if thought.IsRevision {
        if thought.RevisesNum < 1 || thought.RevisesNum > len(s.thoughts) {
            return ErrInvalidRevisionTarget
        }
        s.thoughts[thought.RevisesNum-1] = thought
        return nil
    }

    s.thoughts = append(s.thoughts, thought)
    return nil
}
```

### Session Manager (per-connection isolation)

```go
type SessionManager struct {
    mu       sync.RWMutex
    sessions map[string]*Session
}

func (m *SessionManager) GetOrCreate(id string) *Session {
    m.mu.Lock()
    defer m.mu.Unlock()

    if s, ok := m.sessions[id]; ok {
        return s
    }

    s := &Session{
        branches: make(map[string][]Thought),
    }
    m.sessions[id] = s
    return s
}
```

---

## Resolved Clarifications

| Item | Resolution |
|------|------------|
| Authentication | None - local-only service (per clarification) |
| Rate limiting | None - local-only, trusted clients (per clarification) |
| Retry logic | Fail-fast on DB errors (per clarification) |
| Logging format | Structured JSON with configurable levels (per clarification) |
| Memory name chars | Identifier-safe: a-z, A-Z, 0-9, -, _, . (per clarification) |

---

## Dependencies to Add

```go
require (
    github.com/mark3labs/mcp-go v0.x.x
    github.com/jackc/pgx/v5 v5.x.x
    modernc.org/sqlite v1.x.x        // Pure Go SQLite
    github.com/testcontainers/testcontainers-go v0.x.x
    github.com/stretchr/testify v1.x.x
)
```

---

## 7. mcp-genie Compatibility (SQLite Implementation)

### Decision: Match mcp-genie's stickymemory implementation

**Rationale**: Ensures consistent behavior between zombiekit/brains and mcp-genie. Enables potential code sharing in the future.

**Reference**: `telegraph/ai/tools/mcp-genie/pkg/tools/stickymemory/`

### Key Patterns from mcp-genie

#### Storage Interface

```go
// From mcp-genie/pkg/tools/stickymemory/storage.go
type Storage interface {
    Set(ctx context.Context, name, content string) error
    Get(ctx context.Context, name string) (mo.Maybe[MemoryItem], error)
    Delete(ctx context.Context, name string) error
    List(ctx context.Context, search string) ([]MemoryMetadata, error)
    Clear(ctx context.Context) (int, error)
    Close() error
}
```

#### SQLite Schema (Append-Only Versioning)

```sql
-- From mcp-genie/pkg/tools/stickymemory/sqlite_storage.go
CREATE TABLE IF NOT EXISTS memories (
    name TEXT NOT NULL,
    version INTEGER NOT NULL,
    content TEXT NOT NULL,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    PRIMARY KEY (name, version)
);
```

**Key Schema Difference**: mcp-genie uses `(name, version)` as composite primary key, NOT a separate UUID id field. Each `Set` operation creates a NEW row with an incremented version number.

#### SQLite Storage Implementation

```go
// From mcp-genie sqlite_storage.go

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
    // Expand home directory if needed
    if strings.HasPrefix(dbPath, "~/") {
        homeDir, _ := os.UserHomeDir()
        dbPath = filepath.Join(homeDir, dbPath[2:])
    }

    // Ensure parent directory exists
    os.MkdirAll(filepath.Dir(dbPath), 0o755)

    // Open database (modernc.org/sqlite driver)
    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return nil, err
    }

    storage := &SQLiteStorage{db: db}
    if err := storage.initSchema(); err != nil {
        db.Close()
        return nil, err
    }

    return storage, nil
}
```

#### Set Operation (Creates New Version in Transaction)

```go
func (s *SQLiteStorage) Set(ctx context.Context, name, content string) error {
    name = sanitizeName(name)
    now := time.Now()

    // Transaction for atomic version generation
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()

    // Get next version number
    var nextVersion int
    err = tx.QueryRowContext(ctx, `
        SELECT COALESCE(MAX(version), 0) + 1
        FROM memories
        WHERE name = ?
    `, name).Scan(&nextVersion)
    if err != nil {
        return err
    }

    // Insert new version
    _, err = tx.ExecContext(ctx, `
        INSERT INTO memories (name, version, content, deleted, created_at, updated_at)
        VALUES (?, ?, ?, FALSE, ?, ?)
    `, name, nextVersion, content, now, now)
    if err != nil {
        return err
    }

    return tx.Commit()
}
```

#### Get Operation (Latest Non-Deleted Version)

```go
func (s *SQLiteStorage) Get(ctx context.Context, name string) (mo.Maybe[MemoryItem], error) {
    name = sanitizeName(name)

    query := `
    SELECT name, version, content, deleted, created_at, updated_at
    FROM memories
    WHERE name = ? AND deleted = FALSE
    ORDER BY version DESC
    LIMIT 1
    `

    var item MemoryItem
    err := s.db.QueryRowContext(ctx, query, name).Scan(
        &item.Name, &item.Version, &item.Content,
        &item.Deleted, &item.CreatedAt, &item.UpdatedAt,
    )

    if err == sql.ErrNoRows {
        return mo.Nothing[MemoryItem](), nil
    }
    if err != nil {
        return mo.Nothing[MemoryItem](), err
    }

    return mo.Just(item), nil
}
```

#### Delete Operation (Soft Delete ALL Versions)

```go
func (s *SQLiteStorage) Delete(ctx context.Context, name string) error {
    name = sanitizeName(name)

    _, err := s.db.ExecContext(ctx, `
        UPDATE memories
        SET deleted = TRUE, updated_at = ?
        WHERE name = ? AND deleted = FALSE
    `, time.Now(), name)

    return err
}
```

#### List Operation (Latest Version per Name with Search)

```go
func (s *SQLiteStorage) List(ctx context.Context, search string) ([]MemoryMetadata, error) {
    var query string
    var args []interface{}

    if search == "" {
        query = `
        SELECT name, version, length(content) as size, created_at, updated_at
        FROM memories m1
        WHERE deleted = FALSE
        AND version = (
            SELECT MAX(version)
            FROM memories m2
            WHERE m2.name = m1.name AND m2.deleted = FALSE
        )
        ORDER BY updated_at DESC
        `
    } else {
        searchParam := "%" + search + "%"
        query = `
        SELECT name, version, length(content) as size, created_at, updated_at
        FROM memories m1
        WHERE deleted = FALSE
        AND version = (
            SELECT MAX(version)
            FROM memories m2
            WHERE m2.name = m1.name AND m2.deleted = FALSE
        )
        AND (LOWER(name) LIKE LOWER(?) OR LOWER(content) LIKE LOWER(?))
        ORDER BY updated_at DESC
        `
        args = []interface{}{searchParam, searchParam}
    }

    rows, err := s.db.QueryContext(ctx, query, args...)
    // ... scan rows into []MemoryMetadata
}
```

#### Maybe Monad (from mcp-genie/pkg/mo)

```go
// From mcp-genie/pkg/mo/maybe.go
type Maybe[T any] struct {
    value T
    valid bool
}

func Just[T any](v T) Maybe[T] {
    return Maybe[T]{value: v, valid: true}
}

func Nothing[T any]() Maybe[T] {
    return Maybe[T]{valid: false}
}

func (m Maybe[T]) HasValue() bool { return m.valid }
func (m Maybe[T]) Value() T       { return m.value }
```

#### Name Sanitization

```go
func sanitizeName(name string) string {
    if name == "" {
        return "unnamed"
    }

    runes := []rune(name)
    result := make([]rune, len(runes))

    for i, r := range runes {
        // Keep valid characters: alphanumeric, underscore, hyphen, dot
        if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
            (r >= '0' && r <= '9') || r == '_' || r == '-' || r == '.' {
            result[i] = r
        } else {
            result[i] = '_'
        }
    }

    return string(result)
}
```

#### Backend Selection (Factory Pattern)

```go
type BackendType string

const (
    BackendSQLite   BackendType = "sqlite"
    BackendPostgres BackendType = "postgres"
)

type StorageConfig struct {
    Backend     BackendType
    SQLitePath  string
    PostgresURL string
    MaxConns    int32
    MinConns    int32
}

func NewStorage(ctx context.Context, cfg StorageConfig) (Storage, error) {
    switch cfg.Backend {
    case BackendSQLite:
        return NewSQLiteStorage(cfg.SQLitePath)
    case BackendPostgres:
        return NewPostgresStorage(ctx, cfg)
    default:
        return nil, fmt.Errorf("unknown backend: %s", cfg.Backend)
    }
}

// Default: SQLite for backward compatibility
func LoadConfigFromEnv() StorageConfig {
    backend := os.Getenv("STICKYMEMORY_BACKEND")
    if backend == "" {
        backend = string(BackendSQLite)
    }

    return StorageConfig{
        Backend:    BackendType(backend),
        SQLitePath: getEnvOrDefault("STICKYMEMORY_SQLITE_PATH",
                                    "~/.mcp-genie/stickymemory/memories.db"),
        PostgresURL: os.Getenv("STICKYMEMORY_POSTGRES_URL"),
        MaxConns:   10,
        MinConns:   2,
    }
}
```

### Compatibility Requirements for zombiekit

| Requirement | mcp-genie Pattern | zombiekit Mapping |
|-------------|-------------------|-------------------|
| Schema | `(name, version)` composite PK | Identical |
| Storage interface | `Set`, `Get`, `Delete`, `List`, `Clear`, `Close` | Identical signature |
| Maybe monad | `mo.Maybe[T]`, `Just`, `Nothing` | Copy or implement identically |
| Soft delete | `deleted=TRUE` on ALL versions | Identical |
| Name sanitization | `sanitizeName()` with allowed chars | Identical logic |
| Version management | `MAX(version) + 1` in transaction | Identical |
| Environment vars | `STICKYMEMORY_*` | `BRAINS_*` (renamed) |
| Default SQLite path | `~/.mcp-genie/stickymemory/memories.db` | `~/.brains/memories.db` |

### Implementation Checklist

- [ ] Create `internal/mo/maybe.go` with `Maybe[T]`, `Just`, `Nothing`
- [ ] Create `internal/memory/types.go` with `MemoryItem`, `MemoryMetadata`
- [ ] Create `internal/memory/storage.go` with `Storage` interface
- [ ] Create `internal/memory/sqlite/storage.go` matching mcp-genie patterns
- [ ] Create `internal/memory/postgres/storage.go` with identical interface
- [ ] Create `internal/memory/factory.go` for backend selection
- [ ] Use `modernc.org/sqlite` driver (pure Go, no CGO)
- [ ] Implement `sanitizeName()` with identical logic
- [ ] Default to SQLite backend with `~/.brains/memories.db`
