# HTTP Routes Contract: Sticky Memory Web Plugin

**Feature**: 009-sticky-memory-plugin
**Date**: 2025-12-22
**Base Path**: `/memory`

## Route Summary

| Method | Path | Handler | Description |
|--------|------|---------|-------------|
| GET | `/memory` | list | List all memories with pagination and search |
| GET | `/memory/new` | createForm | Display create memory form |
| POST | `/memory` | create | Create new memory |
| GET | `/memory/{name}` | view | View memory content |
| GET | `/memory/{name}/edit` | editForm | Display edit memory form |
| PUT | `/memory/{name}` | update | Update existing memory |
| GET | `/memory/{name}/delete` | deleteConfirm | Display delete confirmation |
| DELETE | `/memory/{name}` | delete | Delete memory |

## Route Specifications

### GET /memory - List Memories

**Description**: Display paginated list of all memories, optionally filtered by search.

**Query Parameters**:
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| page | int | 1 | Page number (1-indexed) |
| limit | int | 20 | Items per page (10, 20, 50, 100) |
| q | string | "" | Search query (filters name and content) |

**Response** (HTML):
- Full page: Renders `shell.html` wrapping `memory/list.html`
- HTMX: Renders `memory/list.html` only

**Template Data**: `ListData`
```go
{
    Memories:   []MemoryMetadata,
    Pagination: PaginationData,
    Query:      string,
    Error:      string,
}
```

**Behavior**:
- Empty list shows empty state with "Create first memory" prompt
- Invalid page/limit defaults to 1/20
- Search is case-insensitive

---

### GET /memory/new - Create Form

**Description**: Display form to create a new memory.

**Response** (HTML):
- Renders `memory/form.html` with empty form

**Template Data**: `FormData`
```go
{
    Name:    "",
    Content: "",
    Error:   "",
    IsEdit:  false,
}
```

---

### POST /memory - Create Memory

**Description**: Create a new memory from form submission.

**Request Body** (form-urlencoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Memory name (sanitized) |
| content | string | No | Memory content (max 1MB) |

**Response** (HTML):
- Success: Redirect to `/memory` (list) with HTMX
- Validation error: Re-render `memory/form.html` with error

**Error Cases**:
- Empty name: "Name is required"
- Name too long: "Name must be 255 characters or less"
- Content too large: "Content must be 1MB or less"

**Note**: If memory with same name exists, it's updated (new version created).

---

### GET /memory/{name} - View Memory

**Description**: Display full memory content with metadata.

**Path Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| name | string | Memory name (URL-encoded) |

**Response** (HTML):
- Found: Renders `memory/view.html`
- Not found: Renders `memory/view.html` with error

**Template Data**: `ViewData`
```go
{
    Memory:        *MemoryItem,
    FormattedSize: string,
    Error:         string,
}
```

**Behavior**:
- Default view: Rendered markdown (client-side via marked.js)
- Toggle available to view source

---

### GET /memory/{name}/edit - Edit Form

**Description**: Display form to edit existing memory.

**Path Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| name | string | Memory name (URL-encoded) |

**Response** (HTML):
- Found: Renders `memory/form.html` pre-populated
- Not found: Renders `memory/form.html` with error

**Template Data**: `FormData`
```go
{
    Name:    memory.Name,
    Content: memory.Content,
    Error:   "",
    IsEdit:  true,
}
```

---

### PUT /memory/{name} - Update Memory

**Description**: Update existing memory content.

**Path Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| name | string | Memory name (URL-encoded) |

**Request Body** (form-urlencoded):
| Field | Type | Required | Description |
|-------|------|----------|-------------|
| content | string | No | Updated content (max 1MB) |

**Note**: Name cannot be changed via edit. To rename, delete and recreate.

**Response** (HTML):
- Success: Redirect to `/memory/{name}` (view)
- Validation error: Re-render `memory/form.html` with error

**Behavior**:
- Creates new version (version++)
- Updates `updated_at` timestamp

---

### GET /memory/{name}/delete - Delete Confirmation

**Description**: Display delete confirmation page.

**Path Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| name | string | Memory name (URL-encoded) |

**Response** (HTML):
- Found: Renders `memory/delete.html`
- Not found: Redirect to `/memory` with error

**Template Data**: `DeleteData`
```go
{
    Memory: *MemoryMetadata,
    Error:  "",
}
```

---

### DELETE /memory/{name} - Delete Memory

**Description**: Permanently delete (soft) a memory.

**Path Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| name | string | Memory name (URL-encoded) |

**Response** (HTML):
- Success: Redirect to `/memory` (list)
- Error: Re-render `memory/delete.html` with error

**Behavior**:
- Soft delete (sets Deleted=true in storage)
- Memory no longer appears in list or search

## HTMX Integration

All routes support dual-mode rendering:

**Detection**: `HX-Request: true` header

**Full Page Load** (no HX-Request):
- Renders complete HTML with shell (sidebar, layout)
- Browser URL matches route

**HTMX Request** (HX-Request: true):
- Renders content template only
- Response swaps into `#content` div
- `hx-push-url="true"` updates browser history

## Link Patterns

Standard link pattern for all navigation:
```html
<a href="/memory/{path}"
   hx-get="/memory/{path}"
   hx-target="#content"
   hx-push-url="true">
```

Form submission pattern:
```html
<form hx-post="/memory"
      hx-target="#content"
      hx-push-url="true">
```

Update pattern (PUT via HTMX):
```html
<form hx-put="/memory/{name}"
      hx-target="#content"
      hx-push-url="true">
```

Delete pattern:
```html
<button hx-delete="/memory/{name}"
        hx-target="#content"
        hx-push-url="true"
        hx-confirm="Delete this memory?">
```
