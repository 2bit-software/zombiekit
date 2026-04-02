# Feature Specification: Session-Aware Rules Injection

**Feature Branch**: `69cdc5b6-feature-session-aware-rules-injection`  
**Created**: 2026-04-01  
**Status**: Draft  
**Input**: User description: "Replace built-in rules handling with a session-aware hook system that injects rules at file read/write time, tracks injection state per session, handles compaction resets, and works across Claude Code and Gemini CLI."

## Resolved Decisions

| # | Decision | Resolution |
|---|----------|------------|
| D1 | Rules frontmatter format | Superset of Claude Code's `paths` field. v1: `paths` only, identical semantics. |
| D2 | Session state backend | JSON file at `/tmp/zk-session-{SESSION_ID}.json` |
| D3 | Rules file location | `.brains/rules/` (project) + `~/.brains/rules/` (global). NOT `.claude/rules/` — Claude handles its own rules. |
| D4 | Injection format | Plain text to stdout. No JSON wrapping. No `additionalContext`. |
| D5 | Hook binary form | `brains hook` subcommand of existing CLI |
| D6 | Which hooks to register | SessionStart, PostToolUse (Read\|Write\|Edit\|MultiEdit), SessionEnd |
| D7 | Compaction reset | Full reset of injected rules set. Unconditional rules re-inject on next SessionStart. Path-scoped rules re-inject on next matching file operation. |
| D8 | Conflict resolution | Composition — project and global rules with same filename are both injected independently. No shadowing. |
| D9 | SessionStart resume | Reset tracking. Do not assume rules persist across resume. |
| D10 | Custom API agents | Out of scope for MVP. Claude Code + Gemini CLI only. |
| D11 | Bash file creation | Out of scope. Cannot reliably extract file paths from arbitrary shell commands. |

## User Scenarios & Testing

### User Story 1 - Rules Injected on File Read (Priority: P1)

A developer reads a `.go` file in their AI coding session. The system detects the file extension, finds matching rules (e.g., Go coding standards), and injects them into the agent's context. The developer hasn't seen these rules yet this session, so they're injected in full. The agent now has Go-specific guidance *before* attempting any edits.

**Why this priority**: This is the core differentiator — injecting rules at read time so agents make informed decisions before writing. Without this, the entire feature has no value.

**Independent Test**: Create a Go rules file with `paths: ["**/*.go"]`. Start a session, read a `.go` file via the agent. Verify the rules content appears in the agent's context via hook stdout.

**Acceptance Scenarios**:

1. **Given** a rules file exists with `paths: ["**/*.go"]`, **When** the agent reads a `.go` file (PostToolUse on Read), **Then** the matching rules are printed to stdout as plain text.
2. **Given** no rules file matches `*.md`, **When** the agent reads a `.md` file, **Then** no output is produced (exit 0, empty stdout).
3. **Given** a rules file has no `paths` frontmatter, **When** SessionStart fires (any source), **Then** the rules are injected unconditionally.

---

### User Story 2 - Deduplication Within a Session (Priority: P1)

A developer reads two `.go` files in the same session. The Go rules are injected on the first read. On the second read, the system recognizes that the Go rules have already been injected this session and skips re-injection. This prevents context bloat from repeated rules.

**Why this priority**: Without deduplication, every file read would inject rules, rapidly filling the context window and degrading agent performance. This is a hard requirement for the feature to be usable.

**Independent Test**: Start a session, read `foo.go` (rules injected), then read `bar.go` (rules NOT re-injected). Verify via session state file that the rules were tracked as "already injected."

**Acceptance Scenarios**:

1. **Given** Go rules were injected earlier this session, **When** the agent reads another `.go` file, **Then** the rules are NOT re-injected (exit 0, empty stdout).
2. **Given** Go rules were injected and Python rules were not, **When** the agent reads a `.py` file, **Then** Python rules ARE injected (Go rules are not).
3. **Given** a file matches multiple rule sets, **When** the agent reads it, **Then** only the rule sets not yet injected this session are printed to stdout.

---

### User Story 3 - Re-injection After Compaction (Priority: P1)

During a long session, the context is compacted (compressed). After compaction, the previously injected rules are no longer in context. The system detects the compaction event, resets its tracking state, and re-injects unconditional rules immediately. Path-scoped rules re-inject on the next matching file operation.

**Why this priority**: Compaction erases injected rules from context. Without re-injection, the agent loses all rules guidance for the remainder of the session. This is a correctness requirement.

**Independent Test**: Inject Go rules, trigger compaction, then read another `.go` file. Verify rules are re-injected.

**Acceptance Scenarios**:

1. **Given** rules were injected and then compaction occurs, **When** `SessionStart` fires with `source: "compact"`, **Then** the session's "injected rules" set is cleared AND unconditional rules are re-injected immediately (printed to stdout).
2. **Given** the injected set was cleared by compaction, **When** the agent reads a `.go` file, **Then** Go rules are re-injected as if it were the first time.
3. **Given** compaction occurs and unconditional rules exist, **When** `SessionStart source=compact` fires, **Then** unconditional rules are printed to stdout. Path-scoped rules are NOT printed (they inject on next matching file operation).

---

### User Story 4 - Write-Time and MultiEdit Injection (Priority: P2)

When the agent writes, edits, or multi-edits a file type it hasn't previously read in this session, rules are injected at that time. This catches cases where the agent generates a new file without first reading an existing one of that type. MultiEdit operations extract each file path independently and inject rules for any unmatched types.

**Why this priority**: Read-time injection covers most cases, but agents can write files without reading first (e.g., creating a new file from scratch). Write/edit-time injection is the safety net. MultiEdit handles batch operations.

**Independent Test**: Start a session, write a new `.py` file without having read any `.py` file. Verify Python rules are injected at write time. Then multi-edit two `.ts` files — verify TypeScript rules inject once.

**Acceptance Scenarios**:

1. **Given** no `.py` file has been read this session, **When** the agent writes a `.py` file, **Then** Python rules are injected.
2. **Given** Python rules were already injected (via a read), **When** the agent writes a `.py` file, **Then** rules are NOT re-injected.
3. **Given** a MultiEdit touches `foo.go` and `bar.ts`, **When** PostToolUse fires for MultiEdit, **Then** rules for both Go and TypeScript are injected (if not already in session). Each file path in the edit list is matched independently.

---

### User Story 5 - Agent-Agnostic Operation (Priority: P2)

The same rules files and `brains hook` subcommand work for both Claude Code and Gemini CLI sessions. The system detects which agent is running via environment variables and adapts the output format accordingly.

**Why this priority**: Multi-agent support is the strategic motivation for building this. Single-agent support could be achieved with Claude Code's built-in system.

**Independent Test**: Run `brains hook` with Claude Code environment variables, verify output includes `<system-reminder>` wrapping. Run with Gemini CLI environment variables, verify plain markdown output.

**Acceptance Scenarios**:

1. **Given** `CLAUDE_SESSION_ID` is set, **When** the hook runs, **Then** rules output is wrapped in `<system-reminder>` tags.
2. **Given** `GEMINI_SESSION_ID` is set, **When** the hook runs, **Then** rules output is plain markdown (no XML wrapping).
3. **Given** both `CLAUDE_SESSION_ID` and `GEMINI_SESSION_ID` are set, **When** the hook runs, **Then** Claude Code takes precedence (check `CLAUDE_SESSION_ID` first).

---

### User Story 6 - Rules File Authoring (Priority: P3)

A developer creates rules files in `.brains/rules/` (project-scoped) or `~/.brains/rules/` (global) using the same frontmatter format as Claude Code's `.claude/rules/`. Rules use the `paths` frontmatter field for file-type matching.

**Why this priority**: Authoring is a one-time setup cost. The format is a superset of Claude Code's, so existing rules can be copied directly.

**Independent Test**: Place a rules file in `.brains/rules/go.md` with `paths: ["**/*.go"]`. Read a `.go` file. Verify rules are injected.

**Acceptance Scenarios**:

1. **Given** a rules file exists in `.brains/rules/` with `paths: ["**/*.go"]`, **When** a `.go` file is read, **Then** the rules are injected.
2. **Given** `go.md` exists in both `.brains/rules/` (project) and `~/.brains/rules/` (global), **When** a `.go` file is read, **Then** BOTH are injected (composition, not shadowing). Project rules appear first in output.
3. **Given** a rules file uses brace expansion `paths: ["**/*.{ts,tsx}"]`, **When** a `.tsx` file is read, **Then** the rules are injected.

---

### User Story 7 - Session Lifecycle (Priority: P2)

Session state is initialized on SessionStart, tracked across hook invocations, reset on compaction and resume, and cleaned up on SessionEnd.

**Why this priority**: Without proper lifecycle management, session state files accumulate in `/tmp` and stale state causes incorrect deduplication.

**Independent Test**: Start a session (verify state file created), read files (verify state updated), end session (verify state file deleted).

**Acceptance Scenarios**:

1. **Given** a new session starts (`SessionStart source=startup`), **When** the hook runs, **Then** a new session state file is created at `/tmp/zk-session-{SESSION_ID}.json` and unconditional rules are printed to stdout.
2. **Given** a session resumes (`SessionStart source=resume`), **When** the hook runs, **Then** the injected rules set is reset (cleared) and unconditional rules are re-injected. Do not assume rules persist across resume.
3. **Given** a session ends (`SessionEnd`), **When** the hook runs, **Then** the session state file is deleted.
4. **Given** the session state file is missing or corrupted mid-session, **When** any hook runs, **Then** a fresh state file is created and the hook proceeds as if it were a new session.

---

### Edge Cases

- **Multiple rules match one file**: All matching rules are injected (those not already in the session). Composed, not deduplicated.
- **Corrupted/missing session state**: Treat as a fresh session — create new state, inject all matching rules.
- **Hook binary not found or crashes**: Agent session continues without rules injection. Exit code != 0 is non-blocking per hook protocol.
- **Rules files change mid-session**: Changes take effect on next fresh injection (after compaction resets tracking). Deduplication prevents re-injection of a rule that was already injected with different content.
- **Empty rules file (frontmatter only, no body)**: Nothing injected (no content to output).
- **File path with spaces/special characters**: Glob matching handles correctly (gitignore-style).
- **Bash tool creates files**: Out of scope — cannot reliably extract paths from arbitrary shell commands.
- **Race condition (parallel file reads)**: JSON file read-modify-write is not atomic. Accept occasional missed deduplication — correctness favors injecting twice over missing injection.
- **Rules directory doesn't exist**: Skip silently, produce no output.

## Hook Binary Interface Contract

### Invocation

```
brains hook --event <event-type>
```

Event types: `session-start`, `post-tool-use`, `session-end`

The hook reads JSON from stdin (provided by the agent's hook system) and writes plain text rules to stdout.

### Exit Codes

| Code | Meaning | Behavior |
|------|---------|----------|
| 0 | Success | Stdout content (if any) is added to agent's context |
| 1 | Error (non-blocking) | Agent continues, stderr shown in verbose mode |

Exit code 2 is never used (no blocking behavior needed for PostToolUse hooks).

### Stdin JSON Schemas

**SessionStart** (source: startup, resume, or compact):
```json
{
  "session_id": "abc123-def456",
  "hook_event_name": "SessionStart",
  "cwd": "/path/to/project",
  "source": "startup"
}
```

**PostToolUse — Read**:
```json
{
  "session_id": "abc123-def456",
  "hook_event_name": "PostToolUse",
  "cwd": "/path/to/project",
  "tool_name": "Read",
  "tool_input": {
    "file_path": "/path/to/project/main.go"
  }
}
```

**PostToolUse — Write**:
```json
{
  "session_id": "abc123-def456",
  "hook_event_name": "PostToolUse",
  "cwd": "/path/to/project",
  "tool_name": "Write",
  "tool_input": {
    "file_path": "/path/to/project/handler.go",
    "content": "..."
  },
  "tool_response": {
    "filePath": "/path/to/project/handler.go",
    "success": true
  }
}
```

**PostToolUse — Edit**:
```json
{
  "session_id": "abc123-def456",
  "hook_event_name": "PostToolUse",
  "cwd": "/path/to/project",
  "tool_name": "Edit",
  "tool_input": {
    "file_path": "/path/to/project/handler.go",
    "old_string": "...",
    "new_string": "..."
  },
  "tool_response": {
    "filePath": "/path/to/project/handler.go",
    "success": true
  }
}
```

**PostToolUse — MultiEdit**:
```json
{
  "session_id": "abc123-def456",
  "hook_event_name": "PostToolUse",
  "cwd": "/path/to/project",
  "tool_name": "MultiEdit",
  "tool_input": {
    "edits": [
      {"file_path": "/path/to/project/main.go", "old_string": "...", "new_string": "..."},
      {"file_path": "/path/to/project/handler.ts", "old_string": "...", "new_string": "..."}
    ]
  }
}
```

**SessionEnd**:
```json
{
  "session_id": "abc123-def456",
  "hook_event_name": "SessionEnd",
  "cwd": "/path/to/project"
}
```

### File Path Extraction

The hook binary extracts file paths from stdin JSON using these rules:

| Tool | Primary Path | Fallback Path |
|------|-------------|---------------|
| Read | `tool_input.file_path` | — |
| Write | `tool_input.file_path` | `tool_response.filePath` |
| Edit | `tool_input.file_path` | `tool_response.filePath` |
| MultiEdit | `tool_input.edits[].file_path` | — |

Note: `tool_input` uses `snake_case` (`file_path`), `tool_response` uses `camelCase` (`filePath`).

### Stdout Format

Plain text. The hook binary writes matched rules content directly to stdout. When multiple rules match, they are concatenated with a blank line separator inside a single wrapper (to minimize token overhead). Rules with empty body (frontmatter only, no content) are skipped.

**For Claude Code** (`CLAUDE_SESSION_ID` env var is set):
```
<system-reminder>
# Go Standards

- Use `any` instead of `interface{}`
- Always check errors with context

# General Coding Standards

- Compare alternative approaches with pros and cons
</system-reminder>
```

Single `<system-reminder>` wrapper around all concatenated rules.

**For Gemini CLI** (`GEMINI_SESSION_ID` env var is set):
```
# Go Standards

- Use `any` instead of `interface{}`
- Always check errors with context

# General Coding Standards

- Compare alternative approaches with pros and cons
```

Plain markdown, rules concatenated with blank line separator.

**Agent detection priority**: Check `CLAUDE_SESSION_ID` first, then `GEMINI_SESSION_ID`.

### Session State File

Location: `/tmp/zk-session-{SESSION_ID}.json`

```json
{
  "session_id": "abc123-def456",
  "agent": "claude",
  "started_at": "2026-04-01T10:00:00Z",
  "compaction_count": 0,
  "injected_rules": {
    "project:go.md": "2026-04-01T10:01:00Z",
    "global:coding-general.md": "2026-04-01T10:00:05Z"
  }
}
```

**Rule identity key**: Filename (e.g., `go.md`). Rules from different directories with the same filename are tracked independently using `{source}:{filename}` format (e.g., `project:go.md`, `global:go.md`).

### Hook Registration (Claude Code settings.json)

```json
{
  "hooks": {
    "SessionStart": [{
      "matcher": "startup|resume|compact",
      "command": "brains hook --event session-start"
    }],
    "PostToolUse": [{
      "matcher": "Read|Write|Edit|MultiEdit",
      "command": "brains hook --event post-tool-use"
    }],
    "SessionEnd": [{
      "command": "brains hook --event session-end"
    }]
  }
}
```

## Requirements

### Functional Requirements

- **FR-001**: System MUST inject matching rules into agent context when a file is read (PostToolUse on Read).
- **FR-002**: System MUST inject matching rules into agent context when a file is written or edited (PostToolUse on Write|Edit|MultiEdit) if those rules haven't been injected yet this session.
- **FR-003**: System MUST track which rule sets have been injected per session using a session state file at `/tmp/zk-session-{SESSION_ID}.json`.
- **FR-004**: System MUST reset the "injected rules" set when compaction or resume is detected (SessionStart source=compact or source=resume).
- **FR-005**: System MUST match rules to files using gitignore-style glob patterns in the `paths` frontmatter field.
- **FR-006**: System MUST inject unconditional rules (no `paths` field) when SessionStart fires with any source (startup, resume, compact).
- **FR-007**: System MUST resolve rules by walking from `cwd` up to git root, collecting `.brains/rules/` directories at each level (ancestor walk), then `~/.brains/rules/` (global). All sources are composed independently — no shadowing. This supports monorepos where subdirectories have their own rules.
- **FR-008**: System MUST detect the running agent via environment variables (`CLAUDE_SESSION_ID` checked first, then `GEMINI_SESSION_ID`) and wrap output in `<system-reminder>` tags for Claude Code.
- **FR-009**: System MUST use the `paths` frontmatter field with identical semantics to Claude Code rules (string or string[], gitignore globs, brace expansion).
- **FR-010**: System MUST respond within 100ms p99 to avoid perceptible latency on hook invocations.
- **FR-011**: System MUST handle missing or corrupted session state gracefully by treating it as a fresh session.
- **FR-012**: System MUST delete the session state file when SessionEnd fires.
- **FR-013**: For MultiEdit events, the system MUST extract each file path from the `edits` array and match rules independently per path.
- **FR-014**: System MUST be invokable as `brains hook --event <event-type>` subcommand of the existing brains CLI.

### Key Entities

- **Rule**: A markdown file with optional `paths` YAML frontmatter. Contains guidance text for the agent. Identity: `{source}:{filename}` (e.g., `project:go.md`).
- **Session State**: A JSON file at `/tmp/zk-session-{SESSION_ID}.json` tracking which rule identifiers have been injected.
- **Hook Event**: A JSON payload received via stdin, with shape varying by event type (see Interface Contract above).

## Success Criteria

### Measurable Outcomes

- **SC-001**: Rules are injected at read time — agent has file-type guidance before its first write to that file type.
- **SC-002**: No duplicate rule injection within a session (until compaction or resume resets tracking).
- **SC-003**: Rules re-inject correctly after compaction and resume events.
- **SC-004**: Hook response time < 100ms p99 measured over 100 invocations with warm filesystem cache.
- **SC-005**: Same rules files produce correct output for both Claude Code and Gemini CLI sessions.
- **SC-006**: MultiEdit operations correctly inject rules for all file types touched.

## Testing Requirements

### Test Strategy

Integration-first approach. The `brains hook` subcommand is the primary integration boundary — test it by feeding JSON stdin and checking stdout output + session state file mutations.

- **Integration tests**: Feed realistic hook event JSON to `brains hook`, verify output format and session state tracking.
- **Unit tests**: Glob pattern matching logic, frontmatter parsing, agent detection, file path extraction from each tool type.
- **E2E test (manual)**: Register hooks in a Claude Code session, read/write files, verify rules appear in context.

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Pipe Read PostToolUse JSON to `brains hook --event post-tool-use`, verify rules in stdout |
| FR-002 | Integration | Pipe Write PostToolUse JSON, verify rules in stdout (first time) and empty stdout (second time) |
| FR-003 | Integration | Pipe two Read events for same file type, verify state file tracks injected rules, second returns empty |
| FR-004 | Integration | Pipe SessionStart compact JSON, verify state file's injected_rules is cleared |
| FR-005 | Unit | Test glob matching: `**/*.go` matches `src/main.go`, doesn't match `main.py` |
| FR-006 | Integration | Pipe SessionStart startup JSON, verify unconditional rules in stdout |
| FR-007 | Integration | Create rules in both `.brains/rules/` and `~/.brains/rules/` with same filename, verify both are injected |
| FR-008 | Unit | Set `CLAUDE_SESSION_ID`, verify `<system-reminder>` wrapping. Set `GEMINI_SESSION_ID`, verify plain markdown |
| FR-009 | Unit | Parse frontmatter: string paths, array paths, brace expansion, missing paths |
| FR-010 | Integration | Benchmark 100 invocations of `brains hook`, verify p99 < 100ms |
| FR-011 | Integration | Delete state file between invocations, verify fresh state created |
| FR-012 | Integration | Pipe SessionEnd JSON, verify state file is deleted |
| FR-013 | Integration | Pipe MultiEdit JSON with two file types, verify both rule sets in stdout |
| FR-014 | Integration | Run `brains hook --event session-start` as CLI command, verify it works end-to-end |

### Edge Case Coverage

- Multiple rules matching same file → all injected (those not yet seen)
- Corrupted session state JSON → fresh state created, all matching rules injected
- Hook binary crashes → agent continues (non-blocking exit code)
- Rules file changed mid-session → new content on next fresh injection (post-reset)
- Empty rules file (only frontmatter, no body) → nothing injected
- File path with spaces/special characters → glob matching handles correctly
- Parallel file reads (race condition) → accept occasional double-injection over missed injection
- `.brains/rules/` directory doesn't exist → skip silently, no output
- Both CLAUDE_SESSION_ID and GEMINI_SESSION_ID set → Claude takes precedence
