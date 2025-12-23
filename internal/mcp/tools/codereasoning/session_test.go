package codereasoning

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSession_AddThought_Sequential(t *testing.T) {
	s := NewSession()

	err := s.AddThought(ThoughtRequest{
		Thought:           "First thought",
		ThoughtNumber:     1,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})
	require.NoError(t, err)

	err = s.AddThought(ThoughtRequest{
		Thought:           "Second thought",
		ThoughtNumber:     2,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})
	require.NoError(t, err)

	assert.Equal(t, 2, s.GetCurrentThoughtNumber())
	assert.Equal(t, "in_progress", s.GetStatus())
}

func TestSession_AddThought_NonSequential_Error(t *testing.T) {
	s := NewSession()

	err := s.AddThought(ThoughtRequest{
		Thought:           "First thought",
		ThoughtNumber:     1,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})
	require.NoError(t, err)

	// Try to add thought 3 when we should add thought 2
	err = s.AddThought(ThoughtRequest{
		Thought:           "Third thought",
		ThoughtNumber:     3,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected 2")
}

func TestSession_AddThought_ExceedsTotal_Error(t *testing.T) {
	s := NewSession()

	err := s.AddThought(ThoughtRequest{
		Thought:           "First thought",
		ThoughtNumber:     1,
		TotalThoughts:     2,
		NextThoughtNeeded: true,
	})
	require.NoError(t, err)

	err = s.AddThought(ThoughtRequest{
		Thought:           "Second thought",
		ThoughtNumber:     2,
		TotalThoughts:     2,
		NextThoughtNeeded: true,
	})
	require.NoError(t, err)

	// Try to add thought 3 when total is 2
	err = s.AddThought(ThoughtRequest{
		Thought:           "Third thought",
		ThoughtNumber:     3,
		TotalThoughts:     2,
		NextThoughtNeeded: true,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds")
}

func TestSession_AddThought_AfterComplete_Error(t *testing.T) {
	s := NewSession()

	err := s.AddThought(ThoughtRequest{
		Thought:           "Final thought",
		ThoughtNumber:     1,
		TotalThoughts:     1,
		NextThoughtNeeded: false, // Marks session as complete
	})
	require.NoError(t, err)

	assert.True(t, s.IsCompleted())

	// Try to add another thought
	err = s.AddThought(ThoughtRequest{
		Thought:           "Another thought",
		ThoughtNumber:     2,
		TotalThoughts:     2,
		NextThoughtNeeded: true,
	})

	assert.ErrorIs(t, err, ErrSessionCompleted)
}

func TestSession_Revision_ValidTarget(t *testing.T) {
	s := NewSession()

	// Add two thoughts
	s.AddThought(ThoughtRequest{
		Thought:           "Original first thought",
		ThoughtNumber:     1,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})
	s.AddThought(ThoughtRequest{
		Thought:           "Original second thought",
		ThoughtNumber:     2,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})

	// Revise the first thought
	err := s.AddThought(ThoughtRequest{
		Thought:           "Revised first thought",
		ThoughtNumber:     3, // Current thought number
		TotalThoughts:     3,
		IsRevision:        true,
		RevisesThought:    1,
		NextThoughtNeeded: true,
	})
	require.NoError(t, err)

	// Verify the revision is reflected in the format
	format := s.Format()
	assert.Contains(t, format, "🔄")
	assert.Contains(t, format, "Revised first thought")
}

func TestSession_Revision_InvalidTarget_Error(t *testing.T) {
	s := NewSession()

	s.AddThought(ThoughtRequest{
		Thought:           "First thought",
		ThoughtNumber:     1,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})

	// Try to revise a thought that doesn't exist
	err := s.AddThought(ThoughtRequest{
		Thought:           "Revised thought",
		ThoughtNumber:     2,
		TotalThoughts:     3,
		IsRevision:        true,
		RevisesThought:    5, // Doesn't exist
		NextThoughtNeeded: true,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "doesn't exist")
}

func TestSession_Branch_CreatesNewBranch(t *testing.T) {
	s := NewSession()

	// Add initial thoughts
	s.AddThought(ThoughtRequest{
		Thought:           "First thought",
		ThoughtNumber:     1,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})
	s.AddThought(ThoughtRequest{
		Thought:           "Second thought",
		ThoughtNumber:     2,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})

	// Create a branch from thought 1
	err := s.AddThought(ThoughtRequest{
		Thought:           "Alternative approach",
		ThoughtNumber:     3,
		TotalThoughts:     3,
		BranchID:          "alternative",
		BranchFromThought: 1,
		NextThoughtNeeded: true,
	})
	require.NoError(t, err)

	branches := s.GetBranchIDs()
	assert.Contains(t, branches, "alternative")
}

func TestSession_Branch_AddsToExistingBranch(t *testing.T) {
	s := NewSession()

	s.AddThought(ThoughtRequest{
		Thought:           "First thought",
		ThoughtNumber:     1,
		TotalThoughts:     5,
		NextThoughtNeeded: true,
	})

	// Create first thought in branch
	s.AddThought(ThoughtRequest{
		Thought:           "Branch thought 1",
		ThoughtNumber:     2,
		TotalThoughts:     5,
		BranchID:          "test-branch",
		BranchFromThought: 1,
		NextThoughtNeeded: true,
	})

	// Add another thought to the same branch
	err := s.AddThought(ThoughtRequest{
		Thought:           "Branch thought 2",
		ThoughtNumber:     3,
		TotalThoughts:     5,
		BranchID:          "test-branch",
		BranchFromThought: 1,
		NextThoughtNeeded: true,
	})
	require.NoError(t, err)

	format := s.Format()
	assert.Contains(t, format, "test-branch")
	assert.Contains(t, format, "Branch thought 1")
	assert.Contains(t, format, "Branch thought 2")
}

func TestSession_Branch_InvalidBranchPoint_Error(t *testing.T) {
	s := NewSession()

	s.AddThought(ThoughtRequest{
		Thought:           "First thought",
		ThoughtNumber:     1,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})

	// Try to branch from a thought that doesn't exist
	err := s.AddThought(ThoughtRequest{
		Thought:           "Branch thought",
		ThoughtNumber:     2,
		TotalThoughts:     3,
		BranchID:          "test-branch",
		BranchFromThought: 5, // Doesn't exist
		NextThoughtNeeded: true,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "doesn't exist")
}

func TestSession_RevisionAndBranch_Conflict_Error(t *testing.T) {
	s := NewSession()

	s.AddThought(ThoughtRequest{
		Thought:           "First thought",
		ThoughtNumber:     1,
		TotalThoughts:     3,
		NextThoughtNeeded: true,
	})

	// Try to both revise and branch
	err := s.AddThought(ThoughtRequest{
		Thought:           "Invalid thought",
		ThoughtNumber:     2,
		TotalThoughts:     3,
		IsRevision:        true,
		RevisesThought:    1,
		BranchID:          "test-branch",
		BranchFromThought: 1,
		NextThoughtNeeded: true,
	})

	assert.ErrorIs(t, err, ErrCannotReviseAndBranch)
}

func TestSession_Complete_SetsFlag(t *testing.T) {
	s := NewSession()

	s.AddThought(ThoughtRequest{
		Thought:           "Only thought",
		ThoughtNumber:     1,
		TotalThoughts:     1,
		NextThoughtNeeded: false,
	})

	assert.True(t, s.IsCompleted())
	assert.Equal(t, "completed", s.GetStatus())
}

func TestSession_Format_ShowsChain(t *testing.T) {
	s := NewSession()

	s.AddThought(ThoughtRequest{
		Thought:           "First thought",
		ThoughtNumber:     1,
		TotalThoughts:     2,
		NextThoughtNeeded: true,
	})
	s.AddThought(ThoughtRequest{
		Thought:           "Second thought",
		ThoughtNumber:     2,
		TotalThoughts:     2,
		NextThoughtNeeded: false,
	})

	format := s.Format()
	assert.Contains(t, format, "[1/2] First thought")
	assert.Contains(t, format, "[2/2] Second thought")
}

func TestSession_Format_ShowsRevisionMarker(t *testing.T) {
	s := NewSession()

	s.AddThought(ThoughtRequest{
		Thought:           "Original thought",
		ThoughtNumber:     1,
		TotalThoughts:     2,
		NextThoughtNeeded: true,
	})
	s.AddThought(ThoughtRequest{
		Thought:           "Revised thought",
		ThoughtNumber:     2,
		TotalThoughts:     2,
		IsRevision:        true,
		RevisesThought:    1,
		NextThoughtNeeded: false,
	})

	format := s.Format()
	assert.Contains(t, format, "🔄")
}

func TestSession_Format_ShowsBranchMarker(t *testing.T) {
	s := NewSession()

	s.AddThought(ThoughtRequest{
		Thought:           "Main thought",
		ThoughtNumber:     1,
		TotalThoughts:     2,
		NextThoughtNeeded: true,
	})
	s.AddThought(ThoughtRequest{
		Thought:           "Branch thought",
		ThoughtNumber:     2,
		TotalThoughts:     2,
		BranchID:          "alt",
		BranchFromThought: 1,
		NextThoughtNeeded: false,
	})

	format := s.Format()
	assert.Contains(t, format, "🌿")
	assert.Contains(t, format, "Branch: alt")
}

func TestSession_TotalThoughts_CanIncrease(t *testing.T) {
	s := NewSession()

	s.AddThought(ThoughtRequest{
		Thought:           "First thought",
		ThoughtNumber:     1,
		TotalThoughts:     2,
		NextThoughtNeeded: true,
	})

	assert.Equal(t, 2, s.GetTotalThoughts())

	// Increase total thoughts
	s.AddThought(ThoughtRequest{
		Thought:           "Second thought",
		ThoughtNumber:     2,
		TotalThoughts:     5, // Increased from 2 to 5
		NextThoughtNeeded: true,
	})

	assert.Equal(t, 5, s.GetTotalThoughts())
}

func TestSession_ThreadSafe(t *testing.T) {
	s := NewSession()
	done := make(chan bool)

	// Add initial thought
	s.AddThought(ThoughtRequest{
		Thought:           "Initial thought",
		ThoughtNumber:     1,
		TotalThoughts:     100,
		NextThoughtNeeded: true,
	})

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			s.GetStatus()
			s.GetTotalThoughts()
			s.GetCurrentThoughtNumber()
			s.Format()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic or deadlock
	assert.True(t, true)
}

func TestSession_Format_EmptySession(t *testing.T) {
	s := NewSession()
	format := s.Format()
	assert.Equal(t, "", strings.TrimSpace(format))
}
