// Package state provides persistent state management for the orchestrator daemon.
// It owns the StateStore interface and its SQLite implementation.
package state

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Job status constants.
const (
	StatusQueued         = "queued"
	StatusInProgress     = "in-progress"
	StatusNeedsAttention = "needs-attention"
	StatusComplete       = "complete"
	StatusClosed         = "closed"
)

// ValidStatuses lists all recognized job status values.
var ValidStatuses = []string{StatusQueued, StatusInProgress, StatusNeedsAttention, StatusComplete, StatusClosed}

// Job represents an autonomous development task tracked by the orchestrator.
type Job struct {
	TicketID     string
	WorktreePath string
	CmuxSession  string
	PRNumber     *int64
	Status       string
	ProjectID    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ConcurrencySlot represents a per-project concurrency capacity record.
type ConcurrencySlot struct {
	ProjectID   string
	ActiveCount int
	SlotLimit   int
}

// StateStore defines the interface for orchestrator persistent state.
type StateStore interface {
	Migrate(ctx context.Context) error
	Close() error

	CreateJob(ctx context.Context, ticketID, worktreePath, cmuxSession, projectID string) error
	GetJob(ctx context.Context, projectID, ticketID string) (*Job, error)
	ListAllJobs(ctx context.Context) ([]Job, error)
	ListJobsByStatus(ctx context.Context, projectID string, statuses ...string) ([]Job, error)
	DeleteJob(ctx context.Context, projectID, ticketID string) error
	SetJobStatus(ctx context.Context, projectID, ticketID string, status string) error
	SetPR(ctx context.Context, projectID, ticketID string, prNumber int64) error

	GetJobByPR(ctx context.Context, projectID string, prNumber int64) (*Job, error)

	GetCommentWatermark(ctx context.Context, projectID string, prNumber int64) (int64, error)
	SetCommentWatermark(ctx context.Context, projectID string, prNumber int64, commentID int64) error

	TryAcquireSlot(ctx context.Context, projectID string, limit int) (bool, error)
	ReleaseSlot(ctx context.Context, projectID string) error
	ResetAllSlots(ctx context.Context) (int, error)
	ListSlots(ctx context.Context) ([]ConcurrencySlot, error)
}

// SQLiteStore implements StateStore backed by a local SQLite database.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite-backed state store.
// It creates parent directories, opens the connection with
// appropriate pragmas, and runs any pending migrations.
func NewSQLiteStore(ctx context.Context, dbPath string) (*SQLiteStore, error) {
	if dbPath == "" {
		return nil, fmt.Errorf("open state store: %w", ErrInvalidDBPath)
	}

	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create state directory %s: %w", dir, err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open state store: %w", err)
	}

	db.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
		"PRAGMA synchronous=NORMAL",
	}
	for _, pragma := range pragmas {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("set %s: %w", pragma, err)
		}
	}

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("state store connection failed: %w", err)
	}

	if err := RunMigrations(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("run state migrations: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// DB returns the underlying *sql.DB for use by CRUD operations (DEV-154).
func (s *SQLiteStore) DB() *sql.DB {
	return s.db
}

// Migrate runs any pending database migrations.
func (s *SQLiteStore) Migrate(ctx context.Context) error {
	return RunMigrations(ctx, s.db)
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// CreateJob persists a new job record with status "queued".
// Returns ErrJobExists if a job with the same ticket ID already exists.
func (s *SQLiteStore) CreateJob(ctx context.Context, ticketID, worktreePath, cmuxSession, projectID string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO jobs (ticket_id, worktree_path, cmux_session, project_id, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		ticketID, worktreePath, cmuxSession, projectID, StatusQueued, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("create job %s: %w", ticketID, ErrJobExists)
		}
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

// jobColumns is the canonical column list for job SELECT queries.
const jobColumns = `ticket_id, worktree_path, cmux_session, pr_number, status, project_id, created_at, updated_at`

// scanJob scans a single row into a Job struct. The row must contain jobColumns in order.
func scanJob(scanner interface{ Scan(dest ...any) error }) (Job, error) {
	var job Job
	var prNumber sql.NullInt64
	err := scanner.Scan(
		&job.TicketID, &job.WorktreePath, &job.CmuxSession,
		&prNumber, &job.Status, &job.ProjectID,
		&job.CreatedAt, &job.UpdatedAt,
	)
	if prNumber.Valid {
		job.PRNumber = &prNumber.Int64
	}
	return job, err
}

// scanJobs scans all rows into a Job slice. Returns []Job{} (not nil) when empty.
func scanJobs(rows *sql.Rows) ([]Job, error) {
	var jobs []Job
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, fmt.Errorf("scan job row: %w", err)
		}
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate job rows: %w", err)
	}
	if jobs == nil {
		return []Job{}, nil
	}
	return jobs, nil
}

// GetJob retrieves a job by project and ticket ID.
// Returns nil, nil if no job exists for the given key.
func (s *SQLiteStore) GetJob(ctx context.Context, projectID, ticketID string) (*Job, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE project_id = ? AND ticket_id = ?`,
		projectID, ticketID,
	)
	job, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get job %s/%s: %w", projectID, ticketID, err)
	}
	return &job, nil
}

// ListAllJobs returns every job ordered by updated_at descending.
// Returns an empty slice (not nil) when no jobs exist.
func (s *SQLiteStore) ListAllJobs(ctx context.Context) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT `+jobColumns+` FROM jobs ORDER BY updated_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list all jobs: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanJobs(rows)
}

// ListJobsByStatus returns all jobs for a project matching any of the given statuses.
// Returns an empty slice (not nil) when no jobs match.
func (s *SQLiteStore) ListJobsByStatus(ctx context.Context, projectID string, statuses ...string) ([]Job, error) {
	if len(statuses) == 0 {
		return []Job{}, nil
	}

	placeholders := make([]string, len(statuses))
	args := make([]any, 1+len(statuses))
	args[0] = projectID
	for i, status := range statuses {
		placeholders[i] = "?"
		args[i+1] = status
	}

	query := fmt.Sprintf(
		`SELECT `+jobColumns+` FROM jobs WHERE project_id = ? AND status IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list jobs by status: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return scanJobs(rows)
}

// DeleteJob removes a job record by project and ticket ID.
// Returns ErrJobNotFound if no job exists for the given key.
func (s *SQLiteStore) DeleteJob(ctx context.Context, projectID, ticketID string) error {
	result, err := s.db.ExecContext(ctx,
		`DELETE FROM jobs WHERE project_id = ? AND ticket_id = ?`, projectID, ticketID,
	)
	if err != nil {
		return fmt.Errorf("delete job %s/%s: %w", projectID, ticketID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete job %s/%s: %w", projectID, ticketID, err)
	}
	if n == 0 {
		return fmt.Errorf("delete job %s/%s: %w", projectID, ticketID, ErrJobNotFound)
	}
	return nil
}

// SetJobStatus updates the status of a job and its updated_at timestamp.
// Returns ErrJobNotFound if no job exists for the given key.
func (s *SQLiteStore) SetJobStatus(ctx context.Context, projectID, ticketID string, status string) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET status = ?, updated_at = ? WHERE project_id = ? AND ticket_id = ?`,
		status, time.Now(), projectID, ticketID,
	)
	if err != nil {
		return fmt.Errorf("set status for job %s/%s: %w", projectID, ticketID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("set status for job %s/%s: %w", projectID, ticketID, err)
	}
	if n == 0 {
		return fmt.Errorf("set status for job %s/%s: %w", projectID, ticketID, ErrJobNotFound)
	}
	return nil
}

// SetPR associates a PR number with an existing job.
// Returns ErrJobNotFound if no job exists for the given key.
func (s *SQLiteStore) SetPR(ctx context.Context, projectID, ticketID string, prNumber int64) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET pr_number = ?, updated_at = ? WHERE project_id = ? AND ticket_id = ?`,
		prNumber, time.Now(), projectID, ticketID,
	)
	if err != nil {
		return fmt.Errorf("set PR for job %s/%s: %w", projectID, ticketID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("set PR for job %s/%s: %w", projectID, ticketID, err)
	}
	if n == 0 {
		return fmt.Errorf("set PR for job %s/%s: %w", projectID, ticketID, ErrJobNotFound)
	}
	return nil
}

// GetJobByPR retrieves a job by project and PR number.
// Returns nil, nil if no job exists for the given key.
func (s *SQLiteStore) GetJobByPR(ctx context.Context, projectID string, prNumber int64) (*Job, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT `+jobColumns+` FROM jobs WHERE project_id = ? AND pr_number = ?`,
		projectID, prNumber,
	)
	job, err := scanJob(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get job by PR %s/%d: %w", projectID, prNumber, err)
	}
	return &job, nil
}

// GetCommentWatermark returns the last processed comment ID for a project's PR.
// Returns 0 if no watermark has been set.
func (s *SQLiteStore) GetCommentWatermark(ctx context.Context, projectID string, prNumber int64) (int64, error) {
	var commentID int64
	err := s.db.QueryRowContext(ctx,
		`SELECT last_processed_comment_id FROM comment_watermarks WHERE project_id = ? AND pr_number = ?`,
		projectID, prNumber,
	).Scan(&commentID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get comment watermark for %s/PR#%d: %w", projectID, prNumber, err)
	}
	return commentID, nil
}

// SetCommentWatermark sets the last processed comment ID for a project's PR.
// Creates the watermark if it doesn't exist, or updates it if it does.
func (s *SQLiteStore) SetCommentWatermark(ctx context.Context, projectID string, prNumber int64, commentID int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO comment_watermarks (project_id, pr_number, last_processed_comment_id, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(project_id, pr_number) DO UPDATE SET
		     last_processed_comment_id = excluded.last_processed_comment_id,
		     updated_at = excluded.updated_at`,
		projectID, prNumber, commentID, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("set comment watermark for %s/PR#%d: %w", projectID, prNumber, err)
	}
	return nil
}

// TryAcquireSlot atomically attempts to acquire a concurrency slot for a project.
// If the project has no row, one is created with the given limit as the initial slot_limit.
// Returns true if a slot was acquired, false if the project is at capacity.
func (s *SQLiteStore) TryAcquireSlot(ctx context.Context, projectID string, limit int) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("acquire slot for %s: %w", projectID, err)
	}
	defer tx.Rollback()

	// Upsert: create project row if it doesn't exist
	_, err = tx.ExecContext(ctx,
		`INSERT INTO concurrency_slots (project_id, active_count, slot_limit)
		 VALUES (?, 0, ?)
		 ON CONFLICT(project_id) DO NOTHING`,
		projectID, limit,
	)
	if err != nil {
		return false, fmt.Errorf("upsert slot for %s: %w", projectID, err)
	}

	// Atomic check-and-increment
	result, err := tx.ExecContext(ctx,
		`UPDATE concurrency_slots
		 SET active_count = active_count + 1
		 WHERE project_id = ? AND active_count < slot_limit`,
		projectID,
	)
	if err != nil {
		return false, fmt.Errorf("increment slot for %s: %w", projectID, err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("check slot result for %s: %w", projectID, err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit slot acquisition for %s: %w", projectID, err)
	}

	return n > 0, nil
}

// ReleaseSlot decrements the active count for a project, clamping to zero.
// No-op if the project doesn't exist or active count is already zero.
func (s *SQLiteStore) ReleaseSlot(ctx context.Context, projectID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE concurrency_slots
		 SET active_count = MAX(active_count - 1, 0)
		 WHERE project_id = ?`,
		projectID,
	)
	if err != nil {
		return fmt.Errorf("release slot for %s: %w", projectID, err)
	}
	return nil
}

// ListSlots returns all concurrency slot records.
// Returns an empty slice (not nil) when no slots exist.
func (s *SQLiteStore) ListSlots(ctx context.Context) ([]ConcurrencySlot, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT project_id, active_count, slot_limit FROM concurrency_slots`,
	)
	if err != nil {
		return nil, fmt.Errorf("list slots: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var slots []ConcurrencySlot
	for rows.Next() {
		var slot ConcurrencySlot
		if err := rows.Scan(&slot.ProjectID, &slot.ActiveCount, &slot.SlotLimit); err != nil {
			return nil, fmt.Errorf("scan slot row: %w", err)
		}
		slots = append(slots, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate slot rows: %w", err)
	}
	if slots == nil {
		return []ConcurrencySlot{}, nil
	}
	return slots, nil
}

// ResetAllSlots sets all concurrency slot active counts to zero.
// Returns the number of projects that had active slots reset.
func (s *SQLiteStore) ResetAllSlots(ctx context.Context) (int, error) {
	result, err := s.db.ExecContext(ctx,
		`UPDATE concurrency_slots SET active_count = 0 WHERE active_count > 0`,
	)
	if err != nil {
		return 0, fmt.Errorf("reset all slots: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("reset all slots: %w", err)
	}
	return int(n), nil
}
