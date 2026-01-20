---
status: draft
created: 2026-01-19
spec: spec.md
---

# Technical Specification: Unified Startup Command

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     brains start                             │
├─────────────────────────────────────────────────────────────┤
│  StartupConfig (YAML)        │  shutdown.Manager            │
│  - Load from file/env        │  - Signal capture            │
│  - Validate                  │  - Context propagation       │
│  - Default values            │  - Timeout enforcement       │
├─────────────────────────────────────────────────────────────┤
│                        errgroup                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │  GUIService │  │RecallService│  │SignalHandler│         │
│  │  (port 9981)│  │(watch mode) │  │(SIGINT/TERM)│         │
│  └─────────────┘  └─────────────┘  └─────────────┘         │
│         │                │                │                 │
│         └────────────────┼────────────────┘                 │
│                          │                                  │
│              ctx.Done() propagation                         │
└─────────────────────────────────────────────────────────────┘
```

## Components

### 1. shutdown.Manager

```go
package shutdown

type Manager struct {
    timeout time.Duration
}

func New(timeout time.Duration) *Manager
func (m *Manager) Run(services ...func(context.Context) error) error
```

**Behavior:**
1. Creates cancellable context
2. Registers SIGINT/SIGTERM handlers
3. Launches services in errgroup
4. First signal → cancel context
5. Second signal → os.Exit(1)
6. Timeout exceeded → os.Exit(1)

### 2. StartupConfig

```go
package config

type StartupConfig struct {
    Services ServiceConfigs `yaml:"services"`
}

type ServiceConfigs struct {
    GUI    GUIConfig    `yaml:"gui"`
    Recall RecallConfig `yaml:"recall"`
}

type GUIConfig struct {
    Enabled bool   `yaml:"enabled"`
    Port    int    `yaml:"port"`
}

type RecallConfig struct {
    Enabled  bool          `yaml:"enabled"`
    Source   string        `yaml:"source"`
    Interval time.Duration `yaml:"interval"`
    Verbose  bool          `yaml:"verbose"`
}

func LoadStartupConfig() (*StartupConfig, error)
func (c *StartupConfig) Validate() error
```

**File discovery:**
1. `.brains/config.yml` (local)
2. `~/.brains/config.yml` (global)
3. Environment variable overrides

**Defaults:**
```yaml
services:
  gui:
    enabled: true
    port: 9981            # BRAINS_GUI_PORT
  recall:
    enabled: true
    source: claude
    interval: 30s
    verbose: false
```

### 3. Service Interface

```go
package startup

type Service interface {
    Name() string
    Run(ctx context.Context) error
}
```

### 4. GUIService

```go
type GUIService struct {
    config config.GUIConfig
}

func (s *GUIService) Name() string { return "gui" }

func (s *GUIService) Run(ctx context.Context) error {
    // Initialize registry, server config
    // Create web.Server
    // Call server.Start(ctx)
}
```

**Shutdown behavior:** The `web.Server.Start()` already handles ctx cancellation via internal shutdown goroutine. No additional work needed.

### 5. RecallService

```go
type RecallService struct {
    config config.RecallConfig
}

func (s *RecallService) Name() string { return "recall" }

func (s *RecallService) Run(ctx context.Context) error {
    // Initialize storage, embedder
    // Run import loop with ticker
    // Exit on ctx.Done()
}
```

**Shutdown behavior:** Adapted from `recallWatchClaudeAction`, already has ctx.Done() handling in ticker loop.

## Signal Handling Flow

```
User presses Ctrl+C
        │
        v
┌───────────────────┐
│ sigCh <- SIGINT   │  (buffered, capacity 2)
└───────┬───────────┘
        │
        v
┌───────────────────┐
│ cancel() called   │  context canceled
└───────┬───────────┘
        │
        ├──────────────────────────────────────┐
        v                                      v
┌───────────────────┐              ┌───────────────────┐
│ GUI: ctx.Done()   │              │Recall: ctx.Done() │
│ → Server.Shutdown │              │ → exit loop       │
└───────┬───────────┘              └───────┬───────────┘
        │                                  │
        v                                  v
┌───────────────────┐              ┌───────────────────┐
│ GUI returns nil   │              │ Recall returns nil│
└───────┬───────────┘              └───────┬───────────┘
        │                                  │
        └──────────────┬───────────────────┘
                       v
              ┌───────────────────┐
              │ g.Wait() returns  │
              └───────┬───────────┘
                      │
                      v
              ┌───────────────────┐
              │ Shutdown complete │
              └───────────────────┘
```

**Alternative path (timeout):**

```
User presses Ctrl+C → services don't exit within 10s
        │
        v
┌───────────────────────┐
│ time.After fires      │
│ log.Error("timeout")  │
│ os.Exit(1)            │
└───────────────────────┘
```

**Alternative path (force exit):**

```
User presses Ctrl+C twice rapidly
        │
        v
┌───────────────────────┐
│ Second sigCh read     │
│ log("forcing exit")   │
│ os.Exit(1)            │
└───────────────────────┘
```

## Logging

Each service logs with a prefix:

```
[gui] starting web server on port 9981
[gui] serving request GET /api/memories
[recall] watching for new conversations (interval: 30s)
[recall] imported 5 new messages from 2 files
[gui] received shutdown signal, stopping server
[recall] shutting down...
```

Implementation via slog's `WithGroup` or custom handler.

## Error Handling

| Scenario | Behavior |
|----------|----------|
| GUI port in use | GUI service returns error, errgroup cancels all services, command exits |
| Ollama not running | Recall service returns error, errgroup cancels all services, command exits |
| Invalid config YAML | Exit before starting any services, clear parse error |
| Service panics | Recovered by errgroup, treated as error, all services canceled |

**Fail-fast rationale**: Simpler mental model for users. If something's wrong, fix it and restart. No partial states to debug.

## Configuration Validation

Checked before any service starts:

1. `gui.port` must be 1-65535
2. `recall.source` must be "claude" (only supported source)
3. `recall.interval` must be >= 1s
4. File paths (if specified) must exist

## Testing Strategy

| Test Type | Coverage |
|-----------|----------|
| Unit | shutdown.Manager, config.LoadStartupConfig, config.Validate |
| Integration | Start command with mock services |
| Manual | Full stack with real GUI/recall, signal handling |

## File Layout

```
internal/
├── shutdown/
│   ├── manager.go          # Signal handling + timeout
│   └── manager_test.go
├── config/
│   ├── startup.go          # StartupConfig types + loader
│   └── startup_test.go
├── startup/
│   ├── service.go          # Service interface
│   ├── gui_service.go      # GUI wrapper
│   ├── recall_service.go   # Recall wrapper
│   └── service_test.go
└── cli/
    ├── start.go            # brains start command
    └── start_test.go
```
