# Reproduction

## Steps

1. Be on `main` branch
2. Run `/brains.new add a button` (or any feature input)
3. Observe: initiative MCP tool creates branch `{id}-feature-{slug}` and checks it out
4. Observe: agent reaches step 3 "Create Branch" and runs `git checkout -b feat/{slug}/...`
5. Error: `fatal: a branch named '...' already exists`

## Expected

Branch creation should happen exactly once — either in the initiative tool OR in the workflow step, not both.

## Actual

Branch is created by the initiative tool, then the workflow step tries to create it again.
