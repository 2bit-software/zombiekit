// Package admin provides the business logic for orchestrator admin operations.
// It is the reuse boundary between CLI (now) and HTTP endpoints (future).
package admin

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/2bit-software/zombiekit/internal/cmux"
	"github.com/2bit-software/zombiekit/internal/state"
	"github.com/2bit-software/zombiekit/internal/worktree"
)

// Service provides admin operations over orchestrator state.
type Service struct {
	store state.StateStore
}

// New creates an admin Service backed by the given store.
func New(store state.StateStore) *Service {
	return &Service{store: store}
}

// JobFilter controls which jobs are returned by ListJobs.
type JobFilter struct {
	Statuses []string
}

// DeleteResult describes what happened when a job was deleted.
type DeleteResult struct {
	Job             state.Job
	SlotReleased    bool
	SessionKilled   bool
	SessionErr      error
	WorktreeDeleted bool
	WorktreeErr     error
}

// DeleteOption controls optional cleanup during job deletion.
type DeleteOption func(*deleteConfig)

type deleteConfig struct {
	killSession    bool
	sessionMgr     cmux.SessionManager
	deleteWorktree bool
	worktreeMgr    worktree.Manager
}

// WithSessionCleanup kills the cmux workspace for the job.
func WithSessionCleanup(mgr cmux.SessionManager) DeleteOption {
	return func(c *deleteConfig) {
		c.killSession = true
		c.sessionMgr = mgr
	}
}

// WithWorktreeCleanup deletes the git worktree and branch for the job.
func WithWorktreeCleanup(mgr worktree.Manager) DeleteOption {
	return func(c *deleteConfig) {
		c.deleteWorktree = true
		c.worktreeMgr = mgr
	}
}

// ListJobs returns jobs matching the filter. An empty Statuses slice returns all jobs.
func (s *Service) ListJobs(ctx context.Context, projectID string, filter JobFilter) ([]state.Job, error) {
	if len(filter.Statuses) == 0 {
		return s.store.ListAllJobs(ctx)
	}
	return s.store.ListJobsByStatus(ctx, projectID, filter.Statuses...)
}

// GetJob retrieves a single job by ticket ID.
// Returns state.ErrJobNotFound if the job does not exist.
func (s *Service) GetJob(ctx context.Context, projectID, ticketID string) (*state.Job, error) {
	job, err := s.store.GetJob(ctx, projectID, ticketID)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, fmt.Errorf("get job %s: %w", ticketID, state.ErrJobNotFound)
	}
	return job, nil
}

// DeleteJob removes a job and releases its concurrency slot if the job has a project ID.
//
// Optional cleanup (cmux session, git worktree) runs before the DB delete so
// the user can retry on partial failure. Cleanup errors are captured in the
// result, not returned — the job record is always deleted if reachable.
func (s *Service) DeleteJob(ctx context.Context, projectID, ticketID string, opts ...DeleteOption) (*DeleteResult, error) {
	job, err := s.GetJob(ctx, projectID, ticketID)
	if err != nil {
		return nil, err
	}

	var cfg deleteConfig
	for _, opt := range opts {
		opt(&cfg)
	}

	result := &DeleteResult{Job: *job}

	if cfg.killSession && job.CmuxSession != "" {
		if err := cfg.sessionMgr.KillSession(ctx, ticketID); err != nil {
			result.SessionErr = err
		} else {
			result.SessionKilled = true
		}
	}

	if cfg.deleteWorktree && job.WorktreePath != "" {
		if err := cfg.worktreeMgr.DeleteWorktree(ctx, job.WorktreePath); err != nil {
			result.WorktreeErr = err
		} else {
			result.WorktreeDeleted = true
		}
	}

	if err := s.store.DeleteJob(ctx, projectID, ticketID); err != nil {
		return nil, err
	}

	if job.ProjectID != "" {
		if err := s.store.ReleaseSlot(ctx, job.ProjectID); err != nil {
			return nil, fmt.Errorf("release slot for %s: %w", job.ProjectID, err)
		}
		result.SlotReleased = true
	}
	return result, nil
}

// SetJobStatus updates a job's status after validating against known status constants.
func (s *Service) SetJobStatus(ctx context.Context, projectID, ticketID, status string) error {
	if !slices.Contains(state.ValidStatuses, status) {
		return fmt.Errorf("invalid status %q (valid: %s)", status, strings.Join(state.ValidStatuses, ", "))
	}
	return s.store.SetJobStatus(ctx, projectID, ticketID, status)
}

// ListSlots returns all concurrency slot records.
func (s *Service) ListSlots(ctx context.Context) ([]state.ConcurrencySlot, error) {
	return s.store.ListSlots(ctx)
}

// ResetSlots sets all concurrency slot active counts to zero.
// Returns the number of projects that were reset.
func (s *Service) ResetSlots(ctx context.Context) (int, error) {
	return s.store.ResetAllSlots(ctx)
}
