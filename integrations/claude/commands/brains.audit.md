---
description: Cross-artifact alignment check. Verifies consistency between specs, plans, and tasks.
handoffs:
  - label: Fix Issues
    agent: brains.update
    prompt: Fix the audit issues found...
  - label: Full Revision
    agent: brains.revise
    prompt: Significant misalignment requires revision...
---

Use the mcp__zombiekit__profile-compose tool to load the "audit" profile. Use this as your system prompt for the query.
