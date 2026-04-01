# Audit Pass 1 — Summary

**Date**: 2026-03-31
**Result**: CRITICAL and MAJOR findings identified, spec updated.

## Key Findings Addressed

| ID | Severity | Finding | Resolution |
|----|----------|---------|------------|
| C1 | CRITICAL | Runner lifecycle ambiguity | FR-002 now mandates "construct new Runner per call" |
| C2 | CRITICAL | Push action missing from acceptance scenarios | Added push to US1 scenario 6 |
| C3 | CRITICAL | Schema registration not specified | FR-009 added with exact `mcp.WithString` call |
| M1 | MAJOR | Relative path as recommendation | FR-008 added as requirement |
| M2 | MAJOR | Error codes not specified | FR-004/FR-005 now specify `INVALID_DIRECTORY` and `NOT_GIT_REPOSITORY` |
| M3 | MAJOR | MCP schema in server.go not mentioned | FR-009 + Files to Modify section added |
| M4 | MAJOR | API contract not stated | FR-010 added (signatures must not change) |
| M5 | MAJOR | SC-002 "every action" not enumerated | SC-002 now lists all six actions explicitly |
| M6 | MAJOR | Server init when no default workDir | Out of scope for this ticket (tool requires server workDir to register) |
| M7 | MAJOR | Permission denied edge case | Added to edge cases (git error surfaces as-is) |
| m1 | MINOR | Files to modify not listed | Added "Files to Modify" section |
| m2 | MINOR | US2 scenario 2 untestable | Rephrased to testable assertion |
| m3 | MINOR | Empty string edge case phrasing | Changed to directive ("MUST behave as if omitted") |
| m4 | MINOR | Directory description not specified | Included in FR-009 |
| m5 | MINOR | `~` expansion not addressed | Added to edge cases |
| m6 | MINOR | File-not-directory edge case | Added to US3 scenario 3 and edge cases |
| m7 | MINOR | Scope not stated | Added Scope section (gh-pr out of scope) |
