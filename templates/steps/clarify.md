---
name: clarify
description: Identify underspecified areas and generate targeted clarification questions
profiles: []
files:
  - "spec.md"
  - "plan.md"
  - "research.md"
type: step
---
# Clarification Workflow

## Context

You are identifying underspecified areas in the current specification and generating targeted clarification questions. Your goal is to surface ambiguities before they cause implementation problems, then encode answers back into the specification.

### Your Responsibilities

- Review specification for ambiguities and gaps
- Categorize questions by type (requirements, edge cases, constraints, dependencies)
- Generate up to 5 highly targeted questions
- Encode answers back into spec.md

### System Responsibilities (handled by MCP tool)

- File path resolution
- State management

---

## Response Handling

When you receive the MCP response, process fields in this order:

1. **Check `prerequisites.met`**: If false, follow `prerequisites.hint` to unblock
2. **Read `files_to_read`**: Load spec.md, plan.md, research.md
3. **Follow `directive`**: Execute according to this document
4. **Output to `cycle_folder`**: Update spec.md with clarifications

---

## Workflow

### Step 1: Analyze Specification

Scan for ambiguity indicators:

| Indicator | Example | Risk Level |
|-----------|---------|------------|
| Vague terms | "reasonable time", "appropriate" | HIGH |
| Missing quantities | "several", "some", "many" | HIGH |
| Conditional gaps | "if applicable", "when needed" | MEDIUM |
| Implicit requirements | Unstated but assumed behavior | HIGH |
| Edge case gaps | Happy path only | MEDIUM |
| Dependency assumptions | "assumes X exists" | MEDIUM |

### Step 2: Categorize Questions

Use this taxonomy:

| Category | Description | Examples |
|----------|-------------|----------|
| **REQUIREMENTS** | Core functionality unclear | "Should X do Y or Z?" |
| **EDGE_CASES** | Boundary conditions | "What happens when input is empty?" |
| **CONSTRAINTS** | Limits and bounds | "What's the max file size?" |
| **DEPENDENCIES** | External assumptions | "Does this require feature X?" |
| **PRIORITY** | Importance/ordering | "Is A more important than B?" |

### Step 3: Formulate Questions

Each question should be:

1. **Specific**: Ask about one thing
2. **Bounded**: Limited answer space (not open-ended)
3. **Actionable**: Answer leads to spec change
4. **High-Impact**: Affects implementation significantly

Question format:

```markdown
### Q{N}: {Short title}

**Category**: {REQUIREMENTS | EDGE_CASES | CONSTRAINTS | DEPENDENCIES | PRIORITY}
**Artifact**: {spec.md section reference}
**Issue**: {What's unclear}
**Options** (if applicable):
- A: {Option description}
- B: {Option description}
**Impact**: {Why this matters for implementation}
```

### Step 4: Present Questions

Limit to 5 questions maximum. Prioritize by impact.

### Step 5: Encode Answers

When user provides answers:

1. Update spec.md with clarified requirements
2. Mark the clarification source (e.g., "[Clarified: 2025-01-15]")
3. Remove any NEEDS CLARIFICATION markers
4. Ensure the answer is testable

Example encoding:

```markdown
# Before (in spec.md)
The system should respond in a reasonable time.

# After (in spec.md)
The system must respond within 200ms for 95th percentile requests. [Clarified: 2025-01-15]
```

---

## Output

### Question Phase

Present questions to user:

```markdown
# Clarification Questions: {Feature Name}

**Specification**: [spec.md](./spec.md)
**Date**: {YYYY-MM-DD}

I found {N} areas needing clarification. Please answer these questions:

---

### Q1: Authentication timeout duration

**Category**: CONSTRAINTS
**Artifact**: spec.md § User Authentication
**Issue**: "Session expires after inactivity" - duration not specified
**Options**:
- A: 15 minutes (banking standard)
- B: 1 hour (typical web app)
- C: 24 hours (convenience-focused)
- D: Configurable per user preference
**Impact**: Affects UX, security posture, and session management implementation

---

### Q2: Password complexity requirements

**Category**: REQUIREMENTS
**Artifact**: spec.md § Account Creation
**Issue**: "Strong password required" - criteria not defined
**Options**:
- A: 8+ chars, 1 uppercase, 1 number (basic)
- B: 12+ chars, mixed case, number, symbol (strict)
- C: NIST guidelines (passphrase-friendly, no arbitrary rules)
**Impact**: Affects validation logic, user experience, compliance

---

{more questions...}

---

Please provide answers for each question. I'll encode them into the specification.
```

### Answer Encoding Phase

After receiving answers:

```markdown
# Clarification Complete

Updated spec.md with the following clarifications:

1. **Q1**: Session timeout → 1 hour (Option B)
   - Added to § User Authentication: "Sessions expire after 60 minutes of inactivity."

2. **Q2**: Password requirements → NIST guidelines (Option C)
   - Added to § Account Creation: "Passwords must be minimum 8 characters. No arbitrary complexity rules. Check against common password lists."

**Remaining Ambiguities**: None identified

**Next Step**: Continue with /brains.plan or /brains.tasks
```

---

## Success Criteria

- [ ] Specification reviewed for all ambiguity indicators
- [ ] Questions categorized by type
- [ ] Maximum 5 questions (prioritized)
- [ ] Each question has bounded options when possible
- [ ] Answers encoded back into spec.md
- [ ] No NEEDS CLARIFICATION markers remain

---

## Behavior Rules

1. **Limit Questions**: Maximum 5 per clarification round
2. **Prioritize Impact**: Ask about things that affect implementation most
3. **Be Specific**: Vague questions get vague answers
4. **Provide Options**: When possible, give bounded choices
5. **Encode Immediately**: Don't leave answers floating—update the spec
6. **Cite Sources**: Mark where clarifications came from
7. **Remove Markers**: Delete NEEDS CLARIFICATION after resolving
8. **Stay in Scope**: Don't expand requirements, just clarify existing ones

---

## Anti-Patterns

Avoid these question types:

| Bad | Why | Better |
|-----|-----|--------|
| "What should the system do?" | Too open-ended | "Should X happen before or after Y?" |
| "Is this important?" | Subjective | "If X fails, should we retry or abort?" |
| "Any other requirements?" | Fishing | Focus on specific gaps you found |
| "How should we implement X?" | Implementation, not specification | "What behavior is expected when X occurs?" |
