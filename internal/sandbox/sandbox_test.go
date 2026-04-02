package sandbox

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	tests := []struct {
		ticketID string
		want     string
	}{
		{"DEV-123", "zk-dev-123"},
		{"DEV-456", "zk-dev-456"},
		{"PROJ-1", "zk-proj-1"},
		{"dev-123", "zk-dev-123"},
		{"ABC--DEF", "zk-abc-def"},
		{"---", "zk-"},
		{"A", "zk-a"},
		// Long ticket ID gets truncated to 63 chars.
		{"VERY-LONG-TICKET-ID-THAT-EXCEEDS-THE-DNS-LABEL-LIMIT-OF-63-CHARS-TOTAL", "zk-very-long-ticket-id-that-exceeds-the-dns-label-limit-of-63-c"},
	}

	for _, tt := range tests {
		t.Run(tt.ticketID, func(t *testing.T) {
			got := Name(tt.ticketID)
			assert.Equal(t, tt.want, got)
			assert.LessOrEqual(t, len(got), 63)
		})
	}
}

func TestName_Deterministic(t *testing.T) {
	a := Name("DEV-123")
	b := Name("DEV-123")
	assert.Equal(t, a, b, "Name must be deterministic")
}

func TestRewriteCallbackHost(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		host string
		want map[string]string
	}{
		{
			name: "rewrites localhost URL",
			env:  map[string]string{"WORK_CALLBACK_URL": "http://localhost:8666/DEV-123"},
			host: "host.docker.internal",
			want: map[string]string{"WORK_CALLBACK_URL": "http://host.docker.internal:8666/DEV-123"},
		},
		{
			name: "rewrites 127.0.0.1 URL",
			env:  map[string]string{"WORK_CALLBACK_URL": "http://127.0.0.1:8666/DEV-123/complete"},
			host: "host.docker.internal",
			want: map[string]string{"WORK_CALLBACK_URL": "http://host.docker.internal:8666/DEV-123/complete"},
		},
		{
			name: "leaves non-localhost URL unchanged",
			env:  map[string]string{"WORK_CALLBACK_URL": "http://api.example.com:8666/DEV-123"},
			host: "host.docker.internal",
			want: map[string]string{"WORK_CALLBACK_URL": "http://api.example.com:8666/DEV-123"},
		},
		{
			name: "leaves non-URL value unchanged",
			env:  map[string]string{"FOO": "bar", "WORK_CALLBACK_URL": "http://localhost:8666/DEV-123"},
			host: "host.docker.internal",
			want: map[string]string{"FOO": "bar", "WORK_CALLBACK_URL": "http://host.docker.internal:8666/DEV-123"},
		},
		{
			name: "nil env returns nil",
			env:  nil,
			host: "host.docker.internal",
			want: nil,
		},
		{
			name: "empty host returns original",
			env:  map[string]string{"WORK_CALLBACK_URL": "http://localhost:8666/DEV-123"},
			host: "",
			want: map[string]string{"WORK_CALLBACK_URL": "http://localhost:8666/DEV-123"},
		},
		{
			name: "localhost without port",
			env:  map[string]string{"URL": "http://localhost/path"},
			host: "host.docker.internal",
			want: map[string]string{"URL": "http://host.docker.internal/path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RewriteCallbackHost(tt.env, tt.host)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRewriteCallbackHost_DoesNotMutateOriginal(t *testing.T) {
	original := map[string]string{"WORK_CALLBACK_URL": "http://localhost:8666/DEV-123"}
	RewriteCallbackHost(original, "host.docker.internal")
	assert.Equal(t, "http://localhost:8666/DEV-123", original["WORK_CALLBACK_URL"])
}
