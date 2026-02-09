package storage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProfileNotFound = errors.New("profile not found")

type ProfileLocation string

const (
	ProfileLocationLocal  ProfileLocation = "local"
	ProfileLocationGlobal ProfileLocation = "global"
)

type Profile struct {
	ID           uuid.UUID
	Name         string
	Content      string
	Domains      []string
	Dependencies []string
	Location     ProfileLocation
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type ProfileStorage interface {
	Get(ctx context.Context, name string) (*Profile, error)
	List(ctx context.Context) ([]*Profile, error)
	Save(ctx context.Context, profile *Profile) error
	Delete(ctx context.Context, name string) error
}

type PostgresProfileStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresProfileStorage(pool *pgxpool.Pool) *PostgresProfileStorage {
	return &PostgresProfileStorage{pool: pool}
}

func (s *PostgresProfileStorage) Get(ctx context.Context, name string) (*Profile, error) {
	var p Profile
	var domains, deps []string

	err := s.pool.QueryRow(ctx, `
		SELECT id, name, content, domains, dependencies, location, created_at, updated_at
		FROM profiles
		WHERE name = $1
	`, name).Scan(
		&p.ID, &p.Name, &p.Content, &domains, &deps, &p.Location, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}

	p.Domains = domains
	p.Dependencies = deps
	return &p, nil
}

func (s *PostgresProfileStorage) List(ctx context.Context) ([]*Profile, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, content, domains, dependencies, location, created_at, updated_at
		FROM profiles
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []*Profile
	for rows.Next() {
		var p Profile
		var domains, deps []string
		if err := rows.Scan(
			&p.ID, &p.Name, &p.Content, &domains, &deps, &p.Location, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, err
		}
		p.Domains = domains
		p.Dependencies = deps
		profiles = append(profiles, &p)
	}

	return profiles, rows.Err()
}

func (s *PostgresProfileStorage) Save(ctx context.Context, profile *Profile) error {
	now := time.Now()
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
		profile.CreatedAt = now
	}
	profile.UpdatedAt = now

	if profile.Domains == nil {
		profile.Domains = []string{}
	}
	if profile.Dependencies == nil {
		profile.Dependencies = []string{}
	}

	_, err := s.pool.Exec(ctx, `
		INSERT INTO profiles (id, name, content, domains, dependencies, location, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (name) DO UPDATE SET
			content = EXCLUDED.content,
			domains = EXCLUDED.domains,
			dependencies = EXCLUDED.dependencies,
			location = EXCLUDED.location,
			updated_at = EXCLUDED.updated_at
	`, profile.ID, profile.Name, profile.Content, profile.Domains, profile.Dependencies,
		profile.Location, profile.CreatedAt, profile.UpdatedAt)

	return err
}

func (s *PostgresProfileStorage) Delete(ctx context.Context, name string) error {
	result, err := s.pool.Exec(ctx, "DELETE FROM profiles WHERE name = $1", name)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return ErrProfileNotFound
	}
	return nil
}
