package codereasoning

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestTool(t *testing.T) *Tool {
	t.Helper()
	return NewTool(NewSessionManager())
}

func TestTool_Definition_MatchesContract(t *testing.T) {
	tool := setupTestTool(t)

	def := tool.Definition()

	assert.Equal(t, "code-reasoning", def.Name)
	assert.Contains(t, def.Description, "thinking")

	schema := def.InputSchema
	props, ok := schema["properties"].(map[string]interface{})
	require.True(t, ok)

	assert.Contains(t, props, "thought")
	assert.Contains(t, props, "thought_number")
	assert.Contains(t, props, "total_thoughts")
	assert.Contains(t, props, "next_thought_needed")
}

func TestTool_FirstThought_CreatesSession(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	result, err := tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "First reasoning step",
		"thought_number":      float64(1),
		"total_thoughts":      float64(3),
		"next_thought_needed": true,
	})
	require.NoError(t, err)

	var response ThoughtResponse
	err = json.Unmarshal([]byte(result), &response)
	require.NoError(t, err)

	assert.Equal(t, 1, response.ThoughtNumber)
	assert.Equal(t, 3, response.TotalThoughts)
	assert.Equal(t, "in_progress", response.Status)
	assert.Contains(t, response.Chain, "First reasoning step")
}

func TestTool_SequentialThoughts(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	// First thought
	tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "First",
		"thought_number":      float64(1),
		"total_thoughts":      float64(3),
		"next_thought_needed": true,
	})

	// Second thought
	result, err := tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "Second",
		"thought_number":      float64(2),
		"total_thoughts":      float64(3),
		"next_thought_needed": true,
	})
	require.NoError(t, err)

	var response ThoughtResponse
	json.Unmarshal([]byte(result), &response)

	assert.Equal(t, 2, response.ThoughtNumber)
	assert.Contains(t, response.Chain, "First")
	assert.Contains(t, response.Chain, "Second")
}

func TestTool_Revision_ValidRequest(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	// Add two thoughts
	tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "Original first",
		"thought_number":      float64(1),
		"total_thoughts":      float64(3),
		"next_thought_needed": true,
	})
	tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "Second thought",
		"thought_number":      float64(2),
		"total_thoughts":      float64(3),
		"next_thought_needed": true,
	})

	// Revise the first thought
	result, err := tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "Revised first",
		"thought_number":      float64(3),
		"total_thoughts":      float64(3),
		"is_revision":         true,
		"revises_thought":     float64(1),
		"next_thought_needed": true,
	})
	require.NoError(t, err)

	var response ThoughtResponse
	json.Unmarshal([]byte(result), &response)

	assert.Equal(t, 1, response.RevisedNumber)
	assert.Contains(t, response.Chain, "🔄")
}

func TestTool_Revision_MissingTarget_Error(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "First",
		"thought_number":      float64(1),
		"total_thoughts":      float64(3),
		"next_thought_needed": true,
	})

	// Try to revise without specifying target
	_, err := tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "Revised",
		"thought_number":      float64(2),
		"total_thoughts":      float64(3),
		"is_revision":         true,
		"next_thought_needed": true,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revises_thought")
}

func TestTool_Branch_ValidRequest(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "Main thought",
		"thought_number":      float64(1),
		"total_thoughts":      float64(3),
		"next_thought_needed": true,
	})

	result, err := tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "Alternative approach",
		"thought_number":      float64(2),
		"total_thoughts":      float64(3),
		"branch_from_thought": float64(1),
		"branch_id":           "alt",
		"next_thought_needed": true,
	})
	require.NoError(t, err)

	var response ThoughtResponse
	json.Unmarshal([]byte(result), &response)

	assert.Equal(t, "alt", response.BranchID)
	assert.Equal(t, 1, response.BranchedFrom)
	assert.Contains(t, response.Branches, "alt")
}

func TestTool_Branch_MissingBranchID_Error(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "Main thought",
		"thought_number":      float64(1),
		"total_thoughts":      float64(3),
		"next_thought_needed": true,
	})

	_, err := tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "Branch thought",
		"thought_number":      float64(2),
		"total_thoughts":      float64(3),
		"branch_from_thought": float64(1),
		// Missing branch_id
		"next_thought_needed": true,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch_id")
}

func TestTool_Complete_FinalThought(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "First",
		"thought_number":      float64(1),
		"total_thoughts":      float64(2),
		"next_thought_needed": true,
	})

	result, err := tool.Execute(ctx, "test-session", map[string]interface{}{
		"thought":             "Final conclusion",
		"thought_number":      float64(2),
		"total_thoughts":      float64(2),
		"next_thought_needed": false,
	})
	require.NoError(t, err)

	var response ThoughtResponse
	json.Unmarshal([]byte(result), &response)

	assert.Equal(t, "completed", response.Status)
}

func TestTool_MissingRequiredFields_Error(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	tests := []struct {
		name   string
		args   map[string]interface{}
		errMsg string
	}{
		{
			name:   "missing thought",
			args:   map[string]interface{}{"thought_number": float64(1), "total_thoughts": float64(3), "next_thought_needed": true},
			errMsg: "thought",
		},
		{
			name:   "missing thought_number",
			args:   map[string]interface{}{"thought": "test", "total_thoughts": float64(3), "next_thought_needed": true},
			errMsg: "thought_number",
		},
		{
			name:   "missing total_thoughts",
			args:   map[string]interface{}{"thought": "test", "thought_number": float64(1), "next_thought_needed": true},
			errMsg: "total_thoughts",
		},
		{
			name:   "missing next_thought_needed",
			args:   map[string]interface{}{"thought": "test", "thought_number": float64(1), "total_thoughts": float64(3)},
			errMsg: "next_thought_needed",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tool.Execute(ctx, "test-session-"+tc.name, tc.args)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

func TestTool_DifferentSessions(t *testing.T) {
	tool := setupTestTool(t)
	ctx := context.Background()

	// Session 1
	tool.Execute(ctx, "session-1", map[string]interface{}{
		"thought":             "Session 1 thought",
		"thought_number":      float64(1),
		"total_thoughts":      float64(2),
		"next_thought_needed": true,
	})

	// Session 2
	tool.Execute(ctx, "session-2", map[string]interface{}{
		"thought":             "Session 2 thought",
		"thought_number":      float64(1),
		"total_thoughts":      float64(3),
		"next_thought_needed": true,
	})

	// Verify sessions are independent
	result1, _ := tool.Execute(ctx, "session-1", map[string]interface{}{
		"thought":             "Session 1 second",
		"thought_number":      float64(2),
		"total_thoughts":      float64(2),
		"next_thought_needed": false,
	})

	result2, _ := tool.Execute(ctx, "session-2", map[string]interface{}{
		"thought":             "Session 2 second",
		"thought_number":      float64(2),
		"total_thoughts":      float64(3),
		"next_thought_needed": true,
	})

	var resp1, resp2 ThoughtResponse
	json.Unmarshal([]byte(result1), &resp1)
	json.Unmarshal([]byte(result2), &resp2)

	assert.Equal(t, 2, resp1.TotalThoughts)
	assert.Equal(t, 3, resp2.TotalThoughts)
	assert.Equal(t, "completed", resp1.Status)
	assert.Equal(t, "in_progress", resp2.Status)
}
