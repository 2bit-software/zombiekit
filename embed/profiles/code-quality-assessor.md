---
name: code-quality-assessor
description: Performs a comprehensive code quality assessment of a codebase, evaluating DRY principles, modularity, abstraction quality, naming conventions, error handling, dependency architecture, and overall maintainability.
type: skill
---

# Code Quality Assessor

Performs a deep, structured assessment of a codebase's engineering quality. Designed to answer the question: "Is this code well-built enough to trust, maintain, and build on?"

## Input

A path to a codebase to assess. If no path is given, use the current working directory.

## Optional Context

If a business spec exists at `.audit/spec/` (produced by `init-spec-creator`), read it first. The spec provides context about what the software is *supposed* to do, which sharpens the quality analysis — you can evaluate whether the code structure actually reflects the business domains, whether abstractions align with real concepts, and whether modules are organized around capabilities rather than technical layers.

If no spec is available, proceed without it. The audit is still valuable.

## Output

Write the report to `.audit/code-quality-audit.md` (create `.audit/` if it doesn't exist). If running standalone (not from `github-research`), also present a summary to the user.

## Assessment Process

### Step 1: Orientation

Before diving into details, get a feel for the codebase:

- What language(s) and framework(s)?
- How large is it? (file count, rough LOC)
- What's the top-level directory structure?
- Is there a clear organizational principle (by feature, by layer, by domain)?
- What's the dependency management approach?

This context frames everything that follows. A 500-line CLI tool should be judged differently from a 50k-line distributed system.

### Step 2: Assess Each Dimension

Evaluate the following dimensions. For each, provide specific examples from the code — cite file paths and line numbers. Don't just say "naming is inconsistent"; show where and how.

---

#### 2a: Repetition and DRY

Look for duplicated logic — not just identical code, but code that does the same thing in slightly different ways.

**What to look for:**
- Copy-pasted functions or blocks with minor variations
- Multiple implementations of the same business rule in different places
- Repeated patterns that should be extracted (e.g., the same error handling wrapper in 10 places, the same data transformation in every controller)
- Configuration or constants duplicated across files instead of centralized
- Similar data structures defined independently rather than sharing a base

**What NOT to flag:**
- Intentional repetition for clarity (sometimes two similar functions are clearer than one overparameterized abstraction)
- Test code that repeats setup — test readability often trumps DRY
- Protocol/interface implementations that look similar but serve different contracts

The goal is to find repetition that creates maintenance risk — where changing a business rule means finding and updating 5 places instead of 1.

---

#### 2b: Modularity and Cohesion

Evaluate whether the code is organized into modules that have clear, focused responsibilities.

**What to look for:**
- **High cohesion**: Do files/modules/classes group related functionality? Or are they grab-bags of loosely related functions?
- **Low coupling**: Can you understand one module without reading three others? Are there circular dependencies?
- **Module size**: Are there god files/classes that do too much? (>500 lines in a single file is a smell; >1000 is almost always a problem)
- **Responsibility boundaries**: Does each module own its data and behavior, or are there modules that reach into others' internals?
- **Feature vs layer organization**: Is code organized by what it does (better for most apps) or by technical layer (controllers/, models/, services/ — often leads to shotgun surgery)?

**Assess the module graph:**
- Draw the high-level dependency graph mentally. Is it roughly hierarchical (good) or a tangled web (bad)?
- Are there unexpected dependencies (e.g., a utility module importing from a feature module)?
- Could you delete a feature module without cascading breakage?

---

#### 2c: Abstraction Quality

This is about whether the boundaries between layers and components are clean, or whether implementation details leak through.

**Leaky abstractions to look for:**
- Database column names or query syntax appearing in API responses or UI code
- HTTP status codes or request/response shapes leaking into business logic
- ORM-specific patterns (like `.save()`, `.query()`) used outside the data layer
- Framework-specific types passed through layers that should be framework-agnostic
- Error types from one layer propagated raw to another (e.g., a SQL constraint error surfacing as an API response)
- Configuration details (connection strings, file paths) hardcoded in business logic rather than injected

**Good abstractions to note:**
- Clear interfaces/contracts between layers
- Domain objects that represent business concepts independent of persistence or transport
- Adapter patterns that isolate external dependencies
- Error types that translate between layers

---

#### 2d: Naming and Readability

Code is read far more than it's written. Evaluate how easy it is to understand what the code does by reading it.

**What to look for:**
- **Naming consistency**: Does the codebase use one convention or a mix? (camelCase in some files, snake_case in others within the same language)
- **Naming clarity**: Do names describe what something IS or DOES, or are they vague (data, result, item, handle, process, manager)?
- **Abbreviation discipline**: Are abbreviations consistent and well-known, or cryptic? (ctx is fine; prcsEnt is not)
- **Boolean naming**: Do boolean variables/functions read naturally? (isValid, hasPermission vs. flag, check)
- **Function length**: Can you understand what a function does without scrolling? Functions over 40 lines deserve scrutiny.
- **Nesting depth**: Are there deeply nested conditionals or loops (>3 levels)? These are hard to reason about.
- **Magic values**: Are there unexplained literal numbers or strings? (if status == 7 — what is 7?)

---

#### 2e: Error Handling

How does the codebase deal with things going wrong?

**What to look for:**
- **Empty catch blocks**: Errors silently swallowed — the worst pattern
- **Catch-and-log-only**: Logging the error but continuing as if nothing happened
- **Error context**: Are errors wrapped with context as they propagate, or do you get a bare "null pointer exception" with no clue where it originated?
- **Error types**: Does the code distinguish between different kinds of failures (user input error vs. system failure vs. transient issue)?
- **Consistency**: Is error handling done the same way throughout, or does each file invent its own approach?
- **Boundary validation**: Is input validated at system boundaries (API endpoints, CLI args, file reads) or scattered throughout?
- **Recovery strategy**: Does the code have a clear strategy for recoverable vs. unrecoverable errors?

---

#### 2f: Dependency Architecture

Evaluate the health and structure of external and internal dependencies.

**External dependencies:**
- How many direct dependencies? Is the count reasonable for what the software does?
- Are there redundant dependencies (two libraries that do the same thing)?
- Are there heavyweight dependencies pulled in for trivial use? (e.g., lodash for a single function)
- Are dependency versions pinned or floating?
- Are there vendored/copied dependencies instead of proper package management?

**Internal dependency direction:**
- Do dependencies flow in one direction (e.g., handlers → services → repositories), or are there cycles?
- Does core business logic depend on infrastructure, or is it the other way around (infrastructure depends on core)?
- Are there circular imports?
- Could you swap out the database or API framework without rewriting business logic?

---

#### 2g: Consistency and Conventions

Beyond naming, does the codebase feel like it was written by one team with shared standards?

**What to look for:**
- **File structure patterns**: Are similar features structured the same way, or does each one have its own layout?
- **Code style**: Is formatting consistent? (This matters less with formatters, but inconsistency suggests no shared tooling)
- **Patterns**: Does the codebase pick one way to do things and stick with it? (e.g., one state management pattern, one way to define routes, one approach to async)
- **Evolutionary inconsistency**: Are there "old style" and "new style" sections that suggest mid-stream changes in approach without backfilling?

---

### Step 3: Identify Architectural Strengths

Don't just catalog problems. Explicitly note what the codebase does well:
- Clean separation that makes the code easy to reason about
- Well-chosen abstractions that simplify complex domains
- Defensive patterns that prevent common bugs
- Good use of the language/framework's idioms
- Evidence of intentional design decisions

### Step 4: Synthesize

Pull the dimension assessments into an overall evaluation.

## Report Format

```markdown
# Code Quality Assessment: <project-name>

## Overview
| Dimension | Rating | Summary |
|-----------|--------|---------|
| DRY / Repetition | [minimal/moderate/significant] | [one line] |
| Modularity | [strong/adequate/weak] | [one line] |
| Abstraction Quality | [clean/mixed/leaky] | [one line] |
| Naming & Readability | [clear/mostly clear/inconsistent/poor] | [one line] |
| Error Handling | [robust/adequate/inconsistent/poor] | [one line] |
| Dependency Architecture | [healthy/acceptable/concerning] | [one line] |
| Consistency | [unified/mostly/fragmented] | [one line] |

**Overall: [excellent / good / fair / poor]**

## Codebase Profile
[Language, size, structure, organizational principle — from Step 1]

## Strengths
[What this codebase does well — specific examples with file references]

## DRY / Repetition
[Findings with file:line citations]

## Modularity and Cohesion
[Findings with dependency direction analysis]

## Abstraction Quality
[Leaky abstractions found, with specific examples of what leaks where]

## Naming and Readability
[Patterns observed, good and bad, with examples]

## Error Handling
[Strategy assessment, gaps found]

## Dependency Architecture
[External dependency health, internal dependency direction]

## Consistency and Conventions
[Where the codebase is unified vs fragmented]

## Verdict

### Would you build on this code?
[Honest 2-3 sentence answer. Consider: If you inherited this codebase tomorrow,
how would you feel? What would you want to fix first vs what could you live with?]

### Top 5 Improvements
[Ranked by impact on maintainability. For each: what to change, why it matters,
and rough scope of effort (quick fix / moderate refactor / significant rework)]

### Technical Debt Estimate
[Low / moderate / high / critical. Brief justification — is the debt manageable
and well-contained, or is it structural and compounding?]
```

## Calibration

Not every codebase needs to be pristine. Calibrate your assessment to the project's context:

- **A weekend project or prototype**: Don't penalize lack of abstraction layers. Focus on whether the code is understandable and not doing anything dangerous.
- **A library meant for public consumption**: Hold to a higher standard on API design, naming, documentation, and abstraction boundaries.
- **A production application**: Focus on maintainability, error handling, and whether the architecture will scale with the team.
- **A large legacy codebase**: Look for whether there's a clear direction of improvement, not whether it's perfect today.

State your calibration context at the top of the report so the reader understands the lens you're using.
