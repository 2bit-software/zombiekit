package storage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrArtifactNotFound = errors.New("artifact not found")

type Artifact struct {
	ID           uuid.UUID
	InitiativeID uuid.UUID
	Path         string
	Content      []byte
	ContentType  string
	SizeBytes    int64
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ArtifactStorage interface {
	Get(ctx context.Context, initiativeID uuid.UUID, path string) (*Artifact, error)
	Save(ctx context.Context, artifact *Artifact) error
	List(ctx context.Context, initiativeID uuid.UUID, pathPrefix string) ([]*Artifact, error)
}

type PostgresArtifactStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresArtifactStorage(pool *pgxpool.Pool) *PostgresArtifactStorage {
	return &PostgresArtifactStorage{pool: pool}
}

func (s *PostgresArtifactStorage) Get(ctx context.Context, initiativeID uuid.UUID, path string) (*Artifact, error) {
	var a Artifact
	err := s.pool.QueryRow(ctx, `
		SELECT id, initiative_id, path, content, content_type, size_bytes, created_at, updated_at
		FROM artifacts
		WHERE initiative_id = $1 AND path = $2
	`, initiativeID, path).Scan(
		&a.ID, &a.InitiativeID, &a.Path, &a.Content, &a.ContentType, &a.SizeBytes, &a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrArtifactNotFound
		}
		return nil, err
	}
	return &a, nil
}

func (s *PostgresArtifactStorage) Save(ctx context.Context, artifact *Artifact) error {
	if artifact.ID == uuid.Nil {
		artifact.ID = uuid.New()
	}
	now := time.Now()
	artifact.SizeBytes = int64(len(artifact.Content))
	artifact.UpdatedAt = now

	_, err := s.pool.Exec(ctx, `
		INSERT INTO artifacts (id, initiative_id, path, content, content_type, size_bytes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (initiative_id, path) DO UPDATE SET
			content = EXCLUDED.content,
			content_type = EXCLUDED.content_type,
			size_bytes = EXCLUDED.size_bytes,
			updated_at = EXCLUDED.updated_at
	`, artifact.ID, artifact.InitiativeID, artifact.Path, artifact.Content, artifact.ContentType, artifact.SizeBytes, now, artifact.UpdatedAt)

	return err
}

func (s *PostgresArtifactStorage) List(ctx context.Context, initiativeID uuid.UUID, pathPrefix string) ([]*Artifact, error) {
	query := `
		SELECT id, initiative_id, path, content_type, size_bytes, created_at, updated_at
		FROM artifacts
		WHERE initiative_id = $1
	`
	args := []any{initiativeID}

	if pathPrefix != "" {
		query += " AND path LIKE $2 || '%'"
		args = append(args, pathPrefix)
	}
	query += " ORDER BY path"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var artifacts []*Artifact
	for rows.Next() {
		var a Artifact
		if err := rows.Scan(
			&a.ID, &a.InitiativeID, &a.Path, &a.ContentType, &a.SizeBytes, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		artifacts = append(artifacts, &a)
	}

	return artifacts, rows.Err()
}
