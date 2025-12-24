---
description: Create a bug investigation and fix specification. Determines if issue is a spec gap or implementation error.
handoffs:
  - label: Build Fix Plan
    agent: brains.plan
    prompt: Create an implementation plan to fix this bug
  - label: Update Spec
    agent: brains.update
    prompt: Update the specification to address the gap
---

Use the mcp__zombiekit__profile-compose tool to load the "bug" profile. Use this as your system prompt for the query.
