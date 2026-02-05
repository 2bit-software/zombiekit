---
name: audit
description: Cross-artifact alignment check with severity classification
profiles: []
files:
  - "spec.md"
  - "plan.md"
  - "tasks.md"
  - "data-model.md"
  - "contracts/**/*.md"
  - "audit/**/*.md"
type: step
---
# Cross-Artifact Audit Workflow

## Context

You are performing a cross-artifact alignment check. Your goal is to verify that the implementation aligns with the specification, plan, and contracts. You classify findings by severity and recommend corrective actions.

### Your Responsibilities

- Check alignment between spec → plan → tasks → implementation
- Classify issues by severity (CRITICAL, MAJOR, MINOR, INFO)
- Verify test coverage matches requirements
- Document findings with fix suggestions
- Recommend next steps

### System Responsibilities (handled by MCP tool)

- File path resolution
- Profile composition
- State management

---

## Response Handling

When you receive the MCP response, process fields in this order:

1. **Check `prerequisites.met`**: If false, follow `prerequisites.hint` to unblock
2. **Read `files_to_read`**: Load all design documents and previous audits
3. **Follow `directive`**: Execute according to this document
4. **Output to `cycle_folder`**: Save audit report to audit/{YYYY-MM-DD}.md

---

## Workflow

### Step 1: Load Artifacts

Read and catalog:

| Artifact | Purpose |
|----------|---------|
| spec.md | Requirements to verify |
| plan.md | Technical decisions |
| tasks.md | Task completion status |
| data-model.md | Entity definitions |
| contracts/ | API contracts |
| Previous audits | Historical issues |

### Step 2: Check Alignment Matrix

For each requirement in spec.md:

```
Requirement → Plan Coverage → Task Coverage → Implementation Status → Test Coverage
```

Document gaps at each level.

### Step 3: Classify Findings

Use severity levels:

| Severity | Criteria | Action Required |
|----------|----------|-----------------|
| **CRITICAL** | Blocks release; security/data integrity risk; missing core functionality | Must fix before proceeding |
| **MAJOR** | Significant gap; user-facing issue; incomplete requirement | Should fix before release |
| **MINOR** | Suboptimal implementation; minor inconsistency; edge case gaps | Consider fixing |
| **INFO** | Observation; suggestion; documentation opportunity | Optional improvement |

### Step 4: Document Findings

For each finding:

```markdown
### [SEVERITY] Title

**Artifact**: {spec.md | plan.md | tasks.md | code}
**Location**: {file:line or section reference}
**Issue**: {What's wrong}
**Expected**: {What should be}
**Impact**: {Why this matters}
**Fix Suggestion**: {How to resolve}
```

### Step 5: Generate Report

Create audit report with:
1. Summary counts by severity
2. Alignment matrix
3. Detailed findings
4. Recommendations

---

## Output

Create `audit/{YYYY-MM-DD}.md` in `cycle_folder`:

```markdown
# Audit Report: {Feature Name}

**Date**: {YYYY-MM-DD}
**Auditor**: AI Agent
**Scope**: Cross-artifact alignment check

## Summary

| Severity | Count |
|----------|-------|
| CRITICAL | 0 |
| MAJOR | 2 |
| MINOR | 3 |
| INFO | 1 |

**Overall Status**: {PASS | FAIL | CONDITIONAL}

## Alignment Matrix

| Requirement | Plan | Tasks | Impl | Tests | Status |
|-------------|------|-------|------|-------|--------|
| FR-001: User login | Y | Y | Y | Y | PASS |
| FR-002: Password reset | Y | Y | N | N | FAIL |
| FR-003: Session management | Y | P | N | N | PARTIAL |

Legend: Y=Complete, N=Missing, P=Partial

## Findings

### [MAJOR] Password reset not implemented

**Artifact**: spec.md → implementation
**Location**: spec.md section "Password Management"
**Issue**: Requirement FR-002 specifies password reset flow but no implementation exists
**Expected**: POST /api/reset-password endpoint with email validation
**Impact**: Users cannot recover accounts; support burden
**Fix Suggestion**: Add task T025 for password reset implementation

### [MINOR] Missing edge case test

**Artifact**: contracts/auth-contract.md
**Location**: Section "Error Cases"
**Issue**: Contract specifies rate limiting but no test verifies it
**Expected**: Test case for 429 response after 5 failed attempts
**Impact**: Rate limiting may silently break
**Fix Suggestion**: Add integration test in auth_test.go

## Recommendations

1. **BLOCK**: Do not proceed until CRITICAL issues resolved
   - (none)

2. **FIX BEFORE RELEASE**: Address MAJOR issues
   - Implement password reset (FR-002)
   - Add session timeout handling (FR-003)

3. **CONSIDER**: Address MINOR issues if time permits
   - Add rate limiting test
   - Improve error messages

4. **OPTIONAL**: INFO items for future consideration
   - Consider adding metrics endpoint

## Next Steps

{Based on findings, recommend: continue | fix and re-audit | escalate}
```

---

## Success Criteria

- [ ] All artifacts loaded and reviewed
- [ ] Alignment matrix complete
- [ ] Findings classified by severity
- [ ] Each finding has fix suggestion
- [ ] Clear recommendation for next steps

---

## Behavior Rules

1. **Be Thorough**: Check every requirement, not just obvious ones
2. **Be Specific**: Reference exact locations and artifacts
3. **Be Actionable**: Every finding needs a fix suggestion
4. **Be Objective**: Classify by actual impact, not perceived effort
5. **Severity Matters**: CRITICAL = blocks, MAJOR = should fix, MINOR = could fix
6. **Track Progress**: Compare against previous audits if available
7. **Recommend Clearly**: Don't leave ambiguity about next steps

---

## Audit Types

### Specification Audit (during feature/bug/refactor)

Focus:
- Completeness of specification
- Clarity and testability
- Edge cases identified
- No implementation details leaked

### Implementation Audit (after implement step)

Focus:
- Requirement coverage
- Test coverage
- Code quality alignment with plan
- Contract compliance

### Final Audit (before release)

Focus:
- All requirements met
- All tests passing
- Documentation complete
- No CRITICAL or MAJOR issues remain
