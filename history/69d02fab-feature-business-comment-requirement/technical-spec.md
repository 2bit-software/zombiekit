# Technical Spec: Business-Requirement Function Comments

## Design

This is a template/documentation-only change. No Go code is modified. The canonical text block from the implementation plan is inserted into four files with formatting adapted to each file's conventions.

## File-Specific Details

### embed/profiles/implement.md

Append a new top-level section after "Behavior Rules":

```markdown
## Function Comment Style

When creating or materially modifying functions and methods, every non-test function MUST have a doc comment that describes the outcome from the caller's perspective — not how it works internally.

**Litmus test**: Would the comment still be true after a complete reimplementation of the function's internals? If yes, it's business-language. If not, it's a technical description.

- Good: `// ImportMessages brings external conversation history into the system.`
- Good: `// ResolveConflict picks the most recent version when two edits collide.`
- Bad: `// ImportMessages iterates over the input slice and calls db.Insert for each.`
- Bad: `// ResolveConflict compares timestamps and returns the newer struct.`

**Rules:**
- Reuse spec language where it fits naturally
- Interface method declarations follow the same rule — describe the contract
- Generated code (`_gen.go`, `.pb.go`, `_string.go`, or files with `// Code generated ... DO NOT EDIT.`) is excluded
- Test code follows its own existing comment conventions (see "Test Comment Style" in spec/task templates)
- "Creates or updates" means material changes to a function signature or body — trivial edits (typo fixes, whitespace) do not trigger this requirement
```

### .brains/templates/spec-template.md

Inside the HTML comment block, after "Test Comment Style" (line 138), before "If this feature requires NO tests":

```
  Function Comment Style:
  - Doc comments on functions and methods MUST describe the outcome from the
    caller's perspective — NOT technical rephrasings of the implementation.
  - Litmus test: would the comment still be true after a complete reimplementation?
  - Good:  "// ImportMessages brings external conversation history into the system."
  - Good:  "// ResolveConflict picks the most recent version when two edits collide."
  - Bad:   "// ImportMessages iterates over the input slice and calls db.Insert for each."
  - Bad:   "// ResolveConflict compares timestamps and returns the newer struct."
  - Reuse spec language where it fits naturally.
  - Generated code and test code are excluded (test code has its own rules above).
```

### .brains/templates/tasks-template.md

After the "Test Comment Style" section (line 266), add a new section:

```markdown
### Function Comment Style

When creating or modifying functions and methods, doc comments **MUST** describe
the outcome from the caller's perspective — not how it works internally. Litmus
test: would the comment still be true after a complete reimplementation?

- Good: `// ImportMessages brings external conversation history into the system.`
- Good: `// ResolveConflict picks the most recent version when two edits collide.`
- Bad: `// ImportMessages iterates over the input slice and calls db.Insert for each.`
- Bad: `// ResolveConflict compares timestamps and returns the newer struct.`

Reuse spec language where it fits naturally. Generated code and test code are
excluded (test code has its own rules above).
```

### STANDARDS.md

After the existing doc comment examples (line 318), before the `---` separator:

```markdown
### Business-Language Framing

Doc comments must describe the **outcome** from the caller's perspective, not the internal mechanism. Litmus test: would the comment still be true after a complete reimplementation?

```go
// Good — describes the outcome
// ImportMessages brings external conversation history into the system.
func ImportMessages(src io.Reader) error { ... }

// Bad — describes the implementation
// ImportMessages iterates over the input and calls db.Insert for each record.
func ImportMessages(src io.Reader) error { ... }
```

This applies to all functions, methods, and interface method declarations. Generated code is excluded.
```
