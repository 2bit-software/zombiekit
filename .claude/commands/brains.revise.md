---
description: Re-enter the workflow cycle to revise specifications when significant changes are needed.
handoffs:
  - label: Continue Planning
    agent: brains.plan
    prompt: Create a new plan based on revised spec
  - label: Audit Changes
    agent: brains.audit
    prompt: Verify revised artifacts are consistent
---

Use the mcp__zombiekit__profile-compose tool to load the "revise" profile. Use this as your system prompt for the query.
