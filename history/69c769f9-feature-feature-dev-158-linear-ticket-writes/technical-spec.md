# Technical Spec: DEV-158 Linear Ticket Writes

## File Changes

### `internal/linear/client.go`

**Ticket struct** — add `TeamID`:
```go
type Ticket struct {
	ID          string
	Identifier  string
	Title       string
	Description string
	Status      string
	Labels      []string
	Priority    int
	URL         string
	TeamID      string  // NEW
}
```

**CreateTicketInput** — add `ProjectID`:
```go
type CreateTicketInput struct {
	TeamID      string
	Title       string
	Description string
	StateID     string
	LabelIDs    []string
	Priority    *int
	AssigneeID  string
	ProjectID   string  // NEW
}
```

### `internal/linear/http_client.go`

#### issueNode changes

Add team field to `issueNode`:
```go
type issueNode struct {
	// ... existing fields ...
	Team struct {
		ID string `json:"id"`
	} `json:"team"`
}
```

Update `toTicket()`:
```go
func (n issueNode) toTicket() Ticket {
	// ... existing code ...
	return Ticket{
		// ... existing fields ...
		TeamID: n.Team.ID,  // NEW
	}
}
```

#### Query updates

Add `team { id }` to both `getTicketQuery` and `pollReadyTicketsQuery` inside the issue node selection.

#### New GraphQL constants

```go
const resolveWorkflowStateQuery = `
query($teamId: String!, $name: String!) {
  workflowStates(
    filter: {
      team: { id: { eq: $teamId } }
      name: { eq: $name }
    }
  ) {
    nodes {
      id
      name
    }
  }
}`

const resolveLabelQuery = `
query($name: String!) {
  issueLabels(
    filter: {
      name: { eq: $name }
    }
  ) {
    nodes {
      id
      name
    }
  }
}`

const issueUpdateMutation = `
mutation($id: String!, $input: IssueUpdateInput!) {
  issueUpdate(id: $id, input: $input) {
    success
  }
}`

const issueCreateMutation = `
mutation($input: IssueCreateInput!) {
  issueCreate(input: $input) {
    success
    issue {
      id
      identifier
      title
      description
      url
      priority
      state { name }
      labels { nodes { name } }
      team { id }
    }
  }
}`
```

#### New response types

```go
type workflowStatesResponse struct {
	WorkflowStates struct {
		Nodes []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"workflowStates"`
}

type issueLabelsResponse struct {
	IssueLabels struct {
		Nodes []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"issueLabels"`
}

type issueUpdateResponse struct {
	IssueUpdate struct {
		Success bool `json:"success"`
	} `json:"issueUpdate"`
}

type issueCreateResponse struct {
	IssueCreate struct {
		Success bool      `json:"success"`
		Issue   issueNode `json:"issue"`
	} `json:"issueCreate"`
}
```

#### New internal methods

```go
func (c *httpClient) resolveWorkflowStateID(ctx context.Context, teamID, name string) (string, error)
func (c *httpClient) resolveLabelID(ctx context.Context, name string) (string, error)
```

#### Method implementations

**`SetTicketStatus`**: GetTicket → resolveWorkflowStateID → issueUpdate
**`ApplyLabel`**: resolveLabelID → issueUpdate with addedLabelIds
**`RemoveLabel`**: resolveLabelID → issueUpdate with removedLabelIds
**`CreateTicket`**: Build variables map from CreateTicketInput → issueCreate → parse issue from response

#### issueUpdate input variable construction

For `SetTicketStatus`:
```go
vars := map[string]any{
	"id":    id,
	"input": map[string]any{"stateId": stateID},
}
```

For `ApplyLabel`:
```go
vars := map[string]any{
	"id":    id,
	"input": map[string]any{"addedLabelIds": []string{labelID}},
}
```

For `RemoveLabel`:
```go
vars := map[string]any{
	"id":    id,
	"input": map[string]any{"removedLabelIds": []string{labelID}},
}
```

#### issueCreate input variable construction

```go
input := map[string]any{
	"teamId": inp.TeamID,
}
if inp.Title != "" {
	input["title"] = inp.Title
}
if inp.Description != "" {
	input["description"] = inp.Description
}
if inp.StateID != "" {
	input["stateId"] = inp.StateID
}
if len(inp.LabelIDs) > 0 {
	input["labelIds"] = inp.LabelIDs
}
if inp.ProjectID != "" {
	input["projectId"] = inp.ProjectID
}
if inp.Priority != nil {
	input["priority"] = *inp.Priority
}
if inp.AssigneeID != "" {
	input["assigneeId"] = inp.AssigneeID
}
vars := map[string]any{"input": input}
```

### `internal/linear/mock.go`

No changes needed — mock already has function fields for all 4 methods.

### `internal/linear/http_client_test.go`

**New test helper**: Multi-query handler that dispatches based on query content in the request body:

```go
func queryDispatcher(handlers map[string]http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(body))
		for pattern, handler := range handlers {
			if strings.Contains(string(body), pattern) {
				handler(w, r)
				return
			}
		}
		http.Error(w, "unmatched query", 500)
	}
}
```

This enables multi-step tests (e.g., SetTicketStatus which does issue fetch → state resolve → mutation) with a single httptest.Server.

## Error Messages

| Scenario | Error |
|----------|-------|
| Status not found | `"linear: workflow state %q not found for team %s"` |
| Label not found | `"linear: label %q not found"` |
| Label ambiguous | `"linear: label %q is ambiguous (%d matches)"` |
| Mutation failed | `"linear: issueUpdate failed (success=false)"` |
| Create failed | `"linear: issueCreate failed (success=false)"` |

## Spec Traceability

| FR | Implementation |
|----|----------------|
| FR-001 | `SetTicketStatus` method |
| FR-002 | `resolveWorkflowStateID` error path |
| FR-003 | `ApplyLabel` method |
| FR-004 | `RemoveLabel` method |
| FR-005 | `resolveLabelID` error paths (not found + ambiguous) |
| FR-006 | `CreateTicket` method |
| FR-007 | Error propagation in `CreateTicket` |
| FR-008 | `TeamID` on Ticket, `team { id }` in queries |
| FR-009 | `ProjectID` on CreateTicketInput |
