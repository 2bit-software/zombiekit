# Test Comment Business Language

## What
Replace spec-reference comments on tests (BR-XXX, FR-XXX, T0XX, US-X patterns) with short business-language descriptions of the test scenario. Section dividers with spec refs also get rewritten.

## Constraints
- Don't change test names, only comments
- Use business language: what the scenario verifies from a user/system perspective
- Technical details OK when the requirement is inherently technical
- Remove task IDs (T008, T025, etc.) and spec IDs (BR-001, FR-013, etc.)
- Keep section divider formatting (=== lines) where they exist

## Acceptance Criteria
- No `BR-XXX`, `FR-XXX`, `T0XX [P]`, `[US1]` etc. remain in test comments
- Every replaced comment describes the test scenario in plain English
- Tests still compile and pass unchanged
