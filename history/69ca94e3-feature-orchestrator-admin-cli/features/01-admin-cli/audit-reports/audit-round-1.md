# Audit Round 1: Completeness + AI-Consumability

**Date**: 2026-03-30
**Result**: All CRITICAL and MAJOR findings resolved. Spec updated.

## Findings and Resolutions

### CRITICAL (all resolved)

| # | Finding | Resolution |
|---|---------|------------|
| C1 | No `project_id` on Job — can't release slot on delete | Added FR-013: migration adds `project_id` column. User decision. |
| C2 | `ListAllJobs()` missing from interface | Added to Architecture section with exact signature and SQL |
| C3 | `DeleteJob()` missing from interface | Added to Architecture section with exact signature and SQL |
| C4 | `ListSlots()` missing from interface + no `ConcurrencySlot` type | Added both to Architecture section |

### MAJOR (all resolved)

| # | Finding | Resolution |
|---|---------|------------|
| M1 | Status validation not specified (store vs. service) | FR-005: validation in admin service, not store. Allowlist only, no transition rules. |
| M2 | Daemon migration not in spec | Added FR-012: daemon moves to `orchestrator run` |
| M3 | `slots reset` scope ambiguous | FR-007: explicitly global. Single-project scoping out of scope. |
| M4 | `--status` flag syntax unspecified | FR-002: repeated flags via `StringSliceFlag` |
| M5 | Service layer vs. store extension contradictory | Resolved: user chose `internal/admin` service layer. Store gets CRUD, service gets compound ops. |

### MINOR (accepted as-is or addressed)

| # | Finding | Disposition |
|---|---------|-------------|
| m1 | No sort order for lists | Added: `ORDER BY updated_at DESC` for jobs |
| m2 | No tests for FR-010/FR-011 | FR-011 covered by test strategy. FR-010 tested manually. |
| m3 | Timestamp format unspecified | Added: RFC 3339 truncated to seconds, local timezone |
| m4 | `--db-path` default unspecified | Added: required flag, error if missing |
| m5 | No `--help` scenario | Accepted: urfave/cli provides this automatically |
| m6 | Delete active job has no `--force` | Accepted: manual recovery tool, operator knows what they're doing |
| m7 | Orphaned comment_watermarks on delete | Accepted: watermarks are idempotent, no cleanup needed |
| m8 | Output format had no examples | Added: tabwriter columns for lists, Key: Value for detail, examples in Architecture |
| m9 | Confirmation message format unspecified | Added: FR-010 now has explicit format and examples |
