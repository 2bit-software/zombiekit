---
name: spec-updater
description: Updates existing business specifications to reflect recent git commits. Reads the commit diff, compares against current spec files, and surgically updates only affected sections.
type: skill
---

# Spec Updater

Updates existing business specifications to reflect what actually changed in recent commits. This is a surgical tool — it reads the diff, identifies what shifted in business terms, and patches only the affected spec sections. It does not regenerate from scratch.

## Prerequisites

Before starting, verify two things:

1. **Spec files exist.** Look for a spec directory (commonly `*-spec/` or `spec/`) containing `README.md`, `inventory.md`, and numbered domain files (`01-*.md`, etc.). If no specs exist, stop and tell the user to run `init-spec-creator` first.

2. **There are commits to analyze.** Default scope is the last commit (`HEAD~1..HEAD`). The user can override this with a range (e.g., "last 5 commits", "since Tuesday", a specific SHA range). Convert whatever they say into a valid `git log` / `git diff` range.

## Workflow

### Phase 1: Understand the Changes

Run `git log` and `git diff` for the commit range to understand what changed. Focus on:

- New files or deleted files (may signal new/removed capabilities)
- Modified business logic (routes, controllers, services, components, models)
- Changed validations, permissions, or access control
- New or removed integrations
- UI changes (new pages, modified forms, changed navigation)

Ignore changes that don't affect business capabilities:
- Dependency updates (unless they add/remove functionality)
- Refactors that preserve behavior
- Test-only changes
- CI/CD configuration
- Code style / formatting

### Phase 2: Map Changes to Spec Impact

For each meaningful change, determine:

1. **Which domain(s) does it affect?** Read the existing domain spec files to find where this capability lives.
2. **What kind of change is it?**
   - **New capability** — something users can now do that they couldn't before
   - **Modified capability** — existing behavior changed (new fields, different rules, altered workflow)
   - **Removed capability** — something users could do that no longer exists
   - **New business rule** — a constraint that didn't exist before
   - **Changed business rule** — an existing constraint was modified or removed
   - **Access control change** — who can do what shifted

3. **Does it fit an existing domain?** If not, ask the user whether to:
   - Create a new domain spec (suggest a business-focused name)
   - Fold it into an existing domain (suggest which one and why)
4. **Was an entire domain removed?** If all capabilities in a domain were deleted, remove the domain spec file, remove it from README.md and inventory.md, and renumber remaining domain files to keep the sequence contiguous.

### Phase 3: Update Specs

Apply changes to the affected files only. Follow the same conventions as the existing specs — match their voice, structure, and level of detail.

**For each affected domain spec:**
- Update only the sections that changed (Capabilities Summary, Business Rules, User Journey, etc.)
- Preserve all unchanged sections exactly as they are
- Follow the structure in [DOMAIN-TEMPLATE.md](DOMAIN-TEMPLATE.md) for any new sections

**For inventory.md:**
- Add/remove/modify capabilities that changed
- Update the summary statistics count
- Keep capability naming in user-action terms ("View order status", not "GET /orders/:id")

**For README.md:**
- Update domain descriptions if capabilities shifted significantly
- Update Core Concepts if new business terms emerged
- Update Key Workflows if user journeys changed
- Update User Types if access patterns changed
- Do NOT touch the Version Information section — avoid metadata that creates commit loops

**If creating a new domain spec:**
- Follow the numbering convention (next available number)
- Use [DOMAIN-TEMPLATE.md](DOMAIN-TEMPLATE.md) as the template
- Add it to the README.md domain listing and inventory.md

### Phase 4: Consistency Check

After making changes, verify:
- Domain names still match across README and domain specs
- Capability counts in inventory.md are accurate
- Cross-references between domains are still valid (Related Domains sections)
- No contradictions between updated and untouched sections

## Critical: Business Language Only

The same rules from init-spec-creator apply — specs describe WHAT users can do, never HOW the system implements it.

**NEVER include:**
- API routes, endpoints, HTTP methods, status codes
- Database schemas, tables, field names
- Programming languages, frameworks, libraries
- File paths, directory structures, class names
- Code snippets, request/response formats

**ALWAYS translate to business language:**
- `Added POST /api/invoices` -> "Users can now create invoices"
- `Removed admin_override flag` -> "The administrative bypass for this rule has been removed"
- `Added React DatePicker to booking form` -> "Users can now select dates from a calendar when booking"

## Output

When done, provide a brief summary of what changed:

```
Updated specs:
- 03-order-management.md: Added "Export order history" capability, updated business rules for bulk orders
- inventory.md: +2 capabilities in Order Management
- README.md: Updated Order Management description

No changes needed:
- 01-user-access.md, 02-payment-processing.md (unaffected by these commits)
```

This gives the user a clear picture of what was touched and why, without them needing to diff the spec files themselves.
