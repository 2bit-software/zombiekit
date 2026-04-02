# INVENTORY-TEMPLATE.md

Use this template for inventory.md that lists all system capabilities.

---

# [SYSTEM NAME] - Capability Inventory

This document provides a comprehensive list of all capabilities in [SYSTEM NAME].

## Summary Statistics

- **Total Capabilities**: [NUMBER]
- **Breakdown by Domain**:
  - [Domain 1]: [X] capabilities
  - [Domain 2]: [X] capabilities
  - [Domain 3]: [X] capabilities
  [Continue for all domains...]

---

## DOMAIN: [Domain 1 Name]

**Purpose**: [One sentence describing what this domain enables users to do]

### [Sub-category if needed]

| # | Capability | Description | Who Can Use |
|---|---|---|---|
| 1 | [Action name in user terms] | [What the user accomplishes] | [User type(s)] |
| 2 | [Action name] | [Description] | [User type(s)] |
| 3 | [Action name] | [Description] | [User type(s)] |

### [Another sub-category]

| # | Capability | Description | Who Can Use |
|---|---|---|---|
| 1 | [Action] | [Description] | [User type(s)] |

---

## DOMAIN: [Domain 2 Name]

**Purpose**: [One sentence description]

| # | Capability | Description | Who Can Use |
|---|---|---|---|
| 1 | [Action] | [Description] | [User type(s)] |

[Continue for all domains...]

---

## Domain Groupings for Business Specification

The capabilities can be logically grouped into these business domains:

1. **[Domain 1 Name]** - [Brief description of business area]
2. **[Domain 2 Name]** - [Brief description]
3. **[Domain 3 Name]** - [Brief description]
[Continue...]

---

## Capability Naming Guidelines

When listing capabilities, use action-oriented language from the user's perspective:

**Good capability names:**
- "View order status"
- "Submit new booking request"
- "Export monthly report"
- "Approve pending items"
- "Search transaction history"

**Avoid technical names:**
- ~~"GET order by ID"~~
- ~~"POST booking endpoint"~~
- ~~"Query transactions table"~~
- ~~"Execute approval workflow"~~

---

## For Frontend Systems

When documenting frontend capabilities, organize by user task:

| # | Page/View | User Task | Available Actions |
|---|---|---|---|
| 1 | [Page name] | [What user is trying to accomplish] | [List of things user can do here] |
| 2 | Dashboard | Monitor daily activity | View summary, filter by date, export data |

---

## For Backend Systems

When documenting backend capabilities, organize by business operation:

| # | Operation | Business Purpose | Who Triggers It |
|---|---|---|---|
| 1 | [Operation name] | [Why this exists] | [User/System] |
| 2 | Process payment | Complete customer purchase | Customer checkout |
