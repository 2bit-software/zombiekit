# Technical Spec: Callback Event Router

## Package Layout

```
internal/
  archival/
    archiver.go          # Archiver interface + NoopArchiver
  friction/
    auditor.go           # Auditor interface + NoopAuditor
  linear/
    client.go            # Add PostComment to interface
    http_client.go       # Implement PostComment via commentCreate mutation
    mock.go              # Add PostCommentFn
  orchestrator/
    config.go            # Add GitHubOwner, GitHubRepo, BaseBranch, TrackingLabel
    orchestrator.go      # Add github.Client field, update New() and Run()
    router.go            # NEW - Router struct + Run + handlers
    router_test.go       # NEW - Integration tests
cmd/
  orchestrator/
    main.go              # Add CLI flags + github.NewClient wiring
```

## New Types

### Router (internal/orchestrator/router.go)

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

func NewRouter(
    events <-chan callback.Event,
    store state.StateStore,
    gh github.Client,
    lc linear.Client,
    arch archival.Archiver,
    aud friction.Auditor,
    cfg *Config,
    logger *slog.Logger,
) *Router

// Run implements shutdown.ServiceFunc.
func (r *Router) Run(ctx context.Context) error
```

### Archiver (internal/archival/archiver.go)

```go
type Archiver interface {
    Archive(ctx context.Context, ticketID string, eventKind callback.EventKind) error
}

type NoopArchiver struct{}
func (NoopArchiver) Archive(context.Context, string, callback.EventKind) error { return nil }
```

### Auditor (internal/friction/auditor.go)

```go
type Auditor interface {
    Audit(ctx context.Context, ticketID string, eventKind callback.EventKind) error
}

type NoopAuditor struct{}
func (NoopAuditor) Audit(context.Context, string, callback.EventKind) error { return nil }
```

## Modified Types

### Config (internal/orchestrator/config.go)

New fields:
```go
GitHubOwner   string  // Required. CLI: --github-owner, Env: ORCH_GITHUB_OWNER
GitHubRepo    string  // Required. CLI: --github-repo, Env: ORCH_GITHUB_REPO
BaseBranch    string  // Optional. CLI: --base-branch, Env: ORCH_BASE_BRANCH, Default: "main"
TrackingLabel string  // Optional. CLI: --tracking-label, Env: ORCH_TRACKING_LABEL, Default: "ai-managed"
```

Validation: `GitHubOwner` and `GitHubRepo` are required (non-empty). `BaseBranch` and `TrackingLabel` use CLI defaults.

### Orchestrator (internal/orchestrator/orchestrator.go)

New field:
```go
github github.Client
```

Updated constructor:
```go
func New(cfg *Config, store state.StateStore, lc linear.Client,
    gh github.Client, wt worktree.Manager, sm cmux.SessionManager) *Orchestrator
```

Updated `Run()`: creates Router with `callbackSrv.Events()` and passes `router.Run` to shutdown manager.

### linear.Client (internal/linear/client.go)

New method:
```go
PostComment(ctx context.Context, issueID string, body string) error
```

### linear.HTTPClient (internal/linear/http_client.go)

New GraphQL mutation:
```go
const commentCreateMutation = `
mutation($input: CommentCreateInput!) {
  commentCreate(input: $input) {
    success
  }
}`
```

Implementation:
```go
func (c *HTTPClient) PostComment(ctx context.Context, issueID string, body string) error {
    var resp commentCreateResponse
    vars := map[string]any{
        "input": map[string]any{
            "issueId": issueID,
            "body":    body,
        },
    }
    if err := c.doWithRetry(ctx, commentCreateMutation, vars, &resp); err != nil {
        return fmt.Errorf("post comment: %w", err)
    }
    if !resp.CommentCreate.Success {
        return NewAPIError(fmt.Sprintf("linear: commentCreate failed for issue %s", issueID), nil)
    }
    return nil
}
```

### linear.MockClient (internal/linear/mock.go)

New field:
```go
PostCommentFn func(ctx context.Context, issueID string, body string) error
```

## Event Handler Flows

### handleComplete

```
GetJob(ticketID) → nil? discard
  ↓
ReadFile(.ai/pr-description.md) → missing? needs-attention
  ↓
GetTicket(ticketID) → build PR title
  ↓
CreatePR(title, body, branch, base) → fail? needs-attention
  ↓
SetPR(ticketID, prNumber)
  ↓
ApplyLabel(prNumber, trackingLabel)
  ↓
Archive(ticketID, EventComplete)
  ↓
Audit(ticketID, EventComplete)
```

### handleFailed

```
GetJob(ticketID) → may be nil, continue either way
  ↓
defer: ReleaseSlot(projectID)  ← always runs
  ↓
SetTicketStatus(ticketID, "needs-attention")
  ↓
if job != nil: SetJobStatus(ticketID, StatusNeedsAttention)
  ↓
PostComment(ticketID, reason)
  ↓
Archive(ticketID, EventFailed)
```

### handleCommentResolved

```
GetJob(ticketID) → nil? discard
  ↓
job.PRNumber → nil? needs-attention
  ↓
ParseInt64(event.CommentID) → fail? needs-attention
  ↓
ReadFile(.ai/pr-description.md) → missing? needs-attention
  ↓
UpdatePRBody(prNumber, body)
  ↓
PostCommentReply(prNumber, CommentKindReview, commentID, resolution)
  ↓
SetCommentWatermark(prNumber, commentID)
  ↓
Archive(ticketID, EventCommentResolved)
  ↓
Audit(ticketID, EventCommentResolved)
```

## Error Handling Pattern

Each handler follows commit-on-success semantics. On failure at any step:

1. Log the error with structured fields: `ticketID`, `eventKind`, `step` (name of the failed operation), `error`.
2. Call `markNeedsAttention(ctx, ticketID, job)` which:
   - Calls `LinearClient.SetTicketStatus(ctx, ticketID, "needs-attention")` -- logs and continues on failure.
   - If job is non-nil: calls `StateStore.SetJobStatus(ctx, ticketID, state.StatusNeedsAttention)` -- logs and continues on failure.
3. Return from the handler (earlier successful steps are preserved).

Exception: `handleFailed` uses a deferred `ReleaseSlot` call so the slot is always freed, even if the error-handling path itself fails.

## Testing Strategy

All tests use hand-rolled mocks following the existing pattern (`FnField` + `Calls []Call`).

**Test fixtures**:
- `testRouter(t, opts...)`: Creates a Router with mock dependencies, a writable event channel, and a temp directory for worktree simulation.
- Helper to write `.ai/pr-description.md` to the temp worktree.
- Helper to create a job in the mock store.

**Test pattern**:
1. Set up mock return values.
2. Write event to channel.
3. Close channel (or cancel context) to let `Run` exit.
4. Assert mock `Calls` slice for expected method invocations and arguments.
