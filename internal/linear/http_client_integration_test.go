//go:build integration

package linear

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func integrationClient(t *testing.T) *httpClient {
	t.Helper()
	key := os.Getenv("BRAINS_LINEAR_API_KEY")
	if key == "" {
		t.Skip("BRAINS_LINEAR_API_KEY not set")
	}
	c, err := NewClient(key)
	require.NoError(t, err)
	return c
}

func TestIntegration_PollReadyTickets(t *testing.T) {
	c := integrationClient(t)
	projectID := os.Getenv("ORCH_PROJECT_ID")
	if projectID == "" {
		t.Skip("ORCH_PROJECT_ID not set")
	}
	tickets, err := c.PollReadyTickets(context.Background(), "ai-ready", projectID)
	require.NoError(t, err)
	// We can't assert specific results, but we can assert the shape
	for _, ticket := range tickets {
		assert.NotEmpty(t, ticket.ID)
		assert.NotEmpty(t, ticket.Identifier)
		assert.NotEmpty(t, ticket.Title)
		assert.NotEmpty(t, ticket.Description)
	}
}

func TestIntegration_GetTicket(t *testing.T) {
	c := integrationClient(t)
	// DEV-157 is this ticket -- it should exist
	ticket, err := c.GetTicket(context.Background(), "DEV-157")
	require.NoError(t, err)
	assert.NotEmpty(t, ticket.ID)
	assert.Equal(t, "DEV-157", ticket.Identifier)
	assert.NotEmpty(t, ticket.Title)
	assert.NotEmpty(t, ticket.URL)
}

func TestIntegration_GetTicket_NotFound(t *testing.T) {
	c := integrationClient(t)
	_, err := c.GetTicket(context.Background(), "DEV-99999")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestIntegration_GetTicket_HasTeamID(t *testing.T) {
	c := integrationClient(t)
	ticket, err := c.GetTicket(context.Background(), "DEV-157")
	require.NoError(t, err)
	assert.NotEmpty(t, ticket.TeamID, "TeamID should be populated")
}

func TestIntegration_SetTicketStatus(t *testing.T) {
	c := integrationClient(t)
	// Get current status, change it, then change it back
	ticket, err := c.GetTicket(context.Background(), "DEV-158")
	require.NoError(t, err)
	originalStatus := ticket.Status

	// Try setting to a known status — "In Progress" is standard
	targetStatus := "In Progress"
	if originalStatus == targetStatus {
		targetStatus = "Todo"
	}

	err = c.SetTicketStatus(context.Background(), "DEV-158", targetStatus)
	require.NoError(t, err)

	// Restore original status
	err = c.SetTicketStatus(context.Background(), "DEV-158", originalStatus)
	require.NoError(t, err)
}

func TestIntegration_SetTicketStatus_InvalidStatus(t *testing.T) {
	c := integrationClient(t)
	err := c.SetTicketStatus(context.Background(), "DEV-158", "This Status Does Not Exist")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestIntegration_ResolveLabelID(t *testing.T) {
	c := integrationClient(t)
	// Discover a label from an existing ticket
	ticket, err := c.GetTicket(context.Background(), "DEV-157")
	require.NoError(t, err)
	if len(ticket.Labels) == 0 {
		t.Skip("DEV-157 has no labels to test with")
	}
	label := ticket.Labels[0]
	t.Logf("Testing with label: %q", label)

	id, err := c.resolveLabelID(context.Background(), label)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestIntegration_ApplyAndRemoveLabel(t *testing.T) {
	c := integrationClient(t)
	// Discover a label from an existing ticket
	ticket, err := c.GetTicket(context.Background(), "DEV-157")
	require.NoError(t, err)
	if len(ticket.Labels) == 0 {
		t.Skip("DEV-157 has no labels to test with")
	}
	label := ticket.Labels[0]
	t.Logf("Testing with label: %q", label)

	// Apply label
	err = c.ApplyLabel(context.Background(), "DEV-158", label)
	require.NoError(t, err)

	// Apply again (idempotent)
	err = c.ApplyLabel(context.Background(), "DEV-158", label)
	require.NoError(t, err)

	// Remove
	err = c.RemoveLabel(context.Background(), "DEV-158", label)
	require.NoError(t, err)

	// Remove again (idempotent)
	err = c.RemoveLabel(context.Background(), "DEV-158", label)
	require.NoError(t, err)
}

func TestIntegration_ApplyLabel_NotFound(t *testing.T) {
	c := integrationClient(t)
	err := c.ApplyLabel(context.Background(), "DEV-158", "this-label-does-not-exist-xyz")
	require.Error(t, err)
	assert.True(t, IsNotFound(err))
}

func TestIntegration_CreateTicket(t *testing.T) {
	c := integrationClient(t)

	// Get team ID from an existing ticket
	existing, err := c.GetTicket(context.Background(), "DEV-158")
	require.NoError(t, err)
	require.NotEmpty(t, existing.TeamID)

	ticket, err := c.CreateTicket(context.Background(), CreateTicketInput{
		TeamID:      existing.TeamID,
		Title:       "[TEST] Integration test ticket — safe to delete",
		Description: "Created by DEV-158 integration tests. Safe to delete.",
	})
	require.NoError(t, err)
	assert.NotEmpty(t, ticket.ID)
	assert.NotEmpty(t, ticket.Identifier)
	assert.Equal(t, "[TEST] Integration test ticket — safe to delete", ticket.Title)
	assert.NotEmpty(t, ticket.TeamID)

	t.Logf("Created test ticket: %s (%s) — delete manually if needed", ticket.Identifier, ticket.URL)
}
