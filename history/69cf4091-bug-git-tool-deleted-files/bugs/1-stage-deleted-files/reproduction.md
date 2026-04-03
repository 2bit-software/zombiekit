# Reproduction

## Prerequisites

- Git repository with at least one committed file

## Steps

1. Create and commit a file:
   ```
   echo "hello" > tracked.txt
   git add tracked.txt && git commit -m "add tracked.txt"
   ```
2. Delete the file: `rm tracked.txt`
3. Call the MCP git tool with `action: "stage", files: "tracked.txt"`
4. Error: `VALIDATION_ERROR: file does not exist: tracked.txt`

## Expected

The file should be staged (deletion staged), equivalent to `git add tracked.txt` or `git rm tracked.txt`.

## Failing Test Case

A test in `tool_test.go` that creates a tracked file, deletes it from disk, then attempts to stage it via the tool — should succeed but currently fails.
