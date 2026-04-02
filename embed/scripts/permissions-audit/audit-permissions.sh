#!/usr/bin/env bash
set -euo pipefail

PROJECT_DIR="${1:-$PWD}"
LOCAL_FILE="$PROJECT_DIR/.claude/settings.local.json"
GLOBAL_FILE="$HOME/.claude/settings.json"

# Exit 1 if local file missing
if [[ ! -f "$LOCAL_FILE" ]]; then
  echo "Error: No local settings file found at $LOCAL_FILE" >&2
  exit 1
fi

# Ensure jq is available
if ! command -v jq &>/dev/null; then
  echo "Error: jq is required but not installed" >&2
  exit 2
fi

# Extract allow arrays (empty array if missing)
LOCAL_PERMS=$(jq -c '.permissions.allow // []' "$LOCAL_FILE" 2>/dev/null) || {
  echo "Error: Failed to parse $LOCAL_FILE" >&2
  exit 2
}

if [[ -f "$GLOBAL_FILE" ]]; then
  GLOBAL_PERMS=$(jq -c '.permissions.allow // []' "$GLOBAL_FILE" 2>/dev/null) || {
    GLOBAL_PERMS="[]"
  }
else
  GLOBAL_PERMS="[]"
fi

# Categorize permissions with jq
jq -n \
  --arg project "$PROJECT_DIR" \
  --argjson local "$LOCAL_PERMS" \
  --argjson global "$GLOBAL_PERMS" \
'
# Globalize patterns: read-only bash, task runner, common tools
def should_globalize:
  # Read-only bash commands (wildcard patterns only, e.g. Bash(ls:*))
  (test("^Bash\\((ls|cat|grep|find|wc|head|tail|tree|sort|file|which|diff|pwd):"))
  or
  # Task runner (both wildcard and space forms per convention)
  test("^Bash\\(task[: ]")
  or
  # Git commands (wildcard patterns only, e.g. Bash(git add:*))
  test("^Bash\\(git (add|fetch|status|show|log|diff|rev-parse|branch):")
  or
  # WebSearch
  . == "WebSearch"
  or
  # Common WebFetch domains
  (test("^WebFetch\\(domain:github\\.com\\)$") or
   test("^WebFetch\\(domain:raw\\.githubusercontent\\.com\\)$") or
   test("^WebFetch\\(domain:stackoverflow\\.com\\)$"))
  or
  # Common MCP tools
  (test("^mcp__zombiekit__code-reasoning$") or test("^mcp__zombiekit__stickymemory$") or
   test("^mcp__mcp-genie__code-reasoning$") or test("^mcp__mcp-genie__stickymemory$"))
  or
  # Common skills
  (test("^Skill\\(remember\\)$") or test("^Skill\\(commit-message\\)$") or
   test("^Skill\\(research\\)$") or test("^Skill\\(deep-research\\)$"));

def globalize_reason:
  if test("^Bash\\((ls|cat|grep|find|wc|head|tail|tree|sort|file|which|diff|pwd):") then "Read-only filesystem command"
  elif test("^Bash\\(task[: ]") then "Task runner — universal CLI interface"
  elif test("^Bash\\(git (add|fetch|status|show|log|diff|rev-parse|branch):") then "Common git operation"
  elif . == "WebSearch" then "Universal search tool"
  elif test("^WebFetch\\(domain:github\\.com\\)$") then "GitHub — universal dev resource"
  elif test("^WebFetch\\(domain:raw\\.githubusercontent\\.com\\)$") then "GitHub raw content — universal dev resource"
  elif test("^WebFetch\\(domain:stackoverflow\\.com\\)$") then "Stack Overflow — universal dev resource"
  elif test("^mcp__(zombiekit|mcp-genie)__") then "Common MCP tool"
  elif test("^Skill\\(") then "Common skill"
  else "Universal tool"
  end;

# Project-specific patterns
def is_project_specific:
  # Language-specific tools (both Bash(go :*) and Bash(go:*) forms, including env-prefixed)
  (test("^Bash\\([A-Z_]+=\\S+ (go|mix|zig|haxe|npm|npx|bun|cargo|node|python|uv)[: ]") or
   test("^Bash\\((go|mix|zig|haxe|npm|npx|bun|cargo|node|python|uv)[: ]"))
  or
  # Project-relative paths
  test("^Bash\\(\\./")
  or
  # Docker compose commands
  (test("^Bash\\(docker compose[: ]") or test("^Bash\\(docker exec[: ]") or
   test("^Bash\\(docker logs[: ]"))
  or
  # Project-specific MCP
  (test("^mcp__playwright__") or test("^mcp__linear") or test("^mcp__claude_ai_Linear__"))
  or
  # Language-specific doc domains
  (test("^WebFetch\\(domain:hex\\.pm\\)$") or test("^WebFetch\\(domain:hexdocs\\.pm\\)$") or
   test("^WebFetch\\(domain:pkg\\.go\\.dev\\)$") or test("^WebFetch\\(domain:docs\\.rs\\)$") or
   test("^WebFetch\\(domain:npmjs\\.com\\)$") or test("^WebFetch\\(domain:pypi\\.org\\)$") or
   test("^WebFetch\\(domain:crates\\.io\\)$"));

def project_specific_reason:
  if test("^Bash\\(([A-Z_]+=\\S+ )?(go|mix|zig|haxe|npm|npx|bun|cargo|node|python|uv)[: ]") then
    (capture("(?<tool>go|mix|zig|haxe|npm|npx|bun|cargo|node|python|uv)") | "Language tool: \(.tool)")
  elif test("^Bash\\(\\./") then "Project-relative path"
  elif test("^Bash\\(docker (compose|exec|logs)[: ]") then "Docker Compose — project-dependent"
  elif test("^mcp__playwright__") then "Playwright MCP — project-specific testing"
  elif test("^mcp__(linear|claude_ai_Linear)") then "Linear MCP — project-specific integration"
  elif test("^WebFetch\\(domain:") then "Language-specific documentation site"
  else "Project-specific tool"
  end;

# Build the global set for fast lookup
($global | map({key: ., value: true}) | from_entries) as $global_set |

# Categorize each local permission
reduce ($local | unique[]) as $perm (
  {already_global: [], globalize: [], project_specific: [], ambiguous: []};

  if $global_set[$perm] then
    .already_global += [$perm]
  elif ($perm | should_globalize) then
    .globalize += [{permission: $perm, reason: ($perm | globalize_reason)}]
  elif ($perm | is_project_specific) then
    .project_specific += [{permission: $perm, reason: ($perm | project_specific_reason)}]
  else
    .ambiguous += [{permission: $perm, reason: "Could not auto-categorize — needs your input"}]
  end
) |

# Add project path
.project = $project
' 2>/dev/null || {
  echo "Error: jq categorization failed" >&2
  exit 2
}
