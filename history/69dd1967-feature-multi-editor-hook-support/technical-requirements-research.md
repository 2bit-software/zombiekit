# Technical Requirements & Research

**Status**: stub — research in progress

## User-Specified Technical Preferences

From the initiating request:

- Add a `--<codingEnvironment>` flag to each hook invocation so zombiekit knows which editor's schema to parse and which response contract to emit.
- Audit Gemini CLI's actual hook format — do not assume the current "plain markdown fallback" is correct.
- Design for extension: OpenCode and other editors will follow.

## Research Questions

### Gemini CLI hook protocol
- Does `gemini` CLI ship a hook system analogous to Claude Code's `settings.json` hooks?
- What is the JSON schema of the event payload sent to hook commands?
- What output format does Gemini expect? stdout text? JSON envelope? exit code semantics?
- Which events exist (SessionStart, PreToolUse, etc.)?
- Where is this documented (repo, docs site)?

### Current zombiekit hook surface
- What does `internal/cli/hook.go` + `internal/hook/*` currently handle?
- Where is the Claude-specific response envelope emitted?
- How does `DetectAgent()` work today and where is it called?
- What does the audit record look like and does it already carry `Agent`?

### Extension points
- Pattern for per-editor parsers and formatters — interface + registry, or switch?
- Naming: `--claude`/`--gemini` booleans vs. `--editor claude` string — which fits urfave/cli better and keeps future additions clean?

## Findings

_To be populated by research agents._
