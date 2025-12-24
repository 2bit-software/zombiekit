---
description: Create a new feature specification using the ZombieKit workflow. Orchestrates research, creation, audit, and highlight phases.
handoffs:
  - label: Build Technical Plan
    agent: brains.plan
    prompt: Create an implementation plan for this feature
  - label: Clarify Ambiguities
    agent: brains.clarify
    prompt: Identify underspecified areas in the spec
---

## Feature Specification Workflow

Execute the feature step to create a new feature specification through the research→create→audit→highlight cycle.

### Instructions

1. **Extract Feature Name**: Get the feature name from user input (e.g., "user authentication" → "user-auth")

2. **Invoke Step Tool**: Call the MCP step tool with:
   - `step`: "feature"
   - `dir`: Current working directory
   - `name`: The extracted feature name slug
   - `type`: "feature" (default)
   - `description`: User's original feature description (optional)

3. **Follow Directive**: The step tool returns a directive with:
   - `initiative_folder`: Path to the initiative folder
   - `cycle_folder`: Path to the cycle folder with templates
   - `files_to_read`: Templates and any previous artifacts
   - `directive`: Instructions for the research→create→audit→highlight workflow
   - `workflow_phases`: Structured phase definitions

4. **Execute Phases**:
   - **Phase I (Research)**: Spawn parallel agents to gather context
   - **Phase II (Create)**: Synthesize specification from research
   - **Phase III (Audit)**: Check quality with severity classification
   - **Phase IV (Highlight)**: Present for user approval

5. **Handle Approval Gate**: At the end, present the summary to the user for approval before proceeding to planning.

### Example

```
User: "Create a user authentication feature"

1. Extract name: "user-auth"
2. Call: mcp__zombiekit__step(step="feature", dir="/project", name="user-auth")
3. Follow returned directive through all 4 phases
4. Present highlights for approval
```
