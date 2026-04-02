---
name: init-spec-creator
description: Creates comprehensive business specifications by auditing codebases. Produces holistic documentation focused on WHAT the system does and WHY, never HOW it's implemented.
type: skill
---

# Business Spec Generator

Audits codebases to produce **business-focused specifications**. Designed for project managers, stakeholders, and product owners who need to understand system capabilities without technical implementation details.

## Critical: Business Language Only

**NEVER include:**
- API routes, endpoints, HTTP methods, status codes
- Database schemas, tables, field names, foreign keys
- Programming languages, frameworks, libraries, packages
- File paths, directory structures, class names
- Request/response formats, JSON structures, data types
- Technical architecture diagrams, system design
- Code snippets, pseudocode, algorithms

**ALWAYS translate to business language:**
- `GET /api/users/{id}` -> "Users can view profile information"
- `users table with role_id FK` -> "Users have assigned roles that control access"
- `React component with useState` -> "Interactive form that remembers selections"

## Templates

Templates are installed at `~/.brains/templates/init-spec-creator/`. Use them as the structural basis for each output file:

| Template | Output File | Purpose |
|----------|-------------|---------|
| `README-TEMPLATE.md` | `README.md` | Executive overview |
| `INVENTORY-TEMPLATE.md` | `inventory.md` | Capability inventory |
| `DOMAIN-TEMPLATE.md` | `01-domain-name.md` (etc.) | Per-domain specification |

## Output Structure

Generate these artifacts in order in a **flat folder structure** (no subfolders):

1. **README.md** - Executive overview
2. **inventory.md** - Capability inventory
3. **Domain specs** - One per business domain, prefixed with number (e.g., `01-domain-name.md`)

**Important**: All files go in the same directory. Do NOT create a `domains/` subfolder.

## Workflow

### Phase 1: Discovery

Audit the codebase to understand scope:

**For Backend Systems:**
- Identify all exposed capabilities (what can users/systems do?)
- Map data entities to business concepts
- Find business rules and constraints
- Identify user types and access patterns
- Discover integration points with external systems

**For Frontend Systems:**
- Identify all user-facing pages/views
- Map user workflows and journeys
- Find form validations as business rules
- Identify user roles and permission gates
- Discover navigation patterns and user flows

**Questions to answer:**
- Who are the users of this system?
- What can each user type accomplish?
- What business problems does this solve?
- How do different parts of the system relate?
- What are the key business rules and constraints?

### Phase 2: Domain Identification

Group capabilities into **business domains** (not technical modules).

Good domain names (business-focused):
- "Order Management"
- "Customer Onboarding"
- "Payment Processing"
- "Inventory Tracking"
- "User Access Control"

Bad domain names (technically-focused):
- "API Controllers"
- "Database Operations"
- "Redux Store"
- "Authentication Middleware"

Aim for 5-15 domains. If fewer than 5, system may be simple enough for single overview. If more than 15, look for consolidation opportunities.

### Phase 3: Specification Writing

For each artifact, follow the referenced template exactly.

**Writing principles:**
1. Write for someone who has never seen code
2. Describe what users CAN DO, not what systems DO
3. Use active voice: "Users can..." not "The system provides..."
4. Define business terms, not technical entities
5. Describe errors as user experiences, not system states

### Phase 4: Cross-Reference

Ensure all specifications are consistent:
- Domain names match across README and domain specs
- User types are consistent throughout
- Business rules don't contradict across domains
- Integration points are documented from both sides

## Backend vs Frontend Considerations

### Backend Specifications Focus On:
- External capabilities (what can consumers do?)
- Data lifecycle (create, read, update, delete from business view)
- Business rules and validations
- Access control and authorization
- Integration with external systems
- Async operations and their business triggers

### Frontend Specifications Focus On:
- User journeys and task completion
- Page/view purposes and relationships
- Form workflows and user decisions
- Error handling from user perspective
- Navigation and information architecture
- Responsive behavior as user experience

## Quality Checklist

Before completing any specification, verify:

| Check | |
|-------|---|
| Could a non-technical PM read and understand this? | |
| Are all technical terms translated to business language? | |
| Does it describe user capabilities, not system functions? | |
| Are business rules stated as constraints users experience? | |
| Are errors described as user-facing situations? | |
| Is the inventory complete but free of technical details? | |

## Example Translations

| Found in Code | Write in Spec |
|---------------|---------------|
| `POST /v1/{customer_id}/booking/create` | "Customers can submit new booking requests" |
| `role === 'admin' && user.customer_id === resource.customer_id` | "Administrators can only access resources within their own organization" |
| `if (pattern.deleted_at) throw NotFound` | "Deleted patterns are no longer accessible" |
| `<Form onSubmit={...} validation={schema}>` | "Users complete a guided form with required fields" |
| `useEffect(() => fetchData(), [id])` | "Information updates automatically when selections change" |
| `try { ... } catch (e) { showError(e.message) }` | "Users see clear feedback when operations cannot complete" |
