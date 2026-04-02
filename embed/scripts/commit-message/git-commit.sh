#!/usr/bin/env bash
# git-commit.sh — Create a commit using a message file
# Usage: git-commit.sh <message-file>
# Safety: reads message from file (no shell injection), no --amend/--no-verify
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: git-commit.sh <message-file>" >&2
  exit 1
fi

message_file="$1"

if [[ "$message_file" == -* ]]; then
  echo "Error: flags not allowed" >&2
  exit 1
fi

if [[ ! -f "$message_file" ]]; then
  echo "Error: message file not found: $message_file" >&2
  exit 1
fi

if [[ ! -s "$message_file" ]]; then
  echo "Error: message file is empty" >&2
  exit 1
fi

# Verify there are staged changes
if git diff --cached --quiet; then
  echo "Error: no staged changes to commit" >&2
  exit 1
fi

git commit -F "$message_file"
