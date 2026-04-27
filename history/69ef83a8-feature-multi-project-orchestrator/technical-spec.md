# Technical Spec: Multi-Project Orchestrator

## Duration Wrapper (BurntSushi/toml compatibility)

BurntSushi/toml v1.6.0 does NOT decode `time.Duration` from TOML strings. A wrapper type is required.

```go
// Duration wraps time.Duration for TOML unmarshaling.
type Duration struct{ time.Duration }

func (d *Duration) UnmarshalText(text []byte) error {
    var err error
    d.Duration, err = time.ParseDuration(string(text))
    return err
}

func (d Duration) MarshalText() ([]byte, error) {
    return []byte(d.String()), nil
}
```

Used in `GlobalConfig.PollInterval`, `GlobalConfig.ShutdownTimeout`. The `ProjectConfig` receives a copied `PollInterval` during `applyDefaults()` so ProjectRunner doesn't need GlobalConfig access.

## Config Types (final)

```go
type OrchestratorConfig struct {
    Global   GlobalConfig    `toml:"global"`
    Projects []ProjectConfig `toml:"project"`
}

type GlobalConfig struct {
    LinearAPIKey    string   `toml:"linear_api_key"`
    GitHubToken     string   `toml:"github_token"`
    CallbackPort    int      `toml:"callback_port"`
    DBPath          string   `toml:"db_path"`
    PollInterval    Duration `toml:"poll_interval"`
    LogLevel        string   `toml:"log_level"`
    LogJSON         bool     `toml:"log_json"`
    ShutdownTimeout Duration `toml:"shutdown_timeout"`
    BotUsername     string   `toml:"bot_username"`
    Sandbox         string   `toml:"sandbox"`
}

type ProjectConfig struct {
    ID               string   `toml:"id"`
    LinearAPIKey     string   `toml:"linear_api_key"`
    LinearProjectID  string   `toml:"linear_project_id"`
    GitHubToken      string   `toml:"github_token"`
    GitHubOwner      string   `toml:"github_owner"`
    GitHubRepo       string   `toml:"github_repo"`
    RepoDir          string   `toml:"repo_dir"`
    WorktreesRoot    string   `toml:"worktrees_root"`
    BaseBranch       string   `toml:"base_branch"`
    TrackingLabel    string   `toml:"tracking_label"`
    ConcurrencyLimit int      `toml:"concurrency_limit"`
    CopyFiles        []string `toml:"copy_files"`
    ClosedPRStatus   string   `toml:"closed_pr_status"`

    // Copied from GlobalConfig during applyDefaults()
    PollInterval Duration `toml:"-"`
    CallbackPort int      `toml:"-"`
    BotUsername  string   `toml:"-"`
    SandboxMode  string   `toml:"-"` // "auto"/"enabled"/"disabled"
}
```

Fields with `toml:"-"` are not parsed from TOML — they're injected during config post-processing.

## ProjectRunner (final)

```go
type ProjectRunner struct {
    id         string
    cfg        ProjectConfig
    globalCfg  GlobalConfig
    store      state.StateStore
    linear     linear.Client
    github     github.Client
    worktrees  worktree.Manager
    sessions   cmux.SessionManager
    events     <-chan callback.Event
    dispatcher *CommentDispatcher
    logger     *slog.Logger

    // Health tracking
    mu           sync.RWMutex
    watcherState map[string]*watcherHealth
}

type watcherHealth struct {
    LastSuccess   time.Time
    LastError     time.Time
    LastErrorMsg  string
    ConsecFails   int
    CurrentBackoff time.Duration
}

type ProjectHealth struct {
    ID        string                    `json:"id"`
    Healthy   bool                      `json:"healthy"`
    Watchers  map[string]WatcherStatus  `json:"watchers"`
}

type WatcherStatus struct {
    Status     string `json:"status"`      // "running", "restarting", "unhealthy"
    LastError  string `json:"last_error,omitempty"`
    BackoffSec int    `json:"backoff_sec,omitempty"`
}
```

## Composition Root Wiring

```
                    ┌─────────────┐
                    │ TOML Config │
                    └──────┬──────┘
                           │
              ┌────────────┼────────────┐
              │            │            │
              ▼            ▼            ▼
        ┌──────────┐ ┌──────────┐ ┌──────────┐
        │ Linear   │ │ Callback │ │ SQLite   │
        │ Client   │ │ Server   │ │ Store    │
        │ (shared) │ │ (shared) │ │ (shared) │
        └────┬─────┘ └────┬─────┘ └────┬─────┘
             │             │            │
             │      ┌──────┴──────┐     │
             │      │ EventDemuxer│     │
             │      │  (shared)   │     │
             │      └──┬───┬───┬──┘     │
             │         │   │   │        │
    ┌────────┼─────────┤   │   ├────────┼────────┐
    │        │         │   │   │        │        │
    ▼        ▼         ▼   │   ▼        ▼        ▼
┌─────────────────────┐│┌─────────────────────┐
│   ProjectRunner A   │││   ProjectRunner B   │
│ ┌─────────────────┐ │││ ┌─────────────────┐ │
│ │ GitHub Client A │ │││ │ GitHub Client B │ │
│ │ Worktree Mgr A  │ │││ │ Worktree Mgr B  │ │
│ │ CommentDisp A   │ │││ │ CommentDisp B   │ │
│ ├─────────────────┤ │││ ├─────────────────┤ │
│ │ LinearPoller    │ │││ │ LinearPoller    │ │
│ │ PRWatcher       │ │││ │ PRWatcher       │ │
│ │ CommentWatcher  │ │││ │ CommentWatcher  │ │
│ │ EventRouter     │ │││ │ EventRouter     │ │
│ └─────────────────┘ │││ └─────────────────┘ │
└─────────────────────┘│└─────────────────────┘
                       │
              ┌────────┴────────┐
              │ shutdown.Manager│
              │   (errgroup)    │
              └─────────────────┘
```

## shutdown.Manager Service List

```
Services in top-level errgroup:
1. callbackSrv.Run          (infra — failure kills process)
2. demuxer.Run              (infra — failure kills process)
3. projectRunnerA.RunSupervised  (per-project — never returns error)
4. projectRunnerB.RunSupervised  (per-project — never returns error)
...
```

## State Store Query Changes

Every query that currently does `WHERE ticket_id = ?` becomes `WHERE project_id = ? AND ticket_id = ?`.

Key query changes:

| Method | Current WHERE | New WHERE |
|--------|--------------|-----------|
| `GetJob` | `ticket_id = ?` | `project_id = ? AND ticket_id = ?` |
| `GetJobByPR` | `pr_number = ?` | `project_id = ? AND pr_number = ?` |
| `ListJobsByStatus` | `status IN (?)` | `project_id = ? AND status IN (?)` |
| `DeleteJob` | `ticket_id = ?` | `project_id = ? AND ticket_id = ?` |
| `SetJobStatus` | `ticket_id = ?` | `project_id = ? AND ticket_id = ?` |
| `SetPR` | `ticket_id = ?` | `project_id = ? AND ticket_id = ?` |
| `GetCommentWatermark` | `pr_number = ?` | `project_id = ? AND pr_number = ?` |
| `SetCommentWatermark` | `pr_number = ?` | `project_id = ? AND pr_number = ?` |
| `ListAllJobs` | (none) | (none) — unchanged |

## Slot Release Behavior (documented, not changed)

```
Ticket lifecycle:
1. LinearPoller: TryAcquireSlot(projectID, limit) → slot held
2. Agent runs in worktree
3. handleComplete: creates PR → slot STAYS held (intentional)
4. Comment on PR → CommentWatcher: TryAcquireSlot → NEW slot for comment handling
5. handleCommentResolved: ReleaseSlot → releases comment's slot
6. PR merged/closed → PRWatcher: ReleaseSlot → releases ORIGINAL slot from step 1
```

The slot from step 1 represents "this project has an active PR." It prevents the concurrency limit from being exceeded. This is correct behavior.

## Reconciliation Flow

```
ApplyReconciliation(ctx, store, configuredProjectIDs, logger):
  1. allJobs = store.ListAllJobs(ctx)
  2. For each job where job.ProjectID not in configuredProjectIDs:
       - Warn: orphaned job
       - store.ReleaseSlot(ctx, job.ProjectID)
  3. For each configuredProjectID:
       - jobs = store.ListJobsByStatus(ctx, projectID, StatusInProgress)
       - PlanReconciliation(jobs, now) → classify stale jobs
       - For each stale job: SetJobStatus(ctx, projectID, ticketID, StatusNeedsAttention)
  4. store.ResetAllSlots(ctx) — zero all slot counts (fresh start)
```

## Health Endpoint Response

```json
{
  "status": "healthy",
  "projects": {
    "zombiekit": {
      "id": "zombiekit",
      "healthy": true,
      "watchers": {
        "linear-poller": {"status": "running"},
        "pr-watcher": {"status": "running"},
        "comment-watcher": {"status": "running"},
        "event-router": {"status": "running"}
      }
    },
    "clawbeam": {
      "id": "clawbeam",
      "healthy": false,
      "watchers": {
        "linear-poller": {"status": "unhealthy", "last_error": "401 Unauthorized", "backoff_sec": 120},
        "pr-watcher": {"status": "running"},
        "comment-watcher": {"status": "running"},
        "event-router": {"status": "running"}
      }
    }
  }
}
```

Global `status` is `"healthy"` if all projects are healthy, `"degraded"` if any project is unhealthy.
