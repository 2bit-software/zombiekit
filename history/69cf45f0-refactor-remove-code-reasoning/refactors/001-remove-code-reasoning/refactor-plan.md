# Refactor Plan

## Step 1: Remove package

Delete `internal/mcp/tools/codereasoning/` entirely.

## Step 2: Clean up server.go

Remove all references in `internal/mcp/server.go`:
- Import
- Struct fields (`codeReasoning`, `sessionManager`)
- Instantiation in `NewServer`
- Registration block in `registerTools`
- `handleCodeReasoning` method
- Update package doc comment

## Step 3: Clean up config/tools.go

- Remove `"code-reasoning"` from `KnownTools`
- Remove doc comment example for `"code-reasoning"`

## Step 4: Clean up server_test.go

- Remove `TestServer_CodeReasoning_Execute` test

## Step 5: Check config tests

- Verify/fix any tests in `internal/config/` that reference `"code-reasoning"`

## Step 6: Verify

- `go build ./...`
- `go test ./internal/mcp/... ./internal/config/...`
- `go vet ./...`

## Rollback

`git revert` the commit. No schema migrations or external state involved.
