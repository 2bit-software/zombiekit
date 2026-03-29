# Completeness Audit — Round 1

## CRITICAL (1)

1. **Unresolved D3: comment types to process** — Review only, or both review + issue comments? `GetCommentsSince` requires `CommentKind`. Separate ID spaces for each kind affect watermark semantics.

## MAJOR (6)

2. **Unresolved D4: bot username not configured** — FR-006 requires filtering bot comments but no config field exists.
3. **Unresolved D5: comment threading re-entry** — Reviewer follow-up replies after resolution will be re-enqueued (higher ID). No user story covers this.
4. **No FR for watermark advancement** — Who advances the watermark? DEV-202 on success, but what about on failure/drain?
5. **FR-009 missing watermark behavior on failure** — Must advance watermark before clearing queue to prevent retry loops.
6. **Ambiguous event flow terminology** — "When processed by T3/callback router" — who fires events? How does Watcher 2 learn about outcomes?
7. **No notification mechanism specified** — How does Watcher 2 learn about FailureEvent/CommentResolvedEvent?

## MINOR (4)

8. FR-004 session payload mechanism unspecified (worktree file vs env)
9. PR goroutine reaping for removed PRs not in FRs
10. Manual recovery interaction with watermark undocumented
11. IsClosed path missing from FR-008 test mapping
