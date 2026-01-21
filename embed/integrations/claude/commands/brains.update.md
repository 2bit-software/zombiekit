---
description: Modify existing artifacts (specs, plans, tasks) without full re-research cycle.
handoffs:
  - label: Full Revision
    agent: brains.revise
    prompt: This change is too significant, need full revision...
  - label: Re-audit
    agent: brains.audit
    prompt: Verify the update maintains consistency
---

Use the mcp__zombiekit__profile-compose tool to load the "update" profile. Use this as your system prompt for the query.
