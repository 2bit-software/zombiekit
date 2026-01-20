---
status: draft
created: 2026-01-19
spec: spec.md
spike: spike-results.md
---

# Implementation Plan: Unified Startup Command

## Overview

Implement `brains start` command that orchestrates GUI and recall services with unified lifecycle management.

## Implementation Phases

### Phase 1: Shutdown Manager Package

Create `internal/shutdown/manager.go` - encapsulates the validated signal handling pattern.

**Deliverables:**
- `Manager` struct with `Run()` and `Wait()` methods
- Context propagation to services
- 10-second graceful shutdown timeout
- Double Ctrl+C force exit
- Unit tests for the manager

**Rationale:** Isolating shutdown logic makes it testable and reusable. The GUI and recall commands could also adopt this pattern later.

**Files:**
- `internal/shutdown/manager.go` (new)
- `internal/shutdown/manager_test.go` (new)

### Phase 2: Configuration System

Create YAML-based configuration for services.

**Deliverables:**
- `internal/config/startup.go` - config types and loader
- Default values matching current `task up` behavior
- Validation with clear error messages
- Config file discovery: `.brains/config.yml` → `~/.brains/config.yml` → env vars

**Config structure:**
```yaml
services:
  gui:
    enabled: true
    port: 9981
  recall:
    enabled: true
    source: claude
    interval: 30s
    verbose: false
```

**Files:**
- `internal/config/startup.go` (new)
- `internal/config/startup_test.go` (new)

### Phase 3: Service Runner Abstraction

Create interface for running services in the unified lifecycle.

**Deliverables:**
- `Service` interface with `Run(ctx) error` method
- Wrapper for GUI server
- Wrapper for recall watch
- Service-prefixed logging

**Interface:**
```go
type Service interface {
    Name() string
    Run(ctx context.Context) error
}
```

**Files:**
- `internal/startup/service.go` (new)
- `internal/startup/gui_service.go` (new)
- `internal/startup/recall_service.go` (new)

### Phase 4: Start Command CLI

Implement the `brains start` CLI command.

**Deliverables:**
- `internal/cli/start.go` - command implementation
- Integration with shutdown manager
- Log prefixing (`[gui]`, `[recall]`)
- Error handling and reporting

**Files:**
- `internal/cli/start.go` (new)
- `internal/cli/root.go` (modify - register command)

### Phase 5: Integration Testing

End-to-end tests for the start command.

**Deliverables:**
- Test: start command launches services
- Test: Ctrl+C triggers graceful shutdown
- Test: disabled services are skipped
- Test: invalid config produces clear error

**Files:**
- `internal/cli/start_test.go` (new)

## Dependency Order

```
Phase 1 (Shutdown Manager) - no dependencies
        ↓
Phase 2 (Configuration) - no dependencies
        ↓
Phase 3 (Service Runners) - depends on Phase 2
        ↓
Phase 4 (Start Command) - depends on Phase 1, 2, 3
        ↓
Phase 5 (Integration Tests) - depends on Phase 4
```

Phase 1 and Phase 2 can be done in parallel.

## Technical Decisions

### D1: Shutdown Pattern
**Decision:** Adopt errgroup + buffered signal channel pattern from spike
**Rationale:** Validated in spike with tests, matches Kubernetes/Consul patterns

### D2: Service Isolation
**Decision:** Wrap existing GUI/recall into Service interface
**Rationale:** Minimal changes to existing code, services retain their current behavior
**Alternative considered:** Modify existing commands directly - rejected because it would break backward compatibility

### D3: Configuration Format
**Decision:** YAML with local/global hierarchy
**Rationale:** Matches existing `.brains/` directory pattern, human-readable
**Alternative considered:** TOML - rejected for consistency with ZombieKit conventions

### D4: Log Prefixing
**Decision:** Modify logger to accept service name context
**Rationale:** Works with existing slog infrastructure
**Alternative considered:** Separate log streams - rejected due to complexity

## Risk Mitigation

| Risk | Mitigation |
|------|------------|
| GUI server doesn't respect context | Wrap with shutdown goroutine that calls `Server.Shutdown()` |
| Recall watcher doesn't exit on context cancel | Already has ctx.Done() handling - verified in code review |
| Existing tests break | Services are wrapped, not modified |
| Database connections leak | Services use defer storage.Close() pattern already |

## Success Criteria Mapping

| Success Criterion | Implementation Coverage |
|-------------------|------------------------|
| SC-001: Single command startup | Phase 4: `brains start` command |
| SC-002: Ctrl+C stops within 5s | Phase 1: 10s timeout (conservative) |
| SC-003: Logs distinguishable | Phase 3/4: Service-prefixed logging |
| SC-004: Invalid config error | Phase 2: Config validation |

## Out of Scope

- Hot reload of configuration
- Web UI for service management
- Process supervision (restart on crash)
- Custom shutdown order (parallel is fine for gui/recall)
