package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEndpoint = "https://api.linear.app/graphql"
	defaultTimeout  = 30 * time.Second
	maxRetries      = 3
	retryBaseDelay  = 1 * time.Second
	retryMultiplier = 2.0
	maxJitter       = 500 * time.Millisecond
)

// Option configures an httpClient.
type Option func(*httpClient)

// WithEndpoint overrides the default Linear API endpoint.
func WithEndpoint(url string) Option {
	return func(c *httpClient) { c.endpoint = url }
}

// WithHTTPClient overrides the default HTTP client.
func WithHTTPClient(hc *http.Client) Option {
	return func(c *httpClient) { c.httpClient = hc }
}

type httpClient struct {
	apiKey         string
	endpoint       string
	httpClient     *http.Client
	lastHeaders    http.Header
	retryBase      time.Duration
	retryMaxJitter time.Duration
}

// WithRetryTiming overrides retry backoff timing (for tests).
func WithRetryTiming(base, maxJitter time.Duration) Option {
	return func(c *httpClient) {
		c.retryBase = base
		c.retryMaxJitter = maxJitter
	}
}

// NewClient creates a Linear API client with the given API key.
func NewClient(apiKey string, opts ...Option) (*httpClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("linear: API key must not be empty")
	}
	c := &httpClient{
		apiKey:   apiKey,
		endpoint: defaultEndpoint,
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		retryBase:      retryBaseDelay,
		retryMaxJitter: maxJitter,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c, nil
}

// NewClientFromEnv creates a Linear API client using the BRAINS_LINEAR_API_KEY environment variable.
func NewClientFromEnv(opts ...Option) (*httpClient, error) {
	key := os.Getenv("BRAINS_LINEAR_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("linear: BRAINS_LINEAR_API_KEY environment variable not set")
	}
	return NewClient(key, opts...)
}

type graphqlRequest struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables,omitempty"`
}

type graphqlResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []graphqlError  `json:"errors"`
}

type graphqlError struct {
	Message    string `json:"message"`
	Extensions struct {
		Code string `json:"code"`
	} `json:"extensions"`
}

func (c *httpClient) do(ctx context.Context, query string, variables map[string]any, target any) error {
	body, err := json.Marshal(graphqlRequest{Query: query, Variables: variables})
	if err != nil {
		return NewNetworkError(fmt.Sprintf("linear: marshal request: %s", err), err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return NewNetworkError(fmt.Sprintf("linear: create request: %s", err), err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return NewNetworkError("linear: request cancelled", ctx.Err())
		}
		return NewNetworkError(fmt.Sprintf("linear: request failed: %s", err), err)
	}
	defer resp.Body.Close()

	c.lastHeaders = resp.Header

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewNetworkError(fmt.Sprintf("linear: read response: %s", err), err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return NewAPIError("linear: unauthorized (invalid or revoked API key)", nil)
	}

	if resp.StatusCode >= 500 {
		return NewNetworkError(fmt.Sprintf("linear: server error (HTTP %d)", resp.StatusCode), nil)
	}

	var gqlResp graphqlResponse
	if err := json.Unmarshal(respBody, &gqlResp); err != nil {
		return NewNetworkError(fmt.Sprintf("linear: invalid JSON response (HTTP %d)", resp.StatusCode), err)
	}

	if len(gqlResp.Errors) > 0 {
		first := gqlResp.Errors[0]
		if first.Extensions.Code == "RATELIMITED" {
			return NewRateLimitedError(fmt.Sprintf("linear: rate limited: %s", first.Message), nil)
		}
		if strings.Contains(strings.ToLower(first.Message), "not found") {
			return NewNotFoundError(fmt.Sprintf("linear: %s", first.Message), nil)
		}
		return NewAPIError(fmt.Sprintf("linear: API error: %s", first.Message), nil)
	}

	if target != nil && gqlResp.Data != nil {
		if err := json.Unmarshal(gqlResp.Data, target); err != nil {
			return NewNetworkError(fmt.Sprintf("linear: unmarshal response data: %s", err), err)
		}
	}

	return nil
}

func (c *httpClient) doWithRetry(ctx context.Context, query string, variables map[string]any, target any) error {
	var lastErr error
	for attempt := range maxRetries + 1 {
		err := c.do(ctx, query, variables, target)
		if err == nil {
			return nil
		}
		if !IsRateLimited(err) {
			return err
		}
		lastErr = err
		if attempt == maxRetries {
			break
		}

		delay := c.retryDelay(attempt)
		select {
		case <-ctx.Done():
			return NewNetworkError("linear: request cancelled during retry", ctx.Err())
		case <-time.After(delay):
		}
	}
	return lastErr
}

func (c *httpClient) retryDelay(attempt int) time.Duration {
	if resetStr := c.lastHeaders.Get("X-RateLimit-Requests-Reset"); resetStr != "" {
		if resetMs, err := strconv.ParseInt(resetStr, 10, 64); err == nil {
			resetTime := time.UnixMilli(resetMs)
			if delay := time.Until(resetTime); delay > 0 {
				return delay
			}
		}
	}
	base := float64(c.retryBase) * math.Pow(retryMultiplier, float64(attempt))
	jitter := time.Duration(rand.Int64N(int64(c.retryMaxJitter) + 1))
	return time.Duration(base) + jitter
}

// GraphQL queries

const pollReadyTicketsQuery = `
query($label: String!, $projectId: ID!, $after: String) {
  issues(
    filter: {
      labels: { name: { eq: $label } }
      project: { id: { eq: $projectId } }
      description: { null: false }
    }
    first: 50
    after: $after
  ) {
    nodes {
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
    pageInfo {
      hasNextPage
      endCursor
    }
  }
}`

const getTicketQuery = `
query($id: String!) {
  issue(id: $id) {
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
}`

// Mutation and resolution queries

const resolveWorkflowStateQuery = `
query($teamId: ID!, $name: String!) {
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

const commentCreateMutation = `
mutation($input: CommentCreateInput!) {
  commentCreate(input: $input) {
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

// Response types for JSON unmarshaling

type issuesResponse struct {
	Issues struct {
		Nodes    []issueNode `json:"nodes"`
		PageInfo struct {
			HasNextPage bool    `json:"hasNextPage"`
			EndCursor   *string `json:"endCursor"`
		} `json:"pageInfo"`
	} `json:"issues"`
}

type issueResponse struct {
	Issue issueNode `json:"issue"`
}

type issueNode struct {
	ID          string  `json:"id"`
	Identifier  string  `json:"identifier"`
	Title       string  `json:"title"`
	Description *string `json:"description"`
	URL         string  `json:"url"`
	Priority    float64 `json:"priority"`
	State       struct {
		Name string `json:"name"`
	} `json:"state"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Team struct {
		ID string `json:"id"`
	} `json:"team"`
}

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

type commentCreateResponse struct {
	CommentCreate struct {
		Success bool `json:"success"`
	} `json:"commentCreate"`
}

type issueCreateResponse struct {
	IssueCreate struct {
		Success bool      `json:"success"`
		Issue   issueNode `json:"issue"`
	} `json:"issueCreate"`
}

func (n issueNode) toTicket() Ticket {
	labels := make([]string, len(n.Labels.Nodes))
	for i, l := range n.Labels.Nodes {
		labels[i] = l.Name
	}
	desc := ""
	if n.Description != nil {
		desc = *n.Description
	}
	return Ticket{
		ID:          n.ID,
		Identifier:  n.Identifier,
		Title:       n.Title,
		Description: desc,
		Status:      n.State.Name,
		Labels:      labels,
		Priority:    int(n.Priority),
		URL:         n.URL,
		TeamID:      n.Team.ID,
	}
}

func (c *httpClient) PollReadyTickets(ctx context.Context, label string, projectID string) ([]Ticket, error) {
	var tickets []Ticket
	var cursor *string

	for {
		vars := map[string]any{"label": label, "projectId": projectID}
		if cursor != nil {
			vars["after"] = *cursor
		}

		var resp issuesResponse
		if err := c.doWithRetry(ctx, pollReadyTicketsQuery, vars, &resp); err != nil {
			return nil, fmt.Errorf("poll ready tickets: %w", err)
		}

		for _, node := range resp.Issues.Nodes {
			t := node.toTicket()
			if len(t.Description) > 0 {
				tickets = append(tickets, t)
			}
		}

		if !resp.Issues.PageInfo.HasNextPage {
			break
		}
		cursor = resp.Issues.PageInfo.EndCursor
	}

	if tickets == nil {
		tickets = []Ticket{}
	}
	return tickets, nil
}

func (c *httpClient) GetTicket(ctx context.Context, id string) (*Ticket, error) {
	var resp issueResponse
	if err := c.doWithRetry(ctx, getTicketQuery, map[string]any{"id": id}, &resp); err != nil {
		return nil, fmt.Errorf("get ticket: %w", err)
	}
	t := resp.Issue.toTicket()
	return &t, nil
}

func (c *httpClient) resolveWorkflowStateID(ctx context.Context, teamID, name string) (string, error) {
	var resp workflowStatesResponse
	vars := map[string]any{"teamId": teamID, "name": name}
	if err := c.doWithRetry(ctx, resolveWorkflowStateQuery, vars, &resp); err != nil {
		return "", fmt.Errorf("resolve workflow state: %w", err)
	}
	if len(resp.WorkflowStates.Nodes) == 0 {
		return "", NewNotFoundError(fmt.Sprintf("linear: workflow state %q not found for team %s", name, teamID), nil)
	}
	return resp.WorkflowStates.Nodes[0].ID, nil
}

func (c *httpClient) resolveLabelID(ctx context.Context, name string) (string, error) {
	var resp issueLabelsResponse
	vars := map[string]any{"name": name}
	if err := c.doWithRetry(ctx, resolveLabelQuery, vars, &resp); err != nil {
		return "", fmt.Errorf("resolve label: %w", err)
	}
	switch len(resp.IssueLabels.Nodes) {
	case 0:
		return "", NewNotFoundError(fmt.Sprintf("linear: label %q not found", name), nil)
	case 1:
		return resp.IssueLabels.Nodes[0].ID, nil
	default:
		return "", NewAPIError(fmt.Sprintf("linear: label %q is ambiguous (%d matches)", name, len(resp.IssueLabels.Nodes)), nil)
	}
}

func (c *httpClient) SetTicketStatus(ctx context.Context, id string, status string) error {
	ticket, err := c.GetTicket(ctx, id)
	if err != nil {
		return fmt.Errorf("set ticket status: %w", err)
	}

	stateID, err := c.resolveWorkflowStateID(ctx, ticket.TeamID, status)
	if err != nil {
		return fmt.Errorf("set ticket status: %w", err)
	}

	var resp issueUpdateResponse
	vars := map[string]any{
		"id":    id,
		"input": map[string]any{"stateId": stateID},
	}
	if err := c.doWithRetry(ctx, issueUpdateMutation, vars, &resp); err != nil {
		return fmt.Errorf("set ticket status: %w", err)
	}
	if !resp.IssueUpdate.Success {
		return NewAPIError(fmt.Sprintf("linear: issueUpdate failed for ticket %s", id), nil)
	}
	return nil
}

func (c *httpClient) ApplyLabel(ctx context.Context, id string, label string) error {
	labelID, err := c.resolveLabelID(ctx, label)
	if err != nil {
		return fmt.Errorf("apply label: %w", err)
	}

	var resp issueUpdateResponse
	vars := map[string]any{
		"id":    id,
		"input": map[string]any{"addedLabelIds": []string{labelID}},
	}
	if err := c.doWithRetry(ctx, issueUpdateMutation, vars, &resp); err != nil {
		return fmt.Errorf("apply label: %w", err)
	}
	if !resp.IssueUpdate.Success {
		return NewAPIError(fmt.Sprintf("linear: issueUpdate failed for ticket %s", id), nil)
	}
	return nil
}

func (c *httpClient) RemoveLabel(ctx context.Context, id string, label string) error {
	labelID, err := c.resolveLabelID(ctx, label)
	if err != nil {
		return fmt.Errorf("remove label: %w", err)
	}

	var resp issueUpdateResponse
	vars := map[string]any{
		"id":    id,
		"input": map[string]any{"removedLabelIds": []string{labelID}},
	}
	if err := c.doWithRetry(ctx, issueUpdateMutation, vars, &resp); err != nil {
		return fmt.Errorf("remove label: %w", err)
	}
	if !resp.IssueUpdate.Success {
		return NewAPIError(fmt.Sprintf("linear: issueUpdate failed for ticket %s", id), nil)
	}
	return nil
}

func (c *httpClient) CreateTicket(ctx context.Context, input CreateTicketInput) (*Ticket, error) {
	createInput := map[string]any{
		"teamId": input.TeamID,
	}
	if input.Title != "" {
		createInput["title"] = input.Title
	}
	if input.Description != "" {
		createInput["description"] = input.Description
	}
	if input.StateID != "" {
		createInput["stateId"] = input.StateID
	}
	if len(input.LabelIDs) > 0 {
		createInput["labelIds"] = input.LabelIDs
	}
	if input.ProjectID != "" {
		createInput["projectId"] = input.ProjectID
	}
	if input.Priority != nil {
		createInput["priority"] = *input.Priority
	}
	if input.AssigneeID != "" {
		createInput["assigneeId"] = input.AssigneeID
	}

	var resp issueCreateResponse
	vars := map[string]any{"input": createInput}
	if err := c.doWithRetry(ctx, issueCreateMutation, vars, &resp); err != nil {
		return nil, fmt.Errorf("create ticket: %w", err)
	}
	if !resp.IssueCreate.Success {
		return nil, NewAPIError("linear: issueCreate failed", nil)
	}
	t := resp.IssueCreate.Issue.toTicket()
	return &t, nil
}

func (c *httpClient) PostComment(ctx context.Context, issueID string, body string) error {
	var resp commentCreateResponse
	vars := map[string]any{
		"input": map[string]any{
			"issueId": issueID,
			"body":    body,
		},
	}
	if err := c.doWithRetry(ctx, commentCreateMutation, vars, &resp); err != nil {
		return fmt.Errorf("post comment: %w", err)
	}
	if !resp.CommentCreate.Success {
		return NewAPIError(fmt.Sprintf("linear: commentCreate failed for issue %s", issueID), nil)
	}
	return nil
}

func (c *httpClient) UploadAttachment(ctx context.Context, ticketID string, input AttachmentInput) error {
	return fmt.Errorf("UploadAttachment: not implemented")
}
