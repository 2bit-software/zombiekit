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

// Job represents an autonomous development task tracked by the orchestrator.
type Job struct {
	TicketID     string
	WorktreePath string
	CmuxSession  string
	PRNumber     *int64
	Status       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// StateStore defines the interface for orchestrator persistent state.
type StateStore interface {
	Migrate(ctx context.Context) error
	Close() error

	CreateJob(ctx context.Context, ticketID, worktreePath, cmuxSession string) error
	GetJob(ctx context.Context, ticketID string) (*Job, error)
	SetPR(ctx context.Context, ticketID string, prNumber int64) error

	GetCommentWatermark(ctx context.Context, prNumber int64) (int64, error)
	SetCommentWatermark(ctx context.Context, prNumber int64, commentID int64) error

	TryAcquireSlot(ctx context.Context, projectID string, limit int) (bool, error)
	ReleaseSlot(ctx context.Context, projectID string) error
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
func (s *SQLiteStore) CreateJob(ctx context.Context, ticketID, worktreePath, cmuxSession string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO jobs (ticket_id, worktree_path, cmux_session, status, created_at, updated_at)
		 VALUES (?, ?, ?, 'queued', ?, ?)`,
		ticketID, worktreePath, cmuxSession, now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return fmt.Errorf("create job %s: %w", ticketID, ErrJobExists)
		}
		return fmt.Errorf("create job: %w", err)
	}
	return nil
}

// GetJob retrieves a job by ticket ID.
// Returns nil, nil if no job exists for the given ticket ID.
func (s *SQLiteStore) GetJob(ctx context.Context, ticketID string) (*Job, error) {
	var job Job
	var prNumber sql.NullInt64
	err := s.db.QueryRowContext(ctx,
		`SELECT ticket_id, worktree_path, cmux_session, pr_number, status, created_at, updated_at
		 FROM jobs WHERE ticket_id = ?`,
		ticketID,
	).Scan(&job.TicketID, &job.WorktreePath, &job.CmuxSession, &prNumber, &job.Status, &job.CreatedAt, &job.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get job %s: %w", ticketID, err)
	}
	if prNumber.Valid {
		job.PRNumber = &prNumber.Int64
	}
	return &job, nil
}

// SetPR associates a PR number with an existing job.
// Returns ErrJobNotFound if no job exists for the given ticket ID.
func (s *SQLiteStore) SetPR(ctx context.Context, ticketID string, prNumber int64) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE jobs SET pr_number = ?, updated_at = ? WHERE ticket_id = ?`,
		prNumber, time.Now(), ticketID,
	)
	if err != nil {
		return fmt.Errorf("set PR for job %s: %w", ticketID, err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("set PR for job %s: %w", ticketID, err)
	}
	if n == 0 {
		return fmt.Errorf("set PR for job %s: %w", ticketID, ErrJobNotFound)
	}
	return nil
}

// GetCommentWatermark returns the last processed comment ID for a PR.
// Returns 0 if no watermark has been set for the given PR.
func (s *SQLiteStore) GetCommentWatermark(ctx context.Context, prNumber int64) (int64, error) {
	var commentID int64
	err := s.db.QueryRowContext(ctx,
		`SELECT last_processed_comment_id FROM comment_watermarks WHERE pr_number = ?`,
		prNumber,
	).Scan(&commentID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("get comment watermark for PR %d: %w", prNumber, err)
	}
	return commentID, nil
}

// SetCommentWatermark sets the last processed comment ID for a PR.
// Creates the watermark if it doesn't exist, or updates it if it does.
func (s *SQLiteStore) SetCommentWatermark(ctx context.Context, prNumber int64, commentID int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO comment_watermarks (pr_number, last_processed_comment_id, updated_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(pr_number) DO UPDATE SET
		     last_processed_comment_id = excluded.last_processed_comment_id,
		     updated_at = excluded.updated_at`,
		prNumber, commentID, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("set comment watermark for PR %d: %w", prNumber, err)
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
