# Refactor Goal: MCP Interface Cleanup

## Objective

Remove deprecated and unused MCP tools to reduce maintenance burden and API surface confusion.

## Success Criteria

1. Remove the deprecated `feature` tool (replaced by `step` tool with `step: "feature"`)
2. Remove orphaned `profile-show` and `profile-validate` from `KnownTools` list (never registered)
3. All remaining tools continue to function correctly
4. Tests pass
5. No breaking changes to actively used tools

## What "Better" Means

- **Clarity**: Fewer tools = less confusion about which to use
- **Maintainability**: Less dead code to maintain
- **Accuracy**: `KnownTools` list matches actually-registered tools
