# Initiative: graphite-stack-branching

**Type**: feature
**Status**: in_progress
**Created**: 2026-04-03
**ID**: 69d03418-feature-graphite-stack-branching

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-04-03 15:20 |
| plan | completed | 2026-04-03 15:45 |
| tasks | completed | 2026-04-03 15:50 |
| implement | in_progress | 2026-04-03 15:50 |

## Description

Add graphite stacking support to the MCP initiative creation flow. When a user signals stacking intent (via prompt keyword or being on an already-stacked branch), branches are created using `gt create` instead of `git checkout -b`. A startup hook reports graphite availability, repo initialization, and current stack status upfront.

## Goals

- Enable graphite-stacked branch creation during `/brains.new`
- Detect stacking intent from prompt keywords or current branch stack status
- Report graphite status via startup hook for workflow awareness
- Maintain full backward compatibility for non-graphite flows

## Progress

- [x] Research phase complete
- [x] Spec written and audited
- [x] Plan phase complete
- [x] Implementation complete
