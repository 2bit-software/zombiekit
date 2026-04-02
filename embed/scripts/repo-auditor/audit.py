#!/usr/bin/env python3
"""GitHub repository auditor — fetches issues, PRs, and conversations via gh CLI.

Two subcommands:
  overview <owner/repo>              Fetch repo metadata, issues, and PRs
  deep-dive <owner/repo> <n> [n...]  Fetch full conversations for specific items
"""

from __future__ import annotations

import argparse
import html
import json
import re
import subprocess
import sys
from typing import Dict, List, Optional


REPO_SLUG_RE = re.compile(r"^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$")
HTML_TAG_RE = re.compile(r"<[^>]+>")
MAX_COMMENT_LENGTH = 2000
MAX_DEEP_DIVE_ITEMS = 30
MAX_COMMENTS_PER_ITEM = 20
MAX_REVIEWS_PER_ITEM = 15


def validate_repo(slug: str) -> str:
    if not REPO_SLUG_RE.match(slug):
        print(f"Invalid repo slug: {slug!r}", file=sys.stderr)
        sys.exit(1)
    return slug


def run_gh(args: list[str]) -> dict | list | None:
    """Run a gh CLI command and return parsed JSON, or None on failure."""
    cmd = ["gh"] + args
    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=30)
    except subprocess.TimeoutExpired:
        print(f"Timeout: {' '.join(cmd)}", file=sys.stderr)
        return None

    if result.returncode != 0:
        print(f"Failed: {' '.join(cmd)}\n{result.stderr.strip()}", file=sys.stderr)
        return None

    try:
        return json.loads(result.stdout)
    except json.JSONDecodeError:
        print(f"Bad JSON from: {' '.join(cmd)}", file=sys.stderr)
        return None


def strip_html(text: str) -> str:
    if not text:
        return text
    text = html.unescape(text)
    return HTML_TAG_RE.sub("", text)


def truncate(text: str, limit: int = MAX_COMMENT_LENGTH) -> str:
    if not text or len(text) <= limit:
        return text or ""
    return text[:limit] + " [truncated]"


def comment_count(item: dict) -> int:
    """Get comment count from an item — handles both list and int formats."""
    c = item.get("comments", 0)
    if isinstance(c, list):
        return len(c)
    return c


def dedup_by_number(items_lists):
    """Merge multiple lists, deduplicate by 'number', keep richest version."""
    seen = {}
    for items in items_lists:
        if not items:
            continue
        for item in items:
            num = item.get("number")
            if num is None:
                continue
            if num not in seen:
                seen[num] = item
    return sorted(seen.values(), key=comment_count, reverse=True)


# -- overview subcommand --

ISSUE_JSON_FIELDS = "number,title,state,comments,createdAt,updatedAt,labels,author"

OVERVIEW_QUERIES = [
    # (type, gh-subcommand, extra-args)
    # gh issue/pr list uses --search for sorting via GitHub search qualifiers
    ("issues", "issue", ["list", "--state", "open", "--search", "sort:comments-desc", "--limit", "30"]),
    ("issues", "issue", ["list", "--state", "closed", "--search", "sort:comments-desc", "--limit", "20"]),
    ("issues", "issue", ["list", "--state", "open", "--search", "sort:created-desc", "--limit", "15"]),
    ("issues", "issue", ["list", "--state", "closed", "--search", "sort:updated-desc", "--limit", "15"]),
    ("prs", "pr", ["list", "--state", "open", "--search", "sort:comments-desc", "--limit", "15"]),
    ("prs", "pr", ["list", "--state", "merged", "--search", "sort:created-desc", "--limit", "20"]),
    ("prs", "pr", ["list", "--state", "open", "--search", "sort:created-desc", "--limit", "10"]),
]


def cmd_overview(repo: str) -> None:
    repo = validate_repo(repo)

    # 1. Repo metadata
    repo_data = run_gh([
        "repo", "view", repo, "--json",
        "name,description,stargazerCount,forkCount,createdAt,updatedAt,issues,pullRequests",
    ])

    # 2. Issues and PRs (sequential to avoid rate limits)
    raw: dict[str, list[list | None]] = {"issues": [], "prs": []}
    for bucket, subcmd, args in OVERVIEW_QUERIES:
        result = run_gh([subcmd] + args + ["-R", repo, "--json", ISSUE_JSON_FIELDS])
        raw[bucket].append(result)

    issues = dedup_by_number(raw["issues"])
    prs = dedup_by_number(raw["prs"])

    open_issues = sum(1 for i in issues if i.get("state", "").upper() == "OPEN")
    open_prs = sum(1 for p in prs if p.get("state", "").upper() == "OPEN")

    output = {
        "repo": repo_data,
        "issues": issues,
        "prs": prs,
        "summary": {
            "total_issues": len(issues),
            "total_prs": len(prs),
            "open_issues": open_issues,
            "closed_issues": len(issues) - open_issues,
            "open_prs": open_prs,
            "merged_prs": len(prs) - open_prs,
        },
    }

    json.dump(output, sys.stdout, indent=2)
    print()


# -- deep-dive subcommand --

ISSUE_VIEW_FIELDS = "number,title,body,comments,author,createdAt,state,labels"
PR_VIEW_FIELDS = "number,title,body,comments,author,createdAt,state,labels,reviews"


def fetch_item(repo: str, number: int) -> dict | None:
    """Try issue first, fall back to PR."""
    data = run_gh([
        "issue", "view", "-R", repo, str(number), "--json", ISSUE_VIEW_FIELDS,
    ])
    if data is not None:
        data["type"] = "issue"
        return data

    data = run_gh([
        "pr", "view", "-R", repo, str(number), "--json", PR_VIEW_FIELDS,
    ])
    if data is not None:
        data["type"] = "pr"
        return data

    print(f"Could not fetch #{number} as issue or PR", file=sys.stderr)
    return None


def cap_list(items: list, limit: int) -> tuple[list, int]:
    """Keep first 3 + last (limit-3) items if over limit. Returns (capped, omitted)."""
    if len(items) <= limit:
        return items, 0
    head = 3
    tail = limit - head
    omitted = len(items) - limit
    return items[:head] + items[-tail:], omitted


def clean_item(item: dict) -> dict:
    """Strip HTML, truncate long text, and cap comment/review lists."""
    if "body" in item:
        item["body"] = truncate(strip_html(item.get("body", "")))

    comments = item.get("comments", [])
    if isinstance(comments, list):
        comments, omitted = cap_list(comments, MAX_COMMENTS_PER_ITEM)
        for comment in comments:
            if isinstance(comment, dict) and "body" in comment:
                comment["body"] = truncate(strip_html(comment["body"]))
        item["comments"] = comments
        if omitted:
            item["comments_omitted"] = omitted

    reviews = item.get("reviews", [])
    if isinstance(reviews, list):
        reviews, omitted = cap_list(reviews, MAX_REVIEWS_PER_ITEM)
        for review in reviews:
            if isinstance(review, dict) and "body" in review:
                review["body"] = truncate(strip_html(review["body"]))
        item["reviews"] = reviews
        if omitted:
            item["reviews_omitted"] = omitted

    return item


def cmd_deep_dive(repo: str, numbers: list[int]) -> None:
    repo = validate_repo(repo)

    if len(numbers) > MAX_DEEP_DIVE_ITEMS:
        print(
            f"Warning: capping deep-dive from {len(numbers)} to {MAX_DEEP_DIVE_ITEMS} items",
            file=sys.stderr,
        )
        numbers = numbers[:MAX_DEEP_DIVE_ITEMS]

    results = []
    for num in numbers:
        item = fetch_item(repo, num)
        if item is not None:
            results.append(clean_item(item))

    json.dump(results, sys.stdout, indent=2)
    print()


# -- CLI --

def main() -> None:
    parser = argparse.ArgumentParser(description="GitHub repository auditor")
    sub = parser.add_subparsers(dest="command", required=True)

    ov = sub.add_parser("overview", help="Fetch repo overview data")
    ov.add_argument("repo", help="owner/repo slug")

    dd = sub.add_parser("deep-dive", help="Fetch full conversations for items")
    dd.add_argument("repo", help="owner/repo slug")
    dd.add_argument("numbers", type=int, nargs="+", help="Issue/PR numbers")

    args = parser.parse_args()

    if args.command == "overview":
        cmd_overview(args.repo)
    elif args.command == "deep-dive":
        cmd_deep_dive(args.repo, args.numbers)


if __name__ == "__main__":
    main()
