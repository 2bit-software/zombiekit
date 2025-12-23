# Research: Profile Import Subcommand

**Feature**: 005-profile-import
**Date**: 2025-12-22

## Technical Context Resolution

No NEEDS CLARIFICATION items exist. The technical context is fully resolved from the existing codebase (004-source-interface).

## Key Research Findings

### 1. Existing Source Interface

**Decision**: Use existing `ProfileSourceInterface` from 004-source-interface for reading Claude agents.

**Rationale**:
- ClaudeSource already implements `FindProfileDirs()`, `LoadProfiles()`, and `LoadAllProfiles()`
- These methods handle local (`.claude/agents/`) and global (`~/.claude/agents/`) directories
- Profile struct already contains parsed frontmatter including Claude-specific fields (Model, Color)

**Alternatives considered**:
- Direct file reading: Rejected - duplicates existing ClaudeSource logic
- New interface: Rejected - existing interface sufficient for read operations

### 2. Writing Brains Profiles

**Decision**: Write directly to brains profile directories using standard Go file I/O.

**Rationale**:
- BrainsSource.CreateProfile() checks for existing profiles and errors on collision
- Import requires overwrite behavior, so we bypass CreateProfile
- Direct os.WriteFile() allows overwrite semantics

**Alternatives considered**:
- Modify BrainsSource.CreateProfile() to accept --force flag: Rejected - violates existing interface contract
- Add new BrainsSource.ImportProfile() method: Considered but adds complexity for single use case

### 3. Frontmatter Conversion

**Decision**: Convert Claude frontmatter to Brains frontmatter with these mappings:

| Claude Field | Brains Field | Handling |
|--------------|--------------|----------|
| name | name | Copy as-is |
| description | description | Copy as-is |
| includes | includes | Copy as-is |
| model | - | Discard (no equivalent) |
| color | - | Discard (no equivalent) |
| inherits | inherits | Always set to `false` |

**Rationale**:
- User explicitly stated Claude agents have no inherits concept, default to false
- Model and color are Claude-specific display fields with no brains equivalent
- Name, description, includes are semantically identical between formats

### 4. Directory Creation

**Decision**: Create target `.brains/profiles/` directories if they don't exist.

**Rationale**:
- Spec FR-010 requires this behavior
- Standard os.MkdirAll() handles nested creation safely
- Matches user expectation for import command (minimal setup required)

### 5. Error Handling Strategy

**Decision**: Partial failure model - continue importing valid agents, report failures at end.

**Rationale**:
- Spec FR-011 requires this behavior
- Single agent parse failure shouldn't block other imports
- Summary report shows both successes and failures

### 6. Dry Run Implementation

**Decision**: Collect all planned operations without executing file writes.

**Rationale**:
- Dry run should exactly mirror real import logic
- Return same ImportResult structure with operations marked as "planned"
- No state changes during dry run

## Best Practices Applied

### Go File Writing
- Use os.WriteFile() with 0o644 permissions (match BrainsSource.CreateProfile)
- Create parent directories with os.MkdirAll(dir, 0o755)
- Handle path separators with filepath.Join (cross-platform)

### CLI Integration
- Add import subcommand to existing profile command group
- Support --dry-run flag (boolean)
- Support --format json flag (consistent with other subcommands)
- Source argument is positional (required): `brains profiles import claude`

### Testing Strategy
- Unit tests for frontmatter conversion (pure function)
- Integration tests with temp directories for file I/O
- Test edge cases: empty directories, parse errors, permission errors
