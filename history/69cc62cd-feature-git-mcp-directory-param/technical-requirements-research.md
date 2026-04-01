# Technical Requirements Research: Git MCP Directory Parameter

## Technical Preferences (from ticket and codebase analysis)

### Parameter naming
- Other tools in the project use `working_directory` (profile, workflow) or `dir` (initiative).
- The ticket says "directory" — use `directory` to match the ticket wording and keep it concise.

### Implementation approach
- The git.Runner currently stores a fixed `workDir` set at construction time.
- Options:
  1. **Create a new Runner per call** when `directory` is provided. Cheap — Runner is a thin struct.
  2. **Add a method to Runner** that returns a copy with a different workDir. Slightly cleaner.
  3. **Pass directory override through the call chain.** More invasive.
- Recommendation: Option 1. `NewRunner(dir)` is trivial; creating one per-call for the override case keeps the existing runner untouched for the default case.

### Validation approach
- `validateFiles()` in validation.go takes `workDir string` as first arg — already supports arbitrary directories.
- Directory existence: check with `os.Stat()` before creating a runner.
- Git repo check: run `git rev-parse --git-dir` in the directory.

### Schema changes
- Add `mcp.WithString("directory", mcp.Description("Working directory for git operations. If omitted, uses the server default."))` to the tool registration in server.go.
- Do NOT add `mcp.Required()`.

### Error handling
- Return ToolError with appropriate codes:
  - `INVALID_DIRECTORY` — directory does not exist
  - `NOT_GIT_REPOSITORY` — directory exists but isn't a git repo
