package local

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zombiekit/brains/internal/proxy/handlers"
)

type connectionChecker interface {
	IsConfigured() bool
	HealthCheck(ctx context.Context) (bool, string)
	ServerURL() string
	LastCheck() time.Time
}

type statusResponse struct {
	Connected bool   `json:"connected"`
	ServerURL string `json:"server_url"`
	LastCheck string `json:"last_check"`
	Error     string `json:"error,omitempty"`
}

func NewConnectionStatusHandler(conn connectionChecker) handlers.Handler {
	return func(ctx context.Context, _ map[string]any) (string, error) {
		if conn == nil || !conn.IsConfigured() {
			return marshalStatus(statusResponse{
				Connected: false,
				Error:     "server not configured",
			})
		}

		ok, errMsg := conn.HealthCheck(ctx)
		resp := statusResponse{
			Connected: ok,
			ServerURL: conn.ServerURL(),
			LastCheck: conn.LastCheck().Format(time.RFC3339),
		}
		if !ok {
			resp.Error = errMsg
		}
		return marshalStatus(resp)
	}
}

func marshalStatus(resp statusResponse) (string, error) {
	data, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
