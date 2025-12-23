// Package codereasoning provides the MCP code-reasoning tool implementation.
package codereasoning

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Session errors.
var (
	ErrSessionCompleted      = errors.New("session already completed")
	ErrInvalidThoughtNumber  = errors.New("invalid thought number")
	ErrExceedsTotalThoughts  = errors.New("thought number exceeds total_thoughts")
	ErrCannotReviseAndBranch = errors.New("cannot revise and branch in the same request")
	ErrInvalidRevisionTarget = errors.New("invalid revision target")
	ErrInvalidBranchPoint    = errors.New("invalid branch point")
	ErrMissingBranchID       = errors.New("branch_id required when branching")
)

// Session manages a reasoning chain's state for a single connection.
type Session struct {
	mu            sync.RWMutex
	thoughts      []Thought
	branches      map[string][]Thought
	totalThoughts int
	completed     bool
	createdAt     time.Time
}

// NewSession creates a new reasoning session.
func NewSession() *Session {
	return &Session{
		thoughts:  make([]Thought, 0),
		branches:  make(map[string][]Thought),
		createdAt: time.Now(),
	}
}

// AddThought adds a new thought to the session.
func (s *Session) AddThought(req ThoughtRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.completed {
		return ErrSessionCompleted
	}

	// Validate thought number
	if req.IsRevision && req.BranchID != "" {
		return ErrCannotReviseAndBranch
	}

	// Set total thoughts on first thought
	if len(s.thoughts) == 0 && s.totalThoughts == 0 {
		s.totalThoughts = req.TotalThoughts
	}

	// Update total thoughts if adjusted
	if req.TotalThoughts > s.totalThoughts {
		s.totalThoughts = req.TotalThoughts
	}

	// Handle branching
	if req.BranchID != "" {
		return s.addBranchThought(req)
	}

	// Handle revision
	if req.IsRevision {
		return s.addRevisionThought(req)
	}

	// Normal thought - must be sequential
	expectedNumber := len(s.thoughts) + 1
	if req.ThoughtNumber != expectedNumber {
		return fmt.Errorf("%w: expected %d, got %d", ErrInvalidThoughtNumber, expectedNumber, req.ThoughtNumber)
	}

	if req.ThoughtNumber > s.totalThoughts {
		return fmt.Errorf("%w: thought %d exceeds declared total %d", ErrExceedsTotalThoughts, req.ThoughtNumber, s.totalThoughts)
	}

	thought := Thought{
		Number:    req.ThoughtNumber,
		Content:   req.Thought,
		CreatedAt: time.Now(),
	}

	s.thoughts = append(s.thoughts, thought)

	// Mark completed if this is the final thought
	if !req.NextThoughtNeeded {
		s.completed = true
	}

	return nil
}

func (s *Session) addBranchThought(req ThoughtRequest) error {
	if req.BranchID == "" {
		return ErrMissingBranchID
	}

	// Validate branch point
	if req.BranchFromThought < 1 || req.BranchFromThought > len(s.thoughts) {
		return fmt.Errorf("%w: thought %d doesn't exist", ErrInvalidBranchPoint, req.BranchFromThought)
	}

	thought := Thought{
		Number:           req.ThoughtNumber,
		Content:          req.Thought,
		BranchID:         req.BranchID,
		BranchFromNumber: req.BranchFromThought,
		CreatedAt:        time.Now(),
	}

	s.branches[req.BranchID] = append(s.branches[req.BranchID], thought)

	if !req.NextThoughtNeeded {
		s.completed = true
	}

	return nil
}

func (s *Session) addRevisionThought(req ThoughtRequest) error {
	// Validate revision target
	if req.RevisesThought < 1 || req.RevisesThought > len(s.thoughts) {
		return fmt.Errorf("%w: thought %d doesn't exist", ErrInvalidRevisionTarget, req.RevisesThought)
	}

	// Replace the thought at the revision target
	thought := Thought{
		Number:        req.RevisesThought,
		Content:       req.Thought,
		IsRevision:    true,
		RevisesNumber: req.RevisesThought,
		CreatedAt:     time.Now(),
	}

	s.thoughts[req.RevisesThought-1] = thought

	if !req.NextThoughtNeeded {
		s.completed = true
	}

	return nil
}

// Complete marks the session as completed.
func (s *Session) Complete() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.completed = true
}

// IsCompleted returns whether the session is completed.
func (s *Session) IsCompleted() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.completed
}

// Format returns a formatted display of the reasoning chain.
func (s *Session) Format() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sb strings.Builder

	for _, t := range s.thoughts {
		prefix := ""
		if t.IsRevision {
			prefix = "🔄 "
		}
		sb.WriteString(fmt.Sprintf("[%d/%d] %s%s\n", t.Number, s.totalThoughts, prefix, t.Content))
	}

	// Add branches
	for branchID, branchThoughts := range s.branches {
		sb.WriteString(fmt.Sprintf("\n🌿 Branch: %s\n", branchID))
		for _, t := range branchThoughts {
			sb.WriteString(fmt.Sprintf("  [%d] (from %d) %s\n", t.Number, t.BranchFromNumber, t.Content))
		}
	}

	return sb.String()
}

// GetStatus returns the current status ("in_progress" or "completed").
func (s *Session) GetStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.completed {
		return "completed"
	}
	return "in_progress"
}

// GetTotalThoughts returns the total expected thoughts.
func (s *Session) GetTotalThoughts() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.totalThoughts
}

// GetCurrentThoughtNumber returns the current thought number.
func (s *Session) GetCurrentThoughtNumber() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.thoughts)
}

// GetBranchIDs returns all branch identifiers.
func (s *Session) GetBranchIDs() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.branches))
	for id := range s.branches {
		ids = append(ids, id)
	}
	return ids
}

// CreatedAt returns when the session was created.
func (s *Session) CreatedAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.createdAt
}
