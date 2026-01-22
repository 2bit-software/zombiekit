# Technical Specification: --env-file Flag

## Overview

Add `--env-file` flag to `brains serve` command to load environment variables from a file before processing other configuration.

## Dependencies

**New dependency:**
```
github.com/joho/godotenv
```

MIT licensed, 7k+ stars, widely used in Go ecosystem. Handles dotenv parsing edge cases (quotes, comments, multiline, etc.).

## Implementation Design

### Flag Definition

Add to `newServeCommand()` in `internal/cli/serve.go`:

```go
&cli.StringFlag{
    Name:    "env-file",
    Usage:   "Path to environment file to load",
    EnvVars: []string{"BRAINS_ENV_FILE"},
},
```

### Loading Logic

Add at the **very beginning** of `runServe()`, before logging initialization:

```go
func runServe(c *cli.Context) error {
    // Load environment file first (before any other config)
    if envFile := c.String("env-file"); envFile != "" {
        if err := loadEnvFile(envFile); err != nil {
            return err
        }
    }

    ctx := context.Background()
    // ... rest of function
}
```

### loadEnvFile Function

```go
// loadEnvFile loads environment variables from a file.
// Uses godotenv.Load which does NOT override existing environment variables.
func loadEnvFile(path string) error {
    // Check file exists and is readable
    info, err := os.Stat(path)
    if os.IsNotExist(err) {
        return fmt.Errorf("env file not found: %s", path)
    }
    if err != nil {
        return fmt.Errorf("cannot access env file: %w", err)
    }
    if info.IsDir() {
        return fmt.Errorf("env file path is a directory: %s", path)
    }

    // Load environment variables (does not override existing)
    if err := godotenv.Load(path); err != nil {
        return fmt.Errorf("failed to load env file: %w", err)
    }

    return nil
}
```

## Precedence Model

The precedence (lowest to highest) is:

1. **Env file values** - Set by `godotenv.Load()` (does not override existing)
2. **Existing environment** - Already set in shell/process environment
3. **CLI flags** - Applied later in `runServe()` via `c.IsSet()` checks

This is achieved naturally:
- `godotenv.Load()` only sets env vars if they don't already exist
- CLI flag checks like `if c.IsSet("db-type")` override config at runtime

## File Changes

| File | Change |
|------|--------|
| `go.mod` | Add `github.com/joho/godotenv` dependency |
| `internal/cli/serve.go` | Add `--env-file` flag and `loadEnvFile()` function |

## Testing Strategy

### Integration Test: `serve_test.go`

```go
func TestServeCommand_EnvFile(t *testing.T) {
    // Create temp env file
    tmpFile, _ := os.CreateTemp("", "test-*.env")
    defer os.Remove(tmpFile.Name())
    tmpFile.WriteString("BRAINS_BACKEND=postgres\nBRAINS_LOG_LEVEL=debug\n")
    tmpFile.Close()

    // Test that loadEnvFile works
    os.Unsetenv("BRAINS_BACKEND")
    err := loadEnvFile(tmpFile.Name())
    require.NoError(t, err)
    assert.Equal(t, "postgres", os.Getenv("BRAINS_BACKEND"))
}

func TestServeCommand_EnvFile_NotOverride(t *testing.T) {
    // Set existing env var
    os.Setenv("BRAINS_LOG_LEVEL", "error")
    defer os.Unsetenv("BRAINS_LOG_LEVEL")

    // Create env file with different value
    tmpFile, _ := os.CreateTemp("", "test-*.env")
    defer os.Remove(tmpFile.Name())
    tmpFile.WriteString("BRAINS_LOG_LEVEL=debug\n")
    tmpFile.Close()

    // Load should not override
    err := loadEnvFile(tmpFile.Name())
    require.NoError(t, err)
    assert.Equal(t, "error", os.Getenv("BRAINS_LOG_LEVEL"))
}

func TestServeCommand_EnvFile_MissingFile(t *testing.T) {
    err := loadEnvFile("/nonexistent/file.env")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "env file not found")
}

func TestServeCommand_EnvFile_IsDirectory(t *testing.T) {
    err := loadEnvFile("/tmp")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "is a directory")
}
```

## Error Messages

| Scenario | Error Message |
|----------|---------------|
| File not found | `env file not found: /path/to/file.env` |
| Path is directory | `env file path is a directory: /path` |
| Permission denied | `cannot access env file: permission denied` |
| Parse error | `failed to load env file: ...` (godotenv error) |

## Usage Examples

### Claude Code MCP Configuration

```json
{
  "mcpServers": {
    "brains": {
      "command": "brains",
      "args": ["serve", "--env-file", "/path/to/project/.env", "--mode", "stdio"]
    }
  }
}
```

### CLI Usage

```bash
# Load from specific file
brains serve --env-file .env --mode stdio

# With absolute path
brains serve --env-file /home/user/project/.env --mode http

# Via environment variable
export BRAINS_ENV_FILE=/path/to/.env
brains serve --mode stdio
```
