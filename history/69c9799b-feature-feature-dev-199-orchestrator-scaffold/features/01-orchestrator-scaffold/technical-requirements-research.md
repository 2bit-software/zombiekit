# Technical Requirements & Implementation Hints

Extracted from DEV-199 and the Agent Context Brief. These are implementation preferences, NOT business requirements.

## From the Ticket

- Config struct fields: Linear API key, GitHub token, callback port (default 8666), worktrees root path, DB path, concurrency limit per project (default 1)
- Config loading from env vars and/or config file
- Poll interval for all watchers should be configurable from the config struct
- Startup sequence order is enforced: reconciliation must complete before watchers start polling. Use sequential init, not concurrent.
- Stop channels (or context cancellation) are the coordination mechanism between main and watchers. Don't use global state.

## From the Agent Context Brief

- Orchestrator lives in `cmd/orchestrator/`
- It is a long-running Go daemon
- Component package layout:
  ```
  cmd/orchestrator/
    main.go               -- startup, reconciliation, launch watchers
  internal/
    state/                -- StateStore interface + SQLite implementation
    linearclient/         -- LinearClient interface + HTTP implementation
    githubclient/         -- GitHub PR manager
    worktree/             -- git worktree manager
    cmux/                 -- cmux session manager
    callback/             -- HTTP callback server
    archival/             -- conversation archival
    friction/             -- friction auditor
    orchestrator/         -- core watcher loops + post-session processing
  ```
- Three watcher goroutines (types not specified in this ticket -- just launch points)
- HTTP callback server on fixed port (default 8666)
- On-disk state via SQLite (StateStore interface)
- No auto-retry on failure
- Serial comment processing per PR
- Hard boundary: orchestrator knows infra, agent knows code

## Technical Preferences (Inferred)

- Structured logging (slog) -- project already uses this pattern
- Context cancellation for goroutine coordination (not global state)
- Sequential startup with fail-fast on errors
- Graceful drain with timeout on shutdown
