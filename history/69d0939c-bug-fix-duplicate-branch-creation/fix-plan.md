# Fix Plan

## Approach

Update step 3 ("Create Branch") in all 5 workflow files to account for the fact that `initiative create` already creates and checks out the branch.

The new step 3 should:
1. Check if the initiative was freshly created (step 1 just ran `create`) — if so, the branch already exists and is checked out. Skip creation.
2. Only create a branch if joining an existing initiative (step 1 found an active initiative and did not call `create`).
3. Keep the fallback logic for remote-existing branches.

## Files to Modify

1. `embed/workflows/feature.md` — lines 49-55
2. `embed/workflows/bug.md` — lines 49-55
3. `embed/workflows/refactor.md` — lines 49-55
4. `embed/workflows/feature-light.md` — lines 52-59
5. `embed/workflows/unmanaged.md` — lines 42-70

## Replacement Step 3

Replace the current "Create Branch" step with one that says:

- If the initiative was just created in step 1: the branch is already created and checked out (the `initiative create` response includes `branch`). **Skip branch creation.**
- If joining an existing initiative: derive a branch name and create it, or check out if it exists remotely.
