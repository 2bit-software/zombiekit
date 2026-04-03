# Branch Check for /brains.new

## What

Add a "Pre-Classification: Branch Check" step to the `new` command (`embed/commands/new.md`) that detects when the user is on a non-main branch and prompts them to switch before starting new work. This prevents accidentally stacking PRs on top of each other.

## Constraints

- Insert between "Active Initiative Check" and "Classification Task" sections
- Must use `mcp__zombiekit__git` for branch detection
- Must use `AskUserQuestion` for the prompt
- Standard base branches: main, master, develop
- Must support custom branch input for non-standard base branches

## Acceptance Criteria

- [ ] When on main/master/develop, the step is silently skipped
- [ ] When on a feature branch, user is prompted with options: switch to main, switch to develop, type a custom branch, or stay on current branch
- [ ] Selecting a base branch triggers `git checkout <branch> && git pull`
- [ ] "Stay on current branch" skips the check and proceeds as before
- [ ] Custom branch input is supported for non-standard base branches

## Risks

- The git MCP tool may not support checkout directly — may need Bash fallback
