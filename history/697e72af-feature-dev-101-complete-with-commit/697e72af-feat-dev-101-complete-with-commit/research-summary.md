# Research Summary: Commit Offer Integration

## Current Architecture

### Workflow System

The `/brains.complete` command follows this flow:

1. **Claude command** (`.claude/commands/brains.complete.md`) invokes `workflow-compose` tool
2. **Workflow** (`embed/workflows/complete.md`) becomes the system prompt
3. **Workflow instructions** guide Claude to:
   - Check active initiative
   - Generate completion summary
   - Update INITIATIVE.md
   - Clear active state via `initiative` MCP tool

### Key Files

| File | Purpose |
|------|---------|
| `embed/workflows/complete.md` | Workflow definition (system prompt) |
| `embed/profiles/complete.md` | Profile with handoffs (unused by workflow currently) |
| `.claude/commands/brains.complete.md` | Command entry point |
| `embed/integrations/claude/commands/brains.complete.md` | Embedded copy of command |

### Git Integration

- **Location**: `internal/step/git.go`
- **Current capabilities**: Branch creation/switching only
- **No commit functionality exists**
- **Graceful degradation**: Returns nil for non-git environments

### Skill System

The Skill tool can invoke skills by name. Looking at CLAUDE.md, there's a `commit-message` skill listed:

```
- commit-message: Generates conventional commit messages...
```

This appears to be a built-in Claude Code skill, not a ZombieKit skill. ZombieKit uses profiles and workflows, not "skills" in the same sense.

## Integration Options

### Option A: Workflow-Only (Recommended)

Modify `embed/workflows/complete.md` to add commit offer instructions:

**Pros:**
- No code changes required
- Pure prompt engineering
- Easy to test and iterate

**Cons:**
- Relies on Claude following instructions correctly
- Git status check is implicit (Claude runs `git status`)

### Option B: MCP Tool Enhancement

Add `commit` action to the `initiative` MCP tool:

**Pros:**
- More control over git operations
- Can provide structured git status info

**Cons:**
- Requires Go code changes
- More complex testing
- Overkill for the use case

### Option C: New Commit Profile

Create a `commit-offer` profile that can be composed into complete workflow:

**Pros:**
- Reusable across other workflows
- Clean separation of concerns

**Cons:**
- Profile composition adds complexity
- May be overengineered

## Recommendation

**Option A (Workflow-Only)** is the right approach:

1. The workflow already guides Claude to run bash commands
2. Git status/add/commit are straightforward bash operations
3. The commit-message skill (or inline generation) handles message creation
4. No new MCP tools needed

## Implementation Notes

### Changes Required

1. **`embed/workflows/complete.md`**: Add step 4.5 "Offer Commit"
2. **`embed/profiles/complete.md`**: Add matching instructions (for consistency)

### Git Check Pattern

```bash
# Check for uncommitted changes
git status --porcelain
# If output is non-empty, there are changes
```

### Commit Flow

1. Check `git status --porcelain`
2. If changes exist, ask user via AskUserQuestion
3. If user accepts, either:
   - Use Skill tool with `commit-message` (if available)
   - Generate commit message inline following conventional commits
4. Run `git add -A && git commit -m "..."`
5. Proceed with completion regardless of outcome
