---
name: automode
description: Autonomous execution mode — auto-advances through all workflow steps, completes initiative, creates PR, and pushes without human intervention.
type: modifier
inherits: false
---

# AutoMode

You are running in **AutoMode**. This means you execute the entire workflow autonomously without waiting for human intervention between steps.

## Core Rules

1. **Auto-advance**: After completing the current step's work, immediately call `/brains.next` to advance to the next step. Do NOT pause, summarize, or ask for confirmation.
2. **No human checkpoints**: Skip all `AskUserQuestion` prompts. Choose the default/affirmative option and continue.
3. **No handoff suggestions**: Do not present handoff options to the user. Continue the linear step sequence.
4. **Fail-forward**: If a non-critical step encounters issues, log the issue in the initiative's INITIATIVE.md and continue to the next step. Only stop if compilation fails or tests produce errors that block further work.

## At the Complete Step

When you reach the completion phase (all steps are done, or `/brains.complete` is invoked):

1. **Skip all confirmation prompts** — do not ask whether to commit, update Linear, etc.
2. **Commit all changes**: Stage both implementation files and the `history/{initiative}/` directory. Use the `/commit-message` skill to generate and execute the commit.
3. **Prepare PR and notify orchestrator**: If `WORK_CALLBACK_URL` is set, use the `/create-pr-automode` skill. This writes the PR description to `.ai/` and fires a callback so the orchestrator pushes the branch and creates the PR from the host. If `WORK_CALLBACK_URL` is NOT set, fall back to the `/create-pr` skill.
4. **Update Linear** (if a source ticket exists): Post a completion comment and mark the ticket as Done.
5. **Clear active state**: Proceed with normal initiative completion.

## Signal Propagation

When AutoMode is active, every call to `mcp__zombiekit__profile-compose` MUST include `"automode"` in the profiles array alongside the step's own profile. For example:

```json
{"profiles": ["implement", "automode"]}
```

This ensures AutoMode instructions are present at every step.
