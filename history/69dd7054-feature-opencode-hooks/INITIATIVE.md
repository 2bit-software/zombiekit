# Initiative: opencode-hooks

**Type**: feature
**Status**: in_progress
**Created**: 2026-04-13
**ID**: 69dd7054-feature-opencode-hooks

## Steps

| Step | Status | Updated |
|------|--------|--------|
| spec | completed | 2026-04-13 16:02 |
| plan | completed | 2026-04-13 16:18 |
| tasks | completed | 2026-04-13 16:30 |
| implement | in_progress | 2026-04-13 16:30 |

## Description

Add OpenCode as a third supported editor for the `brains hook` rule-injection
pipeline, parallel to existing Claude Code and Gemini CLI support. OpenCode's
plugin model is in-process JS/TS, not subprocess, so the integration has two
pieces: a new `opencode` editor in `internal/hook/` and a `.ts` shim plugin
that bridges OpenCode's mutable-output contract to `brains` over stdin/stdout.

## Goals

- Match feature parity with the Gemini integration for file-edit rule injection.
- Keep all path globbing, rule resolution, and session dedup in the shared
  handler — no duplication per editor.
- Ship a raw `.ts` shim that the user hand-copies into `.opencode/plugins/`.
  An install subcommand is explicitly out of scope for this iteration.
- Support a side-installed `brains-test` binary during development so the
  user can swap the binary without disturbing the running Claude Code session.

## Progress

- 2026-04-13: spec drafted. Critical finding during research: OpenCode does
  not have subprocess hooks — integration requires a JS/TS shim in addition
  to a new editor. User confirmed option 1 (shim) with raw-script
  distribution.
