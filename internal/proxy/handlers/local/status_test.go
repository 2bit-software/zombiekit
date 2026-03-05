package local

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockConnection struct {
	configured bool
	healthy    bool
	errMsg     string
	url        string
	checkTime  time.Time
}

func (m *mockConnection) IsConfigured() bool                             { return m.configured }
func (m *mockConnection) HealthCheck(_ context.Context) (bool, string)   { return m.healthy, m.errMsg }
func (m *mockConnection) ServerURL() string                              { return m.url }
func (m *mockConnection) LastCheck() time.Time                           { return m.checkTime }

func TestConnectionStatus_NotConfigured(t *testing.T) {
	handler := NewConnectionStatusHandler(nil)

	result, err := handler(context.Background(), nil)
	require.NoError(t, err)

	var resp statusResponse
	require.NoError(t, json.Unmarshal([]byte(result), &resp))

	assert.False(t, resp.Connected)
	assert.Equal(t, "server not configured", resp.Error)
}

func TestConnectionStatus_Healthy(t *testing.T) {
	conn := &mockConnection{
		configured: true,
		healthy:    true,
		url:        "http://localhost:8080",
		checkTime:  time.Now(),
	}
	handler := NewConnectionStatusHandler(conn)

	result, err := handler(context.Background(), nil)
	require.NoError(t, err)

	var resp statusResponse
	require.NoError(t, json.Unmarshal([]byte(result), &resp))

	assert.True(t, resp.Connected)
	assert.Equal(t, "http://localhost:8080", resp.ServerURL)
	assert.Empty(t, resp.Error)
}

func TestConnectionStatus_Unhealthy(t *testing.T) {
	conn := &mockConnection{
		configured: true,
		healthy:    false,
		errMsg:     "connection refused",
		url:        "http://localhost:8080",
		checkTime:  time.Now(),
	}
	handler := NewConnectionStatusHandler(conn)

	result, err := handler(context.Background(), nil)
	require.NoError(t, err)

	var resp statusResponse
	require.NoError(t, json.Unmarshal([]byte(result), &resp))

	assert.False(t, resp.Connected)
	assert.Equal(t, "connection refused", resp.Error)
}
