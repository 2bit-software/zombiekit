# DOMAIN-TEMPLATE.md

Use this template for individual domain specifications (e.g., 01-order-management.md).

---

# [Domain Name] - Business Specification

## Overview

[2-3 sentences describing what this domain enables. What business problem does it solve? Who benefits?]

## Capabilities Summary

| Capability | Description | Priority |
|------------|-------------|----------|
| [User action 1] | [What user accomplishes] | Core |
| [User action 2] | [What user accomplishes] | Core |
| [User action 3] | [What user accomplishes] | Supporting |

---

## 1. Business Purpose

### Problem Solved
[What business problem does this domain address? Why does it exist?]

### Who Benefits
- **[User Type 1]**: [How they benefit]
- **[User Type 2]**: [How they benefit]

### Business Value
[Why is this domain important to the organization? What happens without it?]

---

## 2. User Journey

### Typical Workflow

```
[Step-by-step description of how users interact with this domain]

1. User starts by [action]
2. System presents [information/options]
3. User selects/enters [data]
4. System confirms [result]
5. User can then [next action]
```

### Common Scenarios

| Scenario | Description | Outcome |
|----------|-------------|---------|
| [Scenario 1] | [When user wants to...] | [They can...] |
| [Scenario 2] | [When user needs to...] | [They can...] |

---

## 3. What Users Provide

[Describe information users must provide to use this domain. NOT field names or data types—describe in business terms.]

### For [Capability 1]
- [Information type]: [What it is and why it's needed]
- [Information type]: [Description]

### For [Capability 2]
- [Information type]: [Description]

---

## 4. What Users Receive

[Describe what users get back from the system. NOT response formats—describe business outcomes.]

### Confirmations
- [When user does X, they receive confirmation of Y]

### Information
- [Users can view/access...]

### Notifications
- [Users are informed when...]

---

## 5. Business Rules

Rules that govern how this domain operates. State as constraints users experience, not system logic.

| Rule | Description | Why It Exists |
|------|-------------|---------------|
| BR-001 | [Plain language rule] | [Business reason] |
| BR-002 | [Plain language rule] | [Business reason] |
| BR-003 | [Plain language rule] | [Business reason] |

### Access Rules
- [Who can do what in this domain]
- [What determines access]

### Validation Rules
- [What must be true for operations to succeed]
- [What constraints apply to user input]

### Timing Rules
- [Any time-based constraints]
- [When operations are available]

---

## 6. Error Scenarios

What can go wrong and how users experience it.

| Situation | User Experience | Resolution |
|-----------|-----------------|------------|
| [When X happens] | [User sees/experiences...] | [How to fix] |
| [When Y is missing] | [User is informed...] | [What to do] |
| [When Z fails] | [User receives...] | [Next steps] |

### Common Issues
- **[Issue 1]**: [What causes it] → [How user resolves it]
- **[Issue 2]**: [What causes it] → [How user resolves it]

---

## 7. Related Domains

How this domain connects to others.

### Depends On
- **[Domain X]**: [Why this domain needs Domain X]

### Used By
- **[Domain Y]**: [Why Domain Y uses this domain]

### Shares Data With
- **[Domain Z]**: [What information flows between them]

---

## 8. Access Control

Who can do what in this domain.

| User Type | Can View | Can Create | Can Modify | Can Delete |
|-----------|----------|------------|------------|------------|
| [Type 1] | [Yes/No/Limited] | [Yes/No] | [Yes/No] | [Yes/No] |
| [Type 2] | [Yes/No/Limited] | [Yes/No] | [Yes/No] | [Yes/No] |

### Special Permissions
- [Any non-standard access rules]

### Data Visibility
- [What determines what data a user can see]

---

## 9. Lifecycle

How items in this domain progress through states.

```
[State 1] → [State 2] → [State 3] → [Final State]
    ↓           ↓
[Alt State] [Error State]
```

### State Definitions
- **[State 1]**: [What it means, when items are in this state]
- **[State 2]**: [What it means]
- **[Final State]**: [What it means]

### Transitions
- [State 1] → [State 2]: [What causes this transition]
- [State 2] → [State 3]: [What causes this transition]

---

## 10. Success Metrics

How we know this domain is working well.

| Metric | Target | Why It Matters |
|--------|--------|----------------|
| [User-observable metric] | [Target value] | [Business importance] |
| [Completion rate] | [Target] | [Why] |
| [Time to complete] | [Target] | [Why] |

---

## For Frontend Domains

Additional sections when documenting UI/frontend areas:

### Page/View Structure
- **[Page 1]**: [Purpose, what user accomplishes here]
- **[Page 2]**: [Purpose]

### Navigation
- How users move between pages in this domain
- Entry points from other domains
- Exit points to other domains

### Form Workflows
- [Form 1]: [Purpose, key fields in business terms, validation rules as user experiences them]

### Visual Feedback
- [What users see during operations]
- [How success/failure is communicated]

---

## For Backend Domains

Additional sections when documenting API/service areas:

### Integration Points
- **Incoming**: [What other systems send data to this domain]
- **Outgoing**: [What this domain sends to other systems]

### Automation
- [What happens automatically without user action]
- [What triggers automated processes]

### Background Operations
- [Things that happen outside of direct user interaction]
- [Scheduled or event-driven processes]
