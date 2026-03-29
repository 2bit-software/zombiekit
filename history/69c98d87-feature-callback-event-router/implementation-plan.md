# Implementation Plan: Callback Event Router

**Status**: Draft
**Spec**: [spec.md](./spec.md)
**Research**: [research.md](./research.md)

## Implementation Order

Steps are ordered by dependency. Each step produces testable, compilable code.

---

### Step 1: Config Extensions

**Files**: `internal/orchestrator/config.go`, `cmd/orchestrator/main.go`, `internal/orchestrator/config_test.go`

Add new fields to `Config`:
- `GitHubOwner string` (required, env `ORCH_GITHUB_OWNER`)
- `GitHubRepo string` (required, env `ORCH_GITHUB_REPO`)
- `BaseBranch string` (optional, default `"main"`, env `ORCH_BASE_BRANCH`)
- `TrackingLabel string` (optional, default `"ai-managed"`, env `ORCH_TRACKING_LABEL`)

Add corresponding CLI flags in `main.go`. Add validation in `Validate()` for required fields. Defaults are applied via CLI `Value:` field.

**Tests**: Validation tests for missing `GitHubOwner`/`GitHubRepo`, default values for `BaseBranch` and `TrackingLabel`.

**Traces to**: IR-003

---

### Step 2: LinearClient.PostComment

**Files**: `internal/linear/client.go`, `internal/linear/http_client.go`, `internal/linear/mock.go`, `internal/linear/http_client_test.go`

Add `PostComment(ctx context.Context, issueID string, body string) error` to the `Client` interface.

Implement in `HTTPClient`:
1. Define `commentCreateMutation` GraphQL const (Linear's `commentCreate` mutation takes `issueId` and `body`).
2. Define `commentCreateResponse` struct with `CommentCreate.Success` field.
3. Call `c.doWithRetry(ctx, mutation, vars, &resp)` following existing mutation pattern.
4. Check `resp.CommentCreate.Success`.

Add `PostCommentFn` field to `MockClient` and implement the interface method with call recording.

**Tests**: Integration test against the mutation response parsing. Follows existing `TestSetTicketStatus` pattern.

**Traces to**: IR-001

---

### Step 3: Archival and Friction Stub Packages

**Files**:
- `internal/archival/archiver.go` (interface + noop)
- `internal/friction/auditor.go` (interface + noop)

Create two new packages with minimal interfaces:

```go
// internal/archival/archiver.go
type Archiver interface {
    Archive(ctx context.Context, ticketID string, eventKind callback.EventKind) error
}
type NoopArchiver struct{}
func (NoopArchiver) Archive(context.Context, string, callback.EventKind) error { return nil }

// internal/friction/auditor.go
type Auditor interface {
    Audit(ctx context.Context, ticketID string, eventKind callback.EventKind) error
}
type NoopAuditor struct{}
func (NoopAuditor) Audit(context.Context, string, callback.EventKind) error { return nil }
```

No tests needed for no-op implementations.

**Traces to**: IR-004, IR-005

---

### Step 4: Orchestrator Dependency Wiring

**Files**: `internal/orchestrator/orchestrator.go`, `cmd/orchestrator/main.go`, `internal/orchestrator/orchestrator_test.go`

1. Add `github github.Client` field to `Orchestrator` struct.
2. Update `New()` to accept `github.Client` parameter.
3. In `main.go:run()`, create `github.NewClient(cfg.GitHubToken, cfg.GitHubOwner, cfg.GitHubRepo)` and pass to `New()`.
4. Update existing tests that call `New()` to pass a nil or mock GitHub client.

**Tests**: Existing orchestrator tests must still compile and pass.

**Traces to**: IR-002

---

### Step 5: Event Router (Core)

**Files**: `internal/orchestrator/router.go`, `internal/orchestrator/router_test.go`

This is the main deliverable. Implement the `Router` struct and its `Run` method.

**Router struct**:
```go
type Router struct {
    events   <-chan callback.Event
    store    state.StateStore
    github   github.Client
    linear   linear.Client
    archiver archival.Archiver
    auditor  friction.Auditor
    cfg      *Config
    logger   *slog.Logger
}
```

**Run method** (implements `shutdown.ServiceFunc`):
```go
func (r *Router) Run(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return nil
        case evt, ok := <-r.events:
            if !ok {
                return nil  // channel closed
            }
            r.handleEvent(ctx, evt)
        }
    }
}
```

**handleEvent**: Switch on `evt.Kind`, dispatch to typed handler methods. Each handler logs errors and calls `r.markNeedsAttention()` on failure.

**Handler methods** (private, one per event kind):

- `handleComplete(ctx, event)`:
  1. GetJob (nil check)
  2. Read `.ai/pr-description.md` (os.ReadFile)
  3. GetTicket (for title)
  4. CreatePR
  5. SetPR
  6. ApplyLabel
  7. Archive
  8. Audit

- `handleFailed(ctx, event)`:
  1. GetJob (nil is allowed)
  2. LinearClient.SetTicketStatus → "needs-attention"
  3. If job exists: StateStore.SetJobStatus → StatusNeedsAttention
  4. LinearClient.PostComment
  5. defer: StateStore.ReleaseSlot (always runs)
  6. Archive

- `handleCommentResolved(ctx, event)`:
  1. GetJob (nil check)
  2. Verify PRNumber non-nil
  3. Parse CommentID as int64
  4. Read `.ai/pr-description.md`
  5. UpdatePRBody
  6. PostCommentReply
  7. SetCommentWatermark
  8. Archive
  9. Audit

**Helper: markNeedsAttention(ctx, ticketID, job)**: Sets status in both Linear and state (skipping state if job is nil). Logs but doesn't return errors (best-effort during error handling).

**Tests** (integration, mock-based):
- Happy path for each event kind (verify mock call sequences)
- Missing `.ai/pr-description.md` → needs-attention
- Unknown ticket (GetJob nil) → logged and discarded
- CreatePR failure → needs-attention
- Nil PRNumber on CommentResolved → needs-attention
- Invalid CommentID parse → needs-attention
- Linear API failure during FailureEvent → slot still released
- Channel closed → Run returns nil
- Context cancelled → Run returns nil

**Traces to**: FR-001 through FR-009

---

### Step 6: Wire Router into Orchestrator.Run()

**Files**: `internal/orchestrator/orchestrator.go`

In `Orchestrator.Run()`:
1. After creating `callbackSrv`, create the router:
   ```go
   router := NewRouter(
       callbackSrv.Events(), o.store, o.github, o.linear,
       archival.NoopArchiver{}, friction.NoopAuditor{},
       o.cfg, logging.Logger(),
   )
   ```
2. Pass `router.Run` to the shutdown manager alongside the other services.
3. Remove the PR watcher and comment watcher stubs (they're superseded by the router for event handling).

**Tests**: Existing orchestrator lifecycle tests should still pass.

**Traces to**: FR-009 (ServiceFunc integration)

---

## Dependency Graph

```
Step 1 (Config) ─────────────────────┐
Step 2 (PostComment) ────────────────┤
Step 3 (Archival/Friction stubs) ────┤
                                     ▼
                     Step 4 (Orchestrator wiring)
                                     │
                                     ▼
                     Step 5 (Router core + tests)
                                     │
                                     ▼
                     Step 6 (Wire into Orchestrator.Run)
```

Steps 1, 2, 3 are independent of each other and could be implemented in any order.
Step 4 depends on Step 1 (config has new fields).
Step 5 depends on Steps 1-4 (all dependencies available).
Step 6 depends on Step 5.

## Remaining Uncertainties

None. All interfaces, patterns, and signatures are confirmed from the codebase.

## Out of Scope

- Real archival implementation (Epic 5)
- Real friction auditor implementation (Epic 5)
- PR watcher / comment watcher (separate tickets)
- Worktree cleanup on merge (separate ticket)
