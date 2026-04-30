# Goal: Data-Driven Workflows

## Improvement Goal

Separate workflow sequence definitions (data) from step execution instructions (behavior) so that:
- Workflows are pure data: an ordered list of steps, each mapping to one or more profiles
- Profiles are pure behavior: instructions for executing a single step, with no routing awareness
- Commands are orchestration: read workflow data, load profiles, manage state

## What "Better" Means

- **Single source of truth**: Step sequences defined in exactly one place (workflow files)
- **Composability**: Profiles can be reused across different workflows without modification
- **Extensibility**: Users create new workflow types by dropping a markdown file — no Go code changes
- **Correctness**: Step→profile mapping is preserved in INITIATIVE.md, eliminating broken resolution for non-matching step/profile names
- **Testability**: Each concern (data parsing, profile loading, state management) is independently testable

## Success Criteria

1. Workflow files contain the step sequence in YAML frontmatter
2. Profile files contain NO routing metadata (`steps:`, `handoffs:`)
3. INITIATIVE.md step table includes a Profile column that `/brains.next` can read
4. `GetWorkflowSteps()` reads from workflow files, not profile files
5. All existing tests pass
6. New tests cover: workflow parsing, step table with profile column, step→profile resolution
7. End-to-end behavior of `/brains.new` → `/brains.next` → `/brains.complete` is unchanged
