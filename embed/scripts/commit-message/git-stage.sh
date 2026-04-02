#!/usr/bin/env bash
# git-stage.sh — Stage specific files for commit
# Usage: git-stage.sh <file1> [file2] ...
# Safety: rejects flags, validates paths exist
set -euo pipefail

if [[ $# -eq 0 ]]; then
  echo "Usage: git-stage.sh <file1> [file2] ..." >&2
  exit 1
fi

for file in "$@"; do
  if [[ "$file" == -* ]]; then
    echo "Error: flags not allowed: $file" >&2
    exit 1
  fi
  if [[ ! -e "$file" ]]; then
    echo "Error: path not found: $file" >&2
    exit 1
  fi
done

git add -- "$@"
echo "Staged $# file(s):"
git status --short
