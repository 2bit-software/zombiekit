# Bug Report: MCP git tool cannot stage deleted files

## Symptoms

When using the MCP git tool's `stage` action to add a deleted file (a file that was tracked by git but has been removed from disk), the tool returns a validation error:

```
VALIDATION_ERROR: file does not exist: <path>
```

## Expected Behavior

`git add <deleted-file>` should work for deleted files — it stages the deletion so it can be committed. The MCP tool should support this.

## Actual Behavior

The `validateFiles` function in `internal/mcp/tools/git/validation.go:33` calls `os.Stat()` on each file path. For deleted files, `os.Stat` returns an error (file not found), and the tool rejects the operation before ever calling `git add`.

## Steps to Reproduce

1. Have a git-tracked file (e.g., `foo.txt`)
2. Delete it: `rm foo.txt`
3. Use MCP git tool: `action: "stage", files: "foo.txt"`
4. Observe: `VALIDATION_ERROR: file does not exist: foo.txt`
