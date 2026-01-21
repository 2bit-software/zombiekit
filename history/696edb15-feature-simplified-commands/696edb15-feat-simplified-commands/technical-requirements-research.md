# Technical Requirements Research: Simplified Command Structure

**Created**: 2026-01-19
**Related Spec**: spec.md

## Overview

This document captures technical implementation details extracted from the business specification and codebase research. These are implementation hints to guide the planning phase.

## Architecture Decisions

### 1. Unified MCP Tool Approach

**Current State:**
- Separate tools: `initiative` (create/status/complete/list) and `step` (execute steps)
- Each tool registered independently in `internal/mcp/server.go`
- Skills in Claude Code invoke tools directly

**Proposed Approach:**
Create a unified `workflow` tool that consolidates actions:

```go
// New unified tool schema
workflow(
    action: "new|step|next|complete|help|list",
    dir: string,
    type?: "feature|bug|refactor|profile",  // for new
    name?: string,                           // for new
    description?: string,                    // for new (intent detection)
    step?: string,                           // for step action
    alt?: string,                            // for next action (alternative)
)
```

**Rationale:**
- Single registration point
- Cleaner API surface
- Easier to maintain workflow state
- Consistent error handling

### 2. Intent Detection Implementation

**Where it runs:** In the skill profile (Claude Code side), NOT in MCP tool

**Why:**
- Intent classification requires LLM inference
- MCP tools should remain pure/deterministic
- Claude Code already has LLM capabilities
- Keeps MCP server fast and simple

**Implementation Pattern:**
```
User: /brains.new "add rate limiting"
  ↓
Skill profile prompts Claude to classify:
  - Analyze: "add rate limiting" → likely "feature"
  - Confidence: high (>0.8)
  - If low confidence: prompt user for selection
  ↓
Call MCP: workflow(action="new", type="feature", name="rate-limiting", ...)
```

**Classification Keywords (for prompt):**
| Type | Indicators |
|------|------------|
| feature | add, new, implement, create, build, enable |
| bug | fix, broken, doesn't work, error, crash, wrong |
| refactor | refactor, restructure, clean up, reorganize, extract |
| profile | profile, agent, persona, template |

### 3. Workflow Registry Design

**Option A: Dynamic MCP Endpoint (Recommended)**
- New `workflow-registry` tool returns available workflows/steps
- Fresh fetch on every command (no caching)
- Enables future extensibility

**Option B: Embedded Static Registry**
- Hardcoded in step loader
- Faster but less flexible
- Use as fallback for Option A

**Proposed Schema:**
```json
{
  "workflows": [
    {
      "type": "feature",
      "steps": ["feature", "plan", "tasks", "eat", "audit", "clarify"],
      "transitions": {
        "feature": {"next": "plan", "alternatives": ["audit"]},
        "plan": {"next": "tasks", "alternatives": ["audit"]},
        "tasks": {"next": "eat", "alternatives": []},
        "eat": {"next": null, "alternatives": []}
      }
    }
  ]
}
```

### 4. Step Navigation Logic

**Current:** `step/service.go` checks prerequisites and executes

**Enhancement Needed:**
- Track step history in initiative state
- Allow backward navigation without prerequisite re-check
- Preserve artifacts on backward navigation

**State Enhancement:**
```go
type InitiativeState struct {
    Initiative    string
    Cycle         string
    CurrentStep   string
    StepHistory   []string      // NEW: ordered steps visited
    LastActivity  time.Time
    SubTasks      []SubTaskRef  // NEW: for sub-task support
}
```

### 5. Sub-task Implementation

**New Data Model:**
```go
type SubTaskRef struct {
    ID       string
    Type     string  // bug, feature, refactor
    Name     string
    Status   string  // active, completed
    CyclePath string
}
```

**Folder Structure:**
```
history/
└── 696edb15-feature-rate-limiting/
    ├── INITIATIVE.md
    ├── 696edb15-feat-rate-limiting/   # main cycle
    │   ├── spec.md
    │   └── plan.md
    └── subtasks/
        └── abc123-fix-null-pointer/   # bug sub-task
            ├── spec.md
            └── ...
```

### 6. Skill Profile Structure

**New Skills to Create:**

| Skill | File | Invokes |
|-------|------|---------|
| brains.new | .claude/skills/brains-new.md | workflow(action="new") |
| brains.step | .claude/skills/brains-step.md | workflow(action="step") |
| brains.next | .claude/skills/brains-next.md | workflow(action="next") |
| brains.complete | .claude/skills/brains-complete.md | workflow(action="complete") |
| brains.help | .claude/skills/brains-help.md | workflow(action="help") |

**Note:** These replace existing skills:
- brains.feature, brains.bug, brains.refactor → brains.new
- brains.status → brains.help

### 7. Backwards Compatibility

**Migration Strategy:**
1. Keep old skills working (deprecated warning)
2. Old skills internally call new workflow tool
3. Document migration path
4. Remove old skills after 2 release cycles

**Example Deprecation (brains.feature skill):**
```markdown
⚠️ DEPRECATED: Use `/brains.new feature` instead.
This command will be removed in a future version.

[Internally calls: workflow(action="new", type="feature")]
```

## Files to Modify

### MCP Tools
- `internal/mcp/tools/workflow/tool.go` (NEW)
- `internal/mcp/tools/workflow/types.go` (NEW)
- `internal/mcp/tools/workflow/tool_test.go` (NEW)
- `internal/mcp/server.go` (register new tool)

### Initiative Service
- `internal/initiative/types.go` (add StepHistory, SubTasks)
- `internal/initiative/service.go` (add sub-task support)

### Step Service
- `internal/step/service.go` (backward navigation)
- `internal/step/registry.go` (NEW - workflow registry)

### Skills (Claude Code side)
- `.claude/skills/brains-new.md` (NEW)
- `.claude/skills/brains-step.md` (NEW)
- `.claude/skills/brains-next.md` (NEW)
- `.claude/skills/brains-complete.md` (NEW)
- `.claude/skills/brains-help.md` (NEW)

### Config
- `internal/config/tools.go` (add workflow to known tools)

## API Design

### New Workflow Tool Actions

**Action: new**
```
Input: {action: "new", dir, type?, name?, description?}
Output: {
  initiative_id, cycle_path, detected_type?, confidence?,
  needs_confirmation: bool, workflow_options?: []
}
```

**Action: step**
```
Input: {action: "step", dir, step}
Output: {
  directive, files_to_read, composed_prompt,
  current_position, valid_steps: []
}
```

**Action: next**
```
Input: {action: "next", dir, alt?}
Output: {
  next_step, directive, files_to_read, composed_prompt,
  alternatives?: []
}
```

**Action: complete**
```
Input: {action: "complete", dir}
Output: {
  initiative_id, completed_at, incomplete_tasks?: [],
  needs_confirmation: bool
}
```

**Action: help**
```
Input: {action: "help", dir}
Output: {
  commands: [{name, description}],
  current_state?: {initiative, step, progress},
  available_actions?: []
}
```

## Performance Considerations

- Registry fetch should be fast (<100ms) or have fallback
- Step service already caches step definitions
- Consider caching registry for 1 second to handle rapid commands

## Security Considerations

- Validate all step names against registry (prevent injection)
- Sanitize user descriptions before processing
- MCP tool inputs already validated by mcp-go library

## Testing Approach

1. **Unit Tests:** Intent classifier (keywords → type mapping)
2. **Integration Tests:** Workflow tool actions
3. **E2E Tests:** Full command flows via skills

## Open Technical Questions

1. Should registry be persisted to disk or always fetched from MCP?
   - **Recommendation:** Always fetch, with embedded fallback

2. How to handle concurrent sub-task operations?
   - **Recommendation:** Only one sub-task active at a time

3. Should old skills be removed immediately or deprecated first?
   - **Recommendation:** Deprecate first with 2-release grace period
