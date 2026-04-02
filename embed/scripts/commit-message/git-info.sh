#!/usr/bin/env bash
# git-info.sh — Gather git context for commit message generation
# No arguments. Outputs branch, status, recent commits, and diff.
set -euo pipefail

echo "=== Branch ==="
git branch --show-current

echo ""
echo "=== Status ==="
git status --short

echo ""
echo "=== Recent Commits ==="
git log --oneline -10

echo ""
echo "=== Diff (staged + unstaged) ==="
git diff HEAD
