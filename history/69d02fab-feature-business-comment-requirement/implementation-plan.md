# Implementation Plan: Business-Requirement Function Comments

## Overview

Add a "Function Comment Style" requirement to the implement profile and echo it in spec/task templates. Update STANDARDS.md to codify the convention permanently.

## Steps

### Step 1: Add Function Comment Style block to implement profile

**File**: `embed/profiles/implement.md`
**Location**: After the "Behavior Rules" section (end of file, line ~99)
**Action**: Append a new `## Function Comment Style` section containing:
- The business-language definition and litmus test
- Good/Bad examples
- Generated code exclusion
- Reference to existing test comment conventions for test code
- Note about reusing spec verbiage

**Traces to**: FR-001, FR-002, FR-003, FR-004, FR-005, FR-006, FR-007, FR-008

### Step 2: Add Function Comment Style block to spec-template.md

**File**: `.brains/templates/spec-template.md`
**Location**: Inside the Testing Requirements HTML comment block, after the existing "Test Comment Style" block (after line 138, before "If this feature requires NO tests")
**Action**: Add a parallel "Function Comment Style" block using the same format

**Traces to**: FR-006

### Step 3: Add Function Comment Style block to tasks-template.md

**File**: `.brains/templates/tasks-template.md`
**Location**: After the existing "Test Comment Style" section (after line 266)
**Action**: Add a parallel "### Function Comment Style" section using the same format

**Traces to**: FR-006

### Step 4: Update STANDARDS.md Documentation section

**File**: `STANDARDS.md`
**Location**: In the Documentation section (lines 303-318), after the existing doc comment rules
**Action**: Add a subsection on business-language framing with the litmus test and examples

**Traces to**: FR-002

## Dependencies

Steps 1-4 are independent and can be done in parallel.

## Canonical Text Block

All insertions share this core content (adapted to each file's formatting conventions):

```
Function Comment Style:
- Doc comments on functions and methods MUST describe the outcome from the
  caller's perspective — not how it works internally.
- Litmus test: would the comment still be true after a complete reimplementation
  of the function's internals? If yes, it's business-language. If not, it's
  a technical description.
- Good:  "// ImportMessages brings external conversation history into the system."
- Good:  "// ResolveConflict picks the most recent version when two edits collide."
- Bad:   "// ImportMessages iterates over the input slice and calls db.Insert for each."
- Bad:   "// ResolveConflict compares timestamps and returns the newer struct."
- Reuse spec language where it fits naturally.
- Interface method declarations follow the same rule — describe the contract.
- Generated code (_gen.go, .pb.go, _string.go, "Code generated ... DO NOT EDIT")
  is excluded.
- Test code follows its own existing comment conventions (see "Test Comment Style").
- "Creates or updates" means material changes to a function signature or body.
  Trivial edits (typo fixes, whitespace) do not trigger this requirement.
```
