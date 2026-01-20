# Go Project Standards

A comprehensive guide to writing idiomatic, maintainable Go code.

---

## General Principles

- Use meaningful variable names that clearly indicate their purpose
- Write comments explaining WHY code exists, not HOW it works
- Keep functions small and focused on a single responsibility
- Prioritize readability over cleverness
- Optimize for maintainability first, performance second
- Handle errors with proper context and propagation
- Document public interfaces with usage examples
- Prefer composition over inheritance
- Do not add features or enhancements without explicit instruction
- Avoid nesting beyond 3 levels; break deep logic into separate functions
- Avoid if/else blocks exceeding 4 lines; extract branches into smaller functions

---

## Language & Syntax

### Modern Go Idioms

- Use `any` instead of `interface{}` (Go 1.18+)
- Use Go modules for dependency management
- Prefer slices over arrays when length might change
- Implement interfaces implicitly rather than explicitly declaring them
- Leverage Go's concurrency primitives (goroutines, channels) appropriately
- Use `context.Context` for cancellation and timeouts

### Naming Conventions

- Use `MixedCaps` or `mixedCaps` for multiword names—never underscores
- Package names: lowercase, single-word, short and evocative
- Exported names: `UpperCamelCase`
- Unexported names: `lowerCamelCase`
- Receiver names: one or two letters, prefixed with underscore (e.g., `func (_s *Server) Start()`)
- Loop indices: single letters (`i`, `j`, `k`)
- Common variables: short names (`r` for reader, `w` for writer, `ctx` for context, `err` for error)
- Boolean variables: prefix with `Has`, `Is`, `Can`, or `Allow`
- Avoid `Get` prefix for getters—use the field name directly
- Interface naming: single-method interfaces use the method name + `-er` suffix (e.g., `Reader`, `Writer`, `Stringer`)

### Code Organization

- Follow standard Go project layout (`/cmd`, `/internal`, `/pkg`)
- `main.go` should be minimal: gather config/environment variables and pass to an app "run" function
- Keep packages focused; avoid generic names like `util` or `common`
- Use meaningful struct tags for serialization (always specify field names in JSON/YAML tags)
- Group related declarations together; separate unrelated ones

---

## Error Handling

### Core Rules

- Always check errors and return them with additional context
- When comparing errors, ALWAYS use `errors.Is()` or `errors.As()`—never `==`
- Wrap errors using `fmt.Errorf` with `%w` verb to preserve the error chain

```go
if err != nil {
    return fmt.Errorf("failed to process file %s: %w", filename, err)
}
```

### Best Practices

- Add context at each layer of the call stack
- Use sentinel errors for well-defined conditions (e.g., `ErrNotFound`)
- Use custom error types for complex scenarios requiring additional metadata
- Don't over-wrap—add context only when it provides value
- Log errors at the appropriate level with structured logging
- Fail fast on invalid input rather than propagating bad state

### Avoid

- Using panics in non-test code; prefer returning errors
- Never use "must" functions that panic in production code—use safe versions and check for success
- Suppressing or ignoring errors without explicit justification

---

## Testing

### Table-Driven Tests

Table-driven tests are the idiomatic Go approach for testing functions with multiple inputs:

```go
func TestAdd(t *testing.T) {
    tests := map[string]struct {
        a, b     int
        expected int
    }{
        "positive numbers":  {a: 2, b: 3, expected: 5},
        "negative numbers":  {a: -1, b: -2, expected: -3},
        "zero":              {a: 0, b: 0, expected: 0},
    }

    for name, tc := range tests {
        t.Run(name, func(t *testing.T) {
            result := Add(tc.a, tc.b)
            if result != tc.expected {
                t.Errorf("Add(%d, %d) = %d; want %d", tc.a, tc.b, result, tc.expected)
            }
        })
    }
}
```

### Testing Rules

- Use `testify/assert` and `testify/suite` libraries for assertions
- When running tests, pay attention to build tags to determine which tags to include
- Run tests against a folder with `-run` to specify which test: `go test -run "TestFoo|TestBar" ./pkg/...`
- Disable test cache with `-count=1` for fresh runs
- Name every test case descriptively
- Provide clear, detailed error messages showing input, expected, and actual values
- Use `t.Run()` for subtests to enable parallel execution and better isolation
- Use maps for test cases to get undefined iteration order (exposes order-dependent bugs)

### Test Organization

- Test files end with `_test.go`
- Place tests in the same package as the code being tested
- Extract complex setup/teardown into helper functions
- Keep test logic simple—if tests are convoluted, the function may need refactoring

### Database Test Isolation

Tests that interact with databases must be fully isolated from the environment. This prevents tests from accidentally connecting to production or development databases when environment variables are set (e.g., via `.env` files loaded by task runners).

**Environment Variable Override Pattern**

Always set ALL relevant backend configuration, not just the path:

```go
func runDBCmd(t *testing.T, dbPath string, args ...string) (string, error) {
    t.Helper()

    // Set env vars for test isolation - must override any .env settings
    os.Setenv("BRAINS_BACKEND", "sqlite")
    os.Setenv("BRAINS_SQLITE_PATH", dbPath)
    defer func() {
        os.Unsetenv("BRAINS_BACKEND")
        os.Unsetenv("BRAINS_SQLITE_PATH")
    }()

    // ... run command
}
```

The bug: Setting only `BRAINS_SQLITE_PATH` without `BRAINS_BACKEND=sqlite` causes tests to connect to whatever backend is configured in the environment (often postgres from `.env`), ignoring the temp SQLite path entirely.

**SQLite for CLI/Unit Tests**

Use `t.TempDir()` for temporary SQLite databases in CLI and unit tests:

```go
func setupMemoryTest(t *testing.T) (*sqlite.SQLiteStorage, string) {
    t.Helper()

    tmpDir := t.TempDir()
    dbPath := filepath.Join(tmpDir, "test.db")

    storage, err := sqlite.NewSQLiteStorage(context.Background(), dbPath)
    require.NoError(t, err)

    t.Cleanup(func() {
        storage.Close()
    })

    return storage, dbPath
}
```

**Testcontainers for Integration Tests**

Use testcontainers-go for integration tests that need real PostgreSQL (e.g., pgvector operations):

```go
func setupTestHarness(t *testing.T) *testHarness {
    t.Helper()

    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    ctx := context.Background()

    container, err := tcpostgres.Run(ctx,
        "pgvector/pgvector:pg16",
        tcpostgres.WithDatabase("test"),
        tcpostgres.WithUsername("test"),
        tcpostgres.WithPassword("test"),
        testcontainers.WithWaitStrategy(
            wait.ForLog("database system is ready to accept connections").
                WithOccurrence(2).
                WithStartupTimeout(60*time.Second),
        ),
    )
    require.NoError(t, err)

    t.Cleanup(func() {
        if err := container.Terminate(ctx); err != nil {
            t.Logf("failed to terminate container: %v", err)
        }
    })

    // ... setup pool and schema
}
```

**Schema Synchronization**

Test schemas must exactly match production schemas. When adding columns to production, update test schemas too:

```go
// Test schema must include ALL columns from production
_, err = pool.Exec(ctx, `
    CREATE TABLE IF NOT EXISTS recall_chunks (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        content TEXT NOT NULL,
        content_hash TEXT NOT NULL,
        embedding vector(768),
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        source TEXT,
        source_id TEXT,
        conversation_id TEXT,
        metadata JSONB,
        history_gap BOOLEAN NOT NULL DEFAULT FALSE  -- Don't forget new columns!
    )
`)
```

**Test Harness Pattern**

For complex integration tests, create a harness struct that encapsulates dependencies:

```go
type testHarness struct {
    storage  *postgres.Storage
    embedder *mockEmbedder
    tmpDir   string
}

func (h *testHarness) importFile(ctx context.Context, filePath string) (newCount, skipCount int, err error) {
    // ... test helper logic
}
```

---

## Project Structure

### Standard Layout

```
project/
├── cmd/
│   └── myapp/
│       └── main.go           # Minimal entry point
├── internal/                  # Private application code
│   ├── config/
│   ├── domain/
│   └── service/
├── pkg/                       # Public, reusable packages
├── api/                       # API specs (OpenAPI, protobuf)
├── scripts/                   # Build/install scripts
├── configs/                   # Configuration templates
├── docs/                      # Documentation
├── go.mod
└── go.sum
```

### Guidelines

- Put code in `/internal` if you don't want others to import it
- Put code in `/pkg` if it can be reused by other projects
- Match `/cmd` directory names to executable names
- Don't over-structure small projects—flat is fine until complexity demands otherwise
- No "one type, one file" rule; organize by logical grouping

---

## Concurrency

- Use `context.Context` as the first parameter for functions that may block
- Always pass context through the call chain
- Handle cancellation and timeouts explicitly
- Don't leak goroutines—ensure all goroutines have controlled lifetimes
- Use channels for communication; use mutexes for protecting shared state
- Prefer `sync.WaitGroup` for coordinating multiple goroutines
- Be cautious with goroutine spawning in loops—capture loop variables properly

---

## Documentation

- All exported names must have doc comments
- Comments should be full sentences beginning with the name being documented
- End comments with a period

```go
// Request represents a request to run a command.
type Request struct { ... }

// Encode writes the JSON encoding of req to w.
func Encode(w io.Writer, req *Request) error { ... }
```

- Document function purpose and usage, not implementation details
- Use examples in `_test.go` files for runnable documentation

---

## Tooling

### Required

- `gofmt` or `goimports` for formatting (non-negotiable)
- `go vet` for static analysis
- `golangci-lint` for comprehensive linting

### Recommended Linters

- `errcheck` - unchecked errors
- `staticcheck` - advanced static analysis
- `gosimple` - code simplification suggestions
- `ineffassign` - ineffective assignments
- `revive` - modern successor to `golint`

---

## Code Generation

- Never edit files ending in `_gen.go` or with generation headers
- Use the required generation command to regenerate when interfaces change
- Document the generation command in the file header or README

---

## CLI Applications

- Use `urfave/cli` for command-line applications
- Keep `main.go` minimal—just wire up config and call the app runner
- Avoid exporting new flags as side effects of importing packages

---

## Dependencies

- Vendor dependencies when reproducibility is critical
- Use `go mod tidy` regularly to clean up unused dependencies
- Pin major versions for stability
- Evaluate third-party packages for proper error handling before adoption

---

## Performance Considerations

- Profile before optimizing
- Use named return values for clarity when appropriate
- Avoid unnecessary allocations in hot paths
- Prefer value receivers unless mutation is required
- Be mindful of goroutine costs—they're lightweight but not free

---

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Google Go Style Guide](https://google.github.io/styleguide/go/)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
