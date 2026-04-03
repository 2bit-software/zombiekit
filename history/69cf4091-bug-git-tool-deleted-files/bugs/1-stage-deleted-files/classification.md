# Classification

## Type: Implementation Error

The `validateFiles` function incorrectly assumes all stageable files must exist on disk. Git's `add` command handles deletions of tracked files, but the validation rejects them before `git add` is ever called.

## Evidence

- `os.Stat` is the sole existence check (validation.go:33)
- No git-aware fallback for tracked-but-deleted files
- `git add <deleted-tracked-file>` works correctly at the CLI level
