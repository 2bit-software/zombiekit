package ghpr_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/2bit-software/zombiekit/internal/mcp/tools/ghpr"
)

func TestNewToolGracefulError(t *testing.T) {
	// If gh is not in PATH, NewTool should return a ToolError
	// We can't reliably test this since gh may be installed,
	// so we test the tool's validation instead
	tool, err := ghpr.NewTool(t.TempDir())
	if err != nil {
		// gh not found -- expected in some environments
		assert.Contains(t, err.Error(), "GH_NOT_FOUND")
		return
	}
	assert.NotNil(t, tool)
}

func TestCreateValidation(t *testing.T) {
	tool, err := ghpr.NewTool(t.TempDir())
	if err != nil {
		t.Skip("gh CLI not available")
	}

	tests := []struct {
		name    string
		args    map[string]any
		wantErr string
	}{
		{
			name:    "missing action",
			args:    map[string]any{},
			wantErr: "MISSING_REQUIRED_PARAM",
		},
		{
			name:    "invalid action",
			args:    map[string]any{"action": "merge"},
			wantErr: "INVALID_ACTION",
		},
		{
			name:    "create missing title",
			args:    map[string]any{"action": "create", "body": "some body"},
			wantErr: "title is required",
		},
		{
			name:    "create missing body",
			args:    map[string]any{"action": "create", "title": "some title"},
			wantErr: "body is required",
		},
		{
			name:    "create empty title",
			args:    map[string]any{"action": "create", "title": "  ", "body": "body"},
			wantErr: "title is required",
		},
		{
			name:    "comment missing pr_number",
			args:    map[string]any{"action": "comment", "body": "comment text"},
			wantErr: "pr_number is required",
		},
		{
			name:    "comment missing body",
			args:    map[string]any{"action": "comment", "pr_number": float64(1)},
			wantErr: "body is required",
		},
		{
			name:    "edit missing pr_number",
			args:    map[string]any{"action": "edit", "title": "new title"},
			wantErr: "pr_number is required",
		},
		{
			name:    "edit missing title and body",
			args:    map[string]any{"action": "edit", "pr_number": float64(1)},
			wantErr: "at least one of title or body is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Execute(context.Background(), tt.args)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
