---
name: permissions-audit
description: Audits Claude Code tool permissions in the current project and helps migrate common ones to global settings.
type: skill
---

# Permissions Audit

You are an expert at managing Claude Code permissions across projects. You help users identify which tool permissions should be global vs project-specific, and safely migrate them.

## Workflow

### Phase 1 — Audit

Run the audit script to gather and categorize permissions:

```bash
~/.brains/scripts/permissions-audit/audit-permissions.sh "$PWD"
```

Parse the JSON output. If exit code is 1, the project has no local settings file — inform the user and stop.

### Phase 2 — Present Report

Display a clear report using this format:

```
## Permissions Audit: <project-name>

Local: N | Already global: N | Candidates: N

### Already Global (N)
These local permissions are redundant — they already exist in your global settings:
| Permission |
|---|
| `WebSearch` |

### Recommended for Global (N)
| Permission | Reason |
|---|---|
| `Bash(ls:*)` | Read-only filesystem command |

### Project-Specific (N)
| Permission | Reason |
|---|---|
| `Bash(mix compile:*)` | Elixir build tool |

### Needs Your Input (N)
| # | Permission | Reason |
|---|---|---|
| 1 | `Bash(curl:*)` | Could be global or project-specific |

Which ambiguous permissions should be globalized? (e.g., "1, 2" or "all" or "none")
```

If there are no ambiguous items, skip the question and ask: "Apply the recommended changes?"

If there are no recommendations AND no ambiguous items, just show the report and stop.

### Phase 3 — Apply Changes

After the user confirms which permissions to migrate:

1. Read `~/.claude/settings.json`
2. Add approved permissions to `permissions.allow` (create the key path if it doesn't exist), deduplicate, sort
3. Edit with the Edit tool
4. Read `.claude/settings.local.json`
5. Remove migrated permissions from `permissions.allow`
6. Edit with the Edit tool
7. Display summary of what changed

## Safety Rules

- **Never remove from local without first successfully adding to global**
- Only touch the `allow` array — never modify `deny` or other permission keys
- Preserve all other keys in both files (hooks, statusLine, etc.)
- Keep the local file even if its allow array becomes empty
- If the global file has no `permissions` key yet, create `{"permissions": {"allow": []}}` merged with existing content
