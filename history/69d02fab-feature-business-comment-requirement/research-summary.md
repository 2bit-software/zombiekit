---
status: complete
updated: 2026-04-03
---

# Research: Business-Requirement Function Comments

## Executive Summary

The codebase already enforces business-language comments for tests (commit 807ef78) via spec and task templates. No equivalent requirement exists for non-test functions. The implement profile (`embed/profiles/implement.md`) is the natural shared enforcement point since all workflows delegate to it.

## Findings

### Codebase Context

- **Workflow files**: `embed/workflows/{feature,feature-light,bug,refactor,unmanaged}.md`
- **Implement profile**: `embed/profiles/implement.md` — loaded by feature, feature-light, bug, and refactor workflows during their implement steps
- **Existing test comment rule**: Lives in `.brains/templates/spec-template.md` (lines 129-138) and `.brains/templates/tasks-template.md` (lines 253-266)
- **STANDARDS.md**: Requires doc comments on exported names, standard Go format. Does not mention business-language framing.
- **Current function comments**: Standard Go style (`// Name does X.`), mostly technical descriptions

### Domain Knowledge

- Go convention: doc comments start with the declared name and describe purpose
- Business-language comments improve spec traceability without embedding spec IDs in code
- The test comment requirement already established the pattern and examples format

## Decision Points

1. **Where to add the requirement**: The implement profile is the single shared point. Adding it there covers all workflows without duplication.
2. **Template additions**: spec-template.md and tasks-template.md should also include the rule (parallel to existing test comment rule) so specs remind implementers.
3. **STANDARDS.md update**: Add a section on business-language doc comments to make it a permanent project standard, not just a workflow instruction.
