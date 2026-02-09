package storage

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrInitiativeNotFound = errors.New("initiative not found")

type InitiativeType string

const (
	InitiativeTypeFeature  InitiativeType = "feature"
	InitiativeTypeBug      InitiativeType = "bug"
	InitiativeTypeRefactor InitiativeType = "refactor"
)

type InitiativeStatus string

const (
	InitiativeStatusInProgress InitiativeStatus = "in_progress"
	InitiativeStatusCompleted  InitiativeStatus = "completed"
)

type StepStatus string

const (
	StepStatusPending    StepStatus = "pending"
	StepStatusInProgress StepStatus = "in_progress"
	StepStatusCompleted  StepStatus = "completed"
	StepStatusSkipped    StepStatus = "skipped"
)

type WorkflowStep struct {
	Name      string     `json:"name"`
	Status    StepStatus `json:"status"`
	UpdatedAt time.Time  `json:"updated_at,omitempty"`
}

type Initiative struct {
	ID          uuid.UUID
	Name        string
	Type        InitiativeType
	Status      InitiativeStatus
	Description string
	BranchName  string
	Steps       []WorkflowStep
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type InitiativeStorage interface {
	Create(ctx context.Context, init *Initiative) error
	Get(ctx context.Context, id uuid.UUID) (*Initiative, error)
	Update(ctx context.Context, init *Initiative) error
	List(ctx context.Context, statusFilter *InitiativeStatus, limit, offset int) ([]*Initiative, error)
}

type PostgresInitiativeStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresInitiativeStorage(pool *pgxpool.Pool) *PostgresInitiativeStorage {
	return &PostgresInitiativeStorage{pool: pool}
}

func (s *PostgresInitiativeStorage) Create(ctx context.Context, init *Initiative) error {
	if init.ID == uuid.Nil {
		init.ID = uuid.New()
	}
	now := time.Now()
	init.CreatedAt = now
	init.UpdatedAt = now

	if init.Steps == nil {
		init.Steps = []WorkflowStep{}
	}

	stepsJSON, err := json.Marshal(init.Steps)
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO initiatives (id, name, type, status, description, branch_name, steps, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, init.ID, init.Name, init.Type, init.Status, init.Description, init.BranchName, stepsJSON, init.CreatedAt, init.UpdatedAt)

	return err
}

func (s *PostgresInitiativeStorage) Get(ctx context.Context, id uuid.UUID) (*Initiative, error) {
	var init Initiative
	var stepsJSON []byte

	err := s.pool.QueryRow(ctx, `
		SELECT id, name, type, status, description, branch_name, steps, created_at, updated_at
		FROM initiatives
		WHERE id = $1
	`, id).Scan(
		&init.ID, &init.Name, &init.Type, &init.Status, &init.Description, &init.BranchName, &stepsJSON, &init.CreatedAt, &init.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrInitiativeNotFound
		}
		return nil, err
	}

	if len(stepsJSON) > 0 {
		if err := json.Unmarshal(stepsJSON, &init.Steps); err != nil {
			return nil, err
		}
	}

	return &init, nil
}

func (s *PostgresInitiativeStorage) Update(ctx context.Context, init *Initiative) error {
	init.UpdatedAt = time.Now()

	stepsJSON, err := json.Marshal(init.Steps)
	if err != nil {
		return err
	}

	result, err := s.pool.Exec(ctx, `
		UPDATE initiatives
		SET name = $2, type = $3, status = $4, description = $5, branch_name = $6, steps = $7, updated_at = $8
		WHERE id = $1
	`, init.ID, init.Name, init.Type, init.Status, init.Description, init.BranchName, stepsJSON, init.UpdatedAt)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrInitiativeNotFound
	}

	return nil
}

func (s *PostgresInitiativeStorage) List(ctx context.Context, statusFilter *InitiativeStatus, limit, offset int) ([]*Initiative, error) {
	if limit <= 0 {
		limit = 50
	}

	var rows pgx.Rows
	var err error

	if statusFilter != nil {
		rows, err = s.pool.Query(ctx, `
			SELECT id, name, type, status, description, branch_name, steps, created_at, updated_at
			FROM initiatives
			WHERE status = $1
			ORDER BY created_at DESC
			LIMIT $2 OFFSET $3
		`, *statusFilter, limit, offset)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id, name, type, status, description, branch_name, steps, created_at, updated_at
			FROM initiatives
			ORDER BY created_at DESC
			LIMIT $1 OFFSET $2
		`, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var initiatives []*Initiative
	for rows.Next() {
		var init Initiative
		var stepsJSON []byte
		if err := rows.Scan(
			&init.ID, &init.Name, &init.Type, &init.Status, &init.Description, &init.BranchName, &stepsJSON, &init.CreatedAt, &init.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if len(stepsJSON) > 0 {
			if err := json.Unmarshal(stepsJSON, &init.Steps); err != nil {
				return nil, err
			}
		}
		initiatives = append(initiatives, &init)
	}

	return initiatives, rows.Err()
}
