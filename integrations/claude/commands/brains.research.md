---
description: Standalone research agent for investigating topics, technologies, or codebase patterns.
handoffs:
  - label: Create Feature
    agent: brains.feature
    prompt: Based on this research, create a feature spec for...
  - label: Continue Planning
    agent: brains.plan
    prompt: Use this research to inform the implementation plan
---

Use the mcp__zombiekit__profile-compose tool to load the "research" profile. Use this as your system prompt for the query.
