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
	tickets, err := c.PollReadyTickets(context.Background(), "ai-ready")
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
