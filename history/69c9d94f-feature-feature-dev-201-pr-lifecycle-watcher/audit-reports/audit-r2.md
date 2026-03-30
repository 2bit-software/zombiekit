# Audit Report R2: Watcher 3 — PR Lifecycle Detection and Cleanup

**Date**: 2026-03-29
**Spec Version**: R2 (after two audit rounds)

## Audit History

### Round 1 (R1)
- 2 CRITICAL, 7 MAJOR, 6 MINOR findings
- All CRITICAL and MAJOR issues resolved in R1 revisions

### Round 2 (R2) — Post-R1 Revisions
- 2 CRITICAL, 1 MAJOR, 2 MINOR findings (new issues from code verification)

#### R2 Critical Findings (Both Resolved)

**C1: Router does NOT set `StatusComplete` after `handleComplete`**
The spec assumed the router transitions jobs to `StatusComplete`. Verified in `router.go:96-154`: after creating the PR and storing the PR number, the job remains `StatusInProgress`. The detection strategy was rewritten to query `StatusInProgress` jobs with `PRNumber != nil`.

**C2: Router does NOT release slot on `EventComplete`**
The spec assumed slots were released by the router. Verified: only `handleFailed` and `handleCommentResolved` release slots. The slot from the initial session stays held until Watcher 3 releases it. Slot lifecycle documentation was rewritten.

#### R2 Major Finding (Resolved)

**M1: `CleanBranch` fallback has no viable branch name source**
`resolveBranch()` is unexported and requires the worktree to be present. If the worktree is already deleted, the branch name cannot be derived. Resolution: removed `CleanBranch` from the cleanup pipeline. `DeleteWorktree` handles both worktree and branch in the normal case. Orphaned branches are documented as a known limitation.

#### R2 Minor Findings (Accepted)

**m1: Merge vs close indistinguishable in job status**
Both paths set `StatusClosed`. Cannot distinguish "merged" from "closed without merge" from job status alone. Accepted — not needed for any current use case.

**m2: Config flag subcommand unspecified**
Resolved: spec now says `serve` subcommand.

## Current Status

All CRITICAL and MAJOR findings resolved. 2 MINOR findings accepted. Spec is ready for user review of key decisions.
