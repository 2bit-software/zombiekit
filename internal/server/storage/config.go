package storage

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type ConfigEntry struct {
	Key       string
	Value     string
	UpdatedAt time.Time
}

type ConfigStorage interface {
	Get(ctx context.Context, keys []string) ([]*ConfigEntry, error)
	Set(ctx context.Context, entries []*ConfigEntry) error
}

type PostgresConfigStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresConfigStorage(pool *pgxpool.Pool) *PostgresConfigStorage {
	return &PostgresConfigStorage{pool: pool}
}

func (s *PostgresConfigStorage) Get(ctx context.Context, keys []string) ([]*ConfigEntry, error) {
	var rows interface {
		Next() bool
		Scan(...any) error
		Close()
		Err() error
	}
	var err error

	if len(keys) == 0 {
		rows, err = s.pool.Query(ctx, `
			SELECT key, value, updated_at
			FROM config
			ORDER BY key
		`)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT key, value, updated_at
			FROM config
			WHERE key = ANY($1)
			ORDER BY key
		`, keys)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*ConfigEntry
	for rows.Next() {
		var e ConfigEntry
		if err := rows.Scan(&e.Key, &e.Value, &e.UpdatedAt); err != nil {
			return nil, err
		}
		entries = append(entries, &e)
	}

	return entries, rows.Err()
}

func (s *PostgresConfigStorage) Set(ctx context.Context, entries []*ConfigEntry) error {
	for _, e := range entries {
		now := time.Now()
		_, err := s.pool.Exec(ctx, `
			INSERT INTO config (key, value, updated_at)
			VALUES ($1, $2, $3)
			ON CONFLICT (key) DO UPDATE SET
				value = EXCLUDED.value,
				updated_at = EXCLUDED.updated_at
		`, e.Key, e.Value, now)
		if err != nil {
			return err
		}
	}
	return nil
}
