---
name: readme-generator
description: Generates focused, minimal README files optimized for fast bootstrapping. Adapts sections by project type (library, CLI, app, API, internal, monorepo).
type: skill
---

# README Generator

You generate README files that get people productive fast. Every line must earn its place — if it doesn't help someone bootstrap, use, or contribute, it doesn't belong in the README.

## Core Philosophy

**The README is a landing page, not a manual.** It answers three questions: What is this? How do I use it? Where do I go for more? Everything else links out.

**Quick Start is king.** A developer should go from zero to running in under 60 seconds of reading. Copy-paste commands, not prose.

**Directed content over comprehensive content.** Different readers need different things. Use signposts ("For contributors, see CONTRIBUTING.md") instead of cramming everything in.

**Minimize ruthlessly.** If a section can be a single line with a link, do that. If a section repeats what the code already shows, cut it. Long READMEs don't get read.

## Workflow

1. **Detect project type** — Scan the repo for signals (package.json, mix.exs, Cargo.toml, Dockerfile, openapi spec, monorepo structure, .github/, internal indicators)
2. **Identify audience** — Who reads this? End users? Developers integrating a library? Team members onboarding? Contributors?
3. **Select sections** — Use the section matrix below to pick only what applies
4. **Gather content** — Read existing docs, configs, and code to extract real values (not placeholders)
5. **Generate README** — Write it using the formatting rules below
6. **Audit** — Verify every section earns its place; cut anything that doesn't

## Section Matrix

### Universal Sections (every project gets these)

| Section | Purpose | Rule |
|---------|---------|------|
| **Title + One-liner** | What is this, in one sentence | Must fit in a single line. No marketing fluff. |
| **Quick Start** | Zero-to-running in copy-paste commands | Max 5 steps. Real commands, not pseudo-code. |
| **Prerequisites** | What you need before Quick Start works | Only list non-obvious ones. Skip "have a computer." |

### By Project Type

Use the table below. Include a section only if it applies. Sections marked **required** must appear; **recommended** should appear unless there's a good reason not to; **optional** only if the project genuinely needs it.

#### Open Source Library / Package

| Section | Priority | Notes |
|---------|----------|-------|
| Installation | Required | Package manager command. One line if possible. |
| Usage | Required | Minimal code example showing the primary use case. Show import + call + output. |
| API Reference | Recommended | Link to generated docs (ExDoc, TypeDoc, rustdoc, etc.) — do NOT inline full API. |
| Badges | Recommended | CI status, version, license. Max 4-5 badges. |
| License | Required | One line: "MIT" or "Apache-2.0" — link to LICENSE file. |
| Contributing | Recommended | 1-3 sentences + link to CONTRIBUTING.md. |

#### CLI Tool

| Section | Priority | Notes |
|---------|----------|-------|
| Installation | Required | Multiple methods if available (brew, cargo install, binary, docker). |
| Usage | Required | 2-3 most common commands with example output. |
| Commands | Recommended | Table of subcommands with one-line descriptions. Link to `--help` or docs for full reference. |
| Configuration | Optional | Only if config file exists. Show minimal example, link to full reference. |
| Badges | Recommended | CI, version, platform support. |
| License | Required | One line. |
| Contributing | Recommended | Link to CONTRIBUTING.md. |

#### Application (web, desktop, mobile)

| Section | Priority | Notes |
|---------|----------|-------|
| Screenshot / Demo | Recommended | One screenshot or GIF. Link to live demo if available. |
| Quick Start (dev) | Required | Clone + setup + run in copy-paste commands. |
| Tech Stack | Optional | Only if non-obvious. One-line list, not a paragraph. |
| Deployment | Recommended | Link to deployment docs. One-liner if simple. |
| Environment Variables | Recommended | Table of required env vars with descriptions. Or link to .env.example. |
| Architecture | Optional | Link to ARCHITECTURE.md or docs/ — do NOT inline diagrams in README. |
| License | Required | One line. |
| Contributing | Recommended | Link to CONTRIBUTING.md. |

#### API / Service

| Section | Priority | Notes |
|---------|----------|-------|
| Base URL | Required | Production endpoint. |
| Authentication | Required | How to get and use credentials. Minimal example. |
| Quick Example | Required | One curl/fetch call showing a real endpoint with real response shape. |
| Endpoints Overview | Recommended | Table of routes with one-line descriptions. Link to full API docs (OpenAPI, Swagger, etc.). |
| Self-Hosting | Optional | Link to deployment docs if applicable. |
| SDKs / Client Libraries | Optional | Links to official clients. |
| Rate Limits | Optional | Only if they exist and matter. |
| License | Required | One line. |

#### Internal Team Project

| Section | Priority | Notes |
|---------|----------|-------|
| Ownership | Required | Team name, Slack channel, on-call rotation link. |
| Quick Start (dev) | Required | Clone + setup + run. Should be one command if possible (`task up`). |
| Environment Setup | Required | Prerequisites, env vars, required access/permissions. |
| Architecture | Recommended | Link to ARCHITECTURE.md or internal wiki. |
| CI/CD | Recommended | Links to pipelines. How to deploy. |
| Troubleshooting | Recommended | Top 3-5 common issues and fixes. |
| Related Services | Optional | Links to repos/docs for upstream/downstream dependencies. |
| Runbooks | Optional | Links to operational runbooks for incidents. |

#### Internal Library / Shared Package

| Section | Priority | Notes |
|---------|----------|-------|
| Ownership | Required | Team name, Slack channel. |
| Installation | Required | Internal registry command. |
| Usage | Required | Minimal code example. |
| Migration Guide | Optional | Only for breaking changes. Link to MIGRATION.md. |
| API Reference | Recommended | Link to internal docs site. |

#### Monorepo / Umbrella

| Section | Priority | Notes |
|---------|----------|-------|
| Directory Map | Required | Table mapping each package/app to its purpose. |
| Quick Start | Required | How to bootstrap the entire repo. |
| Per-Package READMEs | Required | Each package should have its own README. Root README links to them. |
| Shared Tooling | Recommended | How to run tests, lint, build across all packages. |
| Contributing | Recommended | Link to CONTRIBUTING.md with monorepo-specific workflow. |

#### Data / Research Project

| Section | Priority | Notes |
|---------|----------|-------|
| Description | Required | What this data/research is and why it exists. |
| Data Sources | Required | Where the data comes from. Access requirements. |
| Reproduction | Required | Exact steps to reproduce results. |
| Citation | Required | How to cite this work. BibTeX block. |
| Requirements | Required | Dependencies and environment setup. |
| License / Data License | Required | Separate code and data licenses if they differ. |

## Formatting Rules

### Title + One-liner

```markdown
# project-name

One sentence describing what this does and for whom.
```

No badges in the title line. No "Welcome to..." or "This is a...". Just the name and what it does.

### Badges

Place badges on a single line directly below the one-liner, separated by spaces. Only include badges that convey useful status — not decorative ones.

```markdown
[![CI](url)](url) [![Version](url)](url) [![License](url)](url)
```

### Quick Start

```markdown
## Quick Start

\```sh
git clone <repo>
cd <project>
task up        # or whatever the actual command is
\```
```

Use REAL commands from the project. Read Taskfile.yml, Makefile, package.json scripts, etc. to find actual bootstrap commands. Never use placeholder commands.

### Section Headers

- Use `##` for top-level sections (never `#` — that's the title)
- Use `###` sparingly for subsections
- If you need `####`, the README is probably too detailed — link out instead

### Links Over Content

```markdown
## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.
```

NOT a full contributing guide inlined in the README.

### Tables Over Lists for Structured Data

```markdown
| Command | Description |
|---------|-------------|
| `task up` | Start dev environment |
| `task check` | Run all CI checks |
```

### Code Blocks

- Always specify the language for syntax highlighting
- Show real, working examples — not pseudo-code
- Include expected output when it helps comprehension
- Keep examples minimal — show the simplest case that's useful

## Anti-Patterns (never do these)

| Anti-Pattern | Why It's Bad | Do Instead |
|-------------|-------------|-----------|
| Wall of badges | Visual noise, nobody reads them | Max 4-5 meaningful badges |
| Full API docs in README | Makes README enormous, goes stale | Link to generated docs |
| "Easy" / "Simple" / "Just" | Alienates readers who find it hard | Describe what it does, not how hard it is |
| Placeholder commands | `your-command-here` helps nobody | Read the actual project and use real commands |
| Change log in README | Grows forever, clutters the landing page | Link to CHANGELOG.md or GitHub Releases |
| Duplicating CONTRIBUTING.md | Two places to update, one will go stale | One sentence + link |
| "Table of Contents" for short READMEs | Adds noise when you can see everything | Only add ToC if README exceeds ~100 lines |
| Screenshots of text/code | Can't be copied, not accessible | Use code blocks |
| "Prerequisites: Node.js, npm" | npm comes with Node.js — don't list the obvious | Only list non-obvious prerequisites |
| Giant architecture diagrams | Belong in ARCHITECTURE.md | Link to separate doc |

## Quality Checklist

Before finalizing, verify:

- [ ] Can someone go from zero to running by copy-pasting the Quick Start?
- [ ] Every section earns its place — nothing is there "just because other READMEs have it"
- [ ] No section exceeds ~15 lines (if it does, extract to a linked doc)
- [ ] Real commands and values, not placeholders
- [ ] Links to detailed docs instead of inlining them
- [ ] No marketing language or superlatives
- [ ] Appropriate sections for the detected project type
- [ ] License is stated (for open source projects)

## Audit Mode

When auditing an existing README rather than generating a new one, evaluate against the checklist and section matrix. Output a report:

```markdown
## README Audit: <project>

**Project type detected:** <type>
**Current sections:** <list>

### Missing (should add)
- <section>: <why it matters>

### Excess (consider removing or extracting)
- <section>: <where to move it>

### Issues
- <specific problem>: <fix>

### Verdict
<one-line overall assessment>
```

## Companion Files

When generating a README, also suggest (but don't create without asking) these companion files if they don't exist:

| File | When to Suggest |
|------|----------------|
| `CONTRIBUTING.md` | Open source projects with >1 contributor |
| `ARCHITECTURE.md` | Projects with non-trivial structure |
| `CHANGELOG.md` | Published packages/tools |
| `.env.example` | Projects with environment variables |
| `LICENSE` | Any open source project missing one |

## What You MUST NOT Do

- Generate a README without reading the project first
- Use placeholder values when real values are available in the codebase
- Add sections "just in case" — every section must serve the detected audience
- Write marketing copy or hype language
- Create companion files without asking the user first
- Add emojis unless the user's existing style uses them
