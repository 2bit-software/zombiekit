# README-TEMPLATE.md

Use this template for the main README.md that provides executive overview of the system.

---

# [SYSTEM NAME] - Business Specification

## Overview

[1-2 paragraphs describing what this system does in plain business terms. Answer: What business problem does this solve? Who uses it? What value does it provide?]

**Business Domains**: [NUMBER]

---

## Executive Summary

[SYSTEM NAME] enables [PRIMARY USER TYPES] to [PRIMARY CAPABILITIES]. The system handles:

1. **[Domain 1 Name]** - [One-line description of business value]
2. **[Domain 2 Name]** - [One-line description of business value]
3. **[Domain 3 Name]** - [One-line description of business value]
[Continue for all domains...]

---

## Business Domains

### [01. Domain Name](./01-domain-name.md)
[2-3 sentences describing what users can accomplish in this domain. List key capabilities as bullet points.]
- Capability 1
- Capability 2
- Capability 3

### [02. Domain Name](./02-domain-name.md)
[2-3 sentences describing what users can accomplish in this domain.]
- Capability 1
- Capability 2

[Continue for all domains...]

---

## Core Concepts

Define business terms that appear throughout the specification. These are NOT database entities—they are concepts that business users need to understand.

### [Concept 1 Name]
[Plain language definition. What is this thing from a business perspective? Why does it matter?]

### [Concept 2 Name]
[Plain language definition. How does this relate to other concepts?]

[Continue for key concepts...]

---

## Key Workflows

Describe the primary ways users interact with the system. These are end-to-end journeys, not individual operations.

### [Workflow 1 Name]
```
1. [User starts by...]
2. [Then they...]
3. [System responds with...]
4. [User completes by...]
```

### [Workflow 2 Name]
```
1. [Step...]
2. [Step...]
```

[Include 3-5 primary workflows]

---

## User Types

Who uses this system and what can they do?

### [User Type 1]
- **Description**: [Who is this person? What's their role?]
- **Primary Activities**: [What do they mainly do in the system?]
- **Access Level**: [What can they see and do?]

### [User Type 2]
- **Description**: [Who is this person?]
- **Primary Activities**: [What do they do?]
- **Access Level**: [What can they access?]

[Continue for all user types...]

---

## Access Model

[Describe how access control works from a business perspective. NOT technical implementation.]

- [How are users organized? (e.g., by organization, by role, by team)]
- [What determines what a user can see?]
- [What determines what a user can do?]
- [How is data separated between different users/organizations?]

---

## Files in This Specification

```
[system-name]-spec/
├── README.md                    # This file
├── inventory.md                 # Complete capability inventory
├── 01-[domain-name].md          # Domain specifications
├── 02-[domain-name].md
├── 03-[domain-name].md
└── ...
```

Note: All files are in a flat structure (no subfolders) for compatibility with systems that don't support nested directories.

---

## Version Information

- **Generated**: [DATE]
- **Source**: [What was audited to create this spec]
