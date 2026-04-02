---
name: github-research
description: Clones a GitHub repository and performs a comprehensive multi-phase audit including security analysis, business spec generation, completeness review, testing posture assessment, and code quality evaluation.
type: skill
---

# GitHub Research

Clones a GitHub repository and runs a structured, multi-phase audit to understand what the software does, whether it's safe, how complete it is, how well it's tested, and how well it's built.

## Input

A GitHub repository URL. Accepts any standard format:
- `https://github.com/owner/repo`
- `github.com/owner/repo`
- `git@github.com:owner/repo.git`
- `owner/repo` (shorthand — expand to full GitHub URL)

## Phase 1: Clone

1. Extract the repo name from the URL (the last path segment, minus `.git` if present).
2. Clone to `~/Projects/research/<repo-name>`. If the directory already exists, ask the user whether to re-clone or use the existing copy.
3. `cd` into the cloned directory. All subsequent work happens here.
4. Create the output directory: `.audit/`

## Phase 2: Parallel Audits

Fan out these four audits simultaneously using the Agent tool. Each subagent writes its report to `.audit/` inside the repo.

### 2a: Security Audit (subagent)

**Goal:** Determine what this software sends over the network, what sensitive data it reads, and whether it exhibits any malicious behavior.

Dispatch a subagent with the following instructions:

```
You are performing an in-depth security audit of the repository at <repo-path>.
Write your findings to <repo-path>/.audit/security-audit.md.

Investigate each of the following areas thoroughly. Read actual source code — do not
rely solely on grep pattern matches. Trace data flows from sensitive sources to sinks.

## 1. Network Activity

Find every place the code communicates over the network:
- HTTP clients (fetch, axios, requests, urllib, http.get, curl, HttpClient, etc.)
- WebSocket connections
- gRPC / protobuf / Thrift calls
- DNS lookups
- SMTP / email sending
- Any URL construction (string concatenation, template literals building URLs)

For each network call found:
- What URL/host does it contact?
- What data is sent in the request body, query params, or headers?
- Is user data or local system information included?
- Is the destination hardcoded or configurable?
- Is TLS/HTTPS enforced?

## 2. Secrets and Sensitive Data Access

Find every place the code reads potentially sensitive information that does NOT belong
to the application itself:

**Environment variables**: Catalog every env var the code reads. Flag any that look
like they could contain credentials, API keys, tokens, or secrets from OTHER systems
(e.g., AWS_SECRET_ACCESS_KEY, GITHUB_TOKEN, DATABASE_URL with embedded passwords).

**File system reads of sensitive paths**: Look for reads of:
- ~/.ssh/*, ~/.aws/*, ~/.config/gcloud/*, ~/.kube/config
- /etc/passwd, /etc/shadow, /etc/hosts
- ~/.bash_history, ~/.zsh_history
- ~/.netrc, ~/.npmrc, ~/.pypirc (credential files)
- Keychain / credential store access
- Any path construction using HOME or USERPROFILE + sensitive subdirectories

**Credential patterns in code**: Hardcoded API keys, tokens, passwords, or connection
strings in source files.

## 3. Malicious Behavior Patterns

Look for these red flags:
- **Obfuscated code**: eval(), exec(), Function(), base64-decode-then-execute chains,
  heavily encoded strings, deliberate variable name obfuscation
- **Dynamic code execution**: importing or requiring modules from URLs, dynamic imports
  from user-controlled strings
- **Reverse shells**: socket connections to external hosts with stdin/stdout/stderr
  redirection, /bin/sh or cmd.exe spawning
- **Data exfiltration**: reading local files and sending their contents to remote
  servers, especially in install/postinstall scripts
- **Cryptocurrency mining**: references to mining pools, hashrate, worker IDs, known
  mining binaries
- **Supply chain vectors**: postinstall scripts that download and execute code,
  typosquatting indicators, suspiciously recent package name claims
- **Privilege escalation**: sudo calls, setuid, capability manipulation
- **Steganography / hidden payloads**: binary blobs with embedded executables,
  images with trailing data

## 4. Install-time Behavior

Examine setup scripts, postinstall hooks, Makefiles, and build scripts for:
- Network calls during install
- Code execution during install
- File writes outside the project directory
- Environment modification (PATH, aliases, shell rc files)

## Report Format

Structure the report as:

# Security Audit: <repo-name>

## Risk Summary
One paragraph: overall risk assessment (low / medium / high / critical) with justification.

## Network Activity
[Findings organized by destination host]

## Sensitive Data Access
[Findings organized by data type]

## Malicious Behavior Scan
[Findings or "No malicious patterns detected" with brief description of what was checked]

## Install-time Behavior
[Findings from build/install scripts]

## Recommendations
[Specific, actionable recommendations if any risks were found]
```

### 2b: Business Spec (Skill invocation — runs in main context)

Invoke the `init-spec-creator` profile to generate a business specification. The spec output should go to `.audit/spec/`. After the profile completes, the spec directory will contain README.md, inventory.md, and numbered domain files.

Tell the init-spec-creator to write its output to `.audit/spec/`.

### 2c: Completeness Audit (subagent)

Dispatch a subagent:

```
You are auditing the repository at <repo-path> for completeness.
Write your findings to <repo-path>/.audit/completeness-audit.md.

Systematically search the codebase for signs of incomplete or unfinished work:

## 1. Explicit Markers
Search for these patterns (case-insensitive) across all source files:
- TODO, FIXME, HACK, XXX, TEMP, TEMPORARY
- "not implemented", "not yet implemented", "stub", "placeholder"
- "work in progress", "WIP"
- "coming soon", "TBD", "to be determined"

For each match: file path, line number, the full comment, and surrounding context
(what feature or function is incomplete).

## 2. Structural Gaps
Look for:
- Empty function/method bodies (or bodies that only raise NotImplementedError / throw)
- Functions that return hardcoded dummy values
- Commented-out code blocks (more than 5 lines — suggests deferred work)
- Feature flags that gate unreleased features
- Dead imports (imported but unused — may indicate removed/deferred features)
- Config options that are parsed but never used
- Routes/endpoints defined but with no handler logic
- Database migrations that exist without corresponding model code (or vice versa)

## 3. Documentation Gaps
- Is there a README? Is it substantive or a skeleton?
- Are there doc comments on public APIs?
- Is there a CHANGELOG?
- Are there architectural decision records or design docs?

## 4. Dependency Gaps
- Are there pinned vs unpinned dependencies?
- Are there known deprecated dependencies?
- Are there dependency declaration files that are inconsistent (e.g., requirements.txt
  vs Pipfile, or package.json vs lock file mismatch)?

## Report Format

# Completeness Audit: <repo-name>

## Summary
Overall completeness assessment: what percentage of the codebase feels production-ready
vs still in progress. Highlight the most significant gaps.

## Explicit TODOs and FIXMEs
[Table: file, line, marker, context, severity (blocking vs nice-to-have)]

## Structural Gaps
[Organized by type]

## Documentation Status
[What exists, what's missing]

## Dependency Health
[Findings]

## Prioritized Gap List
[Top 10 most impactful gaps, ranked by likely impact on users or stability]
```

### 2d: Testing Posture Audit (subagent)

Dispatch a subagent:

```
You are auditing the testing posture of the repository at <repo-path>.
Write your findings to <repo-path>/.audit/testing-audit.md.

## 1. Test Infrastructure
- What test framework(s) are in use?
- Is there a test runner configuration?
- Is there CI configuration that runs tests? (check .github/workflows, .circleci,
  Jenkinsfile, .gitlab-ci.yml, etc.)
- Are there test helper utilities, factories, fixtures?
- Is there a test database or mock server setup?

## 2. Test Coverage Assessment
- Count test files vs source files. What is the ratio?
- Which source directories/modules have corresponding test files?
- Which source directories/modules have NO corresponding test files?
- For modules that have tests, do the tests cover the main code paths or just
  happy paths?
- Are there integration tests, or only unit tests?

## 3. Test Quality
- Are tests isolated or do they depend on shared mutable state?
- Are there flaky test indicators (sleep calls, retry loops, timing-dependent
  assertions)?
- Do tests use real dependencies or mocks? Is mocking overused?
- Are edge cases tested (null inputs, empty collections, boundary values, error
  conditions)?
- Are there snapshot tests? Are the snapshots meaningful or just large blobs?

## 4. Testing Layers
Assess which layers of the application are tested:

| Layer | Coverage | Notes |
|-------|----------|-------|
| Unit (individual functions) | ? | |
| Integration (component interaction) | ? | |
| API / endpoint | ? | |
| Database / data layer | ? | |
| UI / frontend | ? | |
| End-to-end | ? | |
| Performance / load | ? | |
| Security | ? | |

## 5. Critical Paths
Identify the application's most important workflows (auth, payments, core business
logic) and assess whether they have adequate test coverage.

## Report Format

# Testing Posture: <repo-name>

## Summary
Overall testing health: strong / adequate / weak / minimal / none.
One paragraph justification.

## Test Infrastructure
[Framework, CI, tooling findings]

## Coverage Map
[Which parts are tested, which aren't — organized by module/feature]

## Testing Layers
[The layer table from above, filled in]

## Quality Assessment
[Flakiness, isolation, edge cases, mock usage]

## Critical Path Coverage
[Are the most important workflows tested?]

## Biggest Gaps
[Top 5 testing gaps ranked by risk — what could break in production and wouldn't
be caught by the current test suite?]
```

### Parallelism Strategy

Since the spec (2b) is invoked via the Skill tool in the main context, it cannot run truly in parallel with subagents. The recommended execution order:

1. Dispatch subagents for **2a** (security), **2c** (completeness), and **2d** (testing) in a single turn using the Agent tool — all three run concurrently.
2. While those run, invoke the **init-spec-creator** profile (2b) in the main context.
3. Wait for all four to complete before proceeding to Phase 3.

## Phase 3: Code Quality Assessment

After Phase 2 completes (the spec is needed as context), invoke the `code-quality-assessor` profile. Point it at the repo and tell it to write its output to `.audit/code-quality-audit.md`.

The spec from Phase 2b provides business context that helps the code quality assessment understand what the code is *supposed* to do, making its analysis more meaningful.

## Phase 4: Summary

After all audits are complete, read all five reports from `.audit/` and produce a unified summary at `.audit/summary.md`:

```markdown
# Research Summary: <repo-name>

## At a Glance
| Dimension | Rating | Key Finding |
|-----------|--------|-------------|
| Security | [safe/caution/danger] | [one-line summary] |
| Completeness | [complete/mostly/partial/early] | [one-line summary] |
| Testing | [strong/adequate/weak/minimal] | [one-line summary] |
| Code Quality | [excellent/good/fair/poor] | [one-line summary] |

## What This Software Does
[2-3 sentence plain-English summary derived from the business spec]

## Should You Use This?
[Honest recommendation based on all audit findings. Consider: Is it safe?
Is it finished enough? Is it well-tested? Is the code maintainable?]

## Top Concerns
[Ranked list of the 5 most important findings across all audits]

## Top Strengths
[What this project does well]
```

Present the summary to the user when done. Let them know the full reports are in `.audit/` if they want to dig deeper into any area.
