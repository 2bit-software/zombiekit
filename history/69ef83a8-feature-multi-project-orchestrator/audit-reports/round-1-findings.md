# Audit Round 1 Findings

## Summary

| Severity | Count | Status |
|----------|-------|--------|
| CRITICAL | 4 | All resolved in spec v2 |
| MAJOR | 9 | All resolved in spec v2 |
| MINOR | 7 | 6 resolved, 1 accepted (backoff not configurable — intentional) |

## Critical Findings (all resolved)

1. **StateStore interface underspecified** — Full updated interface now in tech spec with every method signature, including new `GetJobByTicketID` for backward compat
2. **Migration backfill strategy unresolved** — Resolved: `--migrate-project-id` CLI flag, required when empty rows exist
3. **Callback URL backward compat unresolved** — Resolved: dual URL pattern support (old resolves via job lookup)
4. **EventDemuxer lifecycle unspecified** — Full API now defined: Register/Deregister, Run loop, buffer sizing (64/project), shutdown behavior, startup ordering

## Major Findings (all resolved)

1. **ListJobsByStatus unscoped** — Added projectID param to interface
2. **Slot release gap in handleComplete** — Added as explicit bug fix in acceptance criteria
3. **Credential inheritance contradiction** — Resolved: per-project optional with global fallback, removed from Out of Scope
4. **Reconciliation not addressed** — Added: runs globally at startup, detects orphaned jobs
5. **Two-tier shutdown semantics** — Resolved: ProjectRunner never returns error, retries forever, health surfaced via /healthz
6. **ProjectRunner extraction boundary** — Full type definition with shared vs per-project deps
7. **Config validation timing** — All at parse time, fail-fast before any project starts
8. **runWithRestart backoff reset** — Reset after successful run of 1 poll interval
9. **Router's store calls after composite PK** — ProjectID flows through Event struct from callback URL

## Minor Findings

- TOML struct tags: specified as required for all fields
- Reconciliation timing: global, once, before projects start
- Linear label constants: acknowledged, not changed (tracking_label is for GitHub, not Linear)
- copy_files validation: not added (existing behavior, low risk)
- Health endpoint: single /healthz with per-project JSON
- Admin commands: keep --db-path for now, note for future --config support
- Backoff parameters: intentionally hard-coded (1s-2min), not configurable
