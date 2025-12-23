package importer

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ImportMetadata tracks import history for incremental imports.
type ImportMetadata struct {
	// ID is the unique identifier.
	ID int `json:"id"`

	// SourcePathHash is the SHA256 hash of the absolute SQLite path.
	SourcePathHash string `json:"source_path_hash"`

	// SourcePath is the original path for display/logging.
	SourcePath string `json:"source_path"`

	// LastImportAt is when the last import completed.
	LastImportAt time.Time `json:"last_import_at"`

	// LastImportedUpdatedAt is the max updated_at from source at import time.
	LastImportedUpdatedAt *time.Time `json:"last_imported_updated_at,omitempty"`

	// ItemsImported is the total items imported in the last run.
	ItemsImported int `json:"items_imported"`

	// CreatedAt is when this record was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this record was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// MetadataRepository manages import metadata in PostgreSQL.
type MetadataRepository struct {
	pool *pgxpool.Pool
}

// NewMetadataRepository creates a new metadata repository.
func NewMetadataRepository(pool *pgxpool.Pool) *MetadataRepository {
	return &MetadataRepository{pool: pool}
}

// EnsureSchema creates the import_metadata table if it doesn't exist.
func (r *MetadataRepository) EnsureSchema(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS import_metadata (
			id SERIAL PRIMARY KEY,
			source_path_hash TEXT NOT NULL UNIQUE,
			source_path TEXT NOT NULL,
			last_import_at TIMESTAMPTZ NOT NULL,
			last_imported_updated_at TIMESTAMPTZ,
			items_imported INTEGER NOT NULL DEFAULT 0,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("create import_metadata table: %w", err)
	}
	return nil
}

// GetLastImport retrieves the last import metadata for a given source path.
// Returns nil if no previous import exists.
func (r *MetadataRepository) GetLastImport(ctx context.Context, sourcePath string) (*ImportMetadata, error) {
	hash := hashSourcePath(sourcePath)

	var meta ImportMetadata
	err := r.pool.QueryRow(ctx, `
		SELECT id, source_path_hash, source_path, last_import_at,
		       last_imported_updated_at, items_imported, created_at, updated_at
		FROM import_metadata
		WHERE source_path_hash = $1
	`, hash).Scan(
		&meta.ID, &meta.SourcePathHash, &meta.SourcePath,
		&meta.LastImportAt, &meta.LastImportedUpdatedAt,
		&meta.ItemsImported, &meta.CreatedAt, &meta.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query import metadata: %w", err)
	}

	return &meta, nil
}

// SaveImportMetadata creates or updates import metadata for a source path.
func (r *MetadataRepository) SaveImportMetadata(ctx context.Context, sourcePath string, itemsImported int, maxUpdatedAt *time.Time) error {
	hash := hashSourcePath(sourcePath)
	now := time.Now().UTC()

	_, err := r.pool.Exec(ctx, `
		INSERT INTO import_metadata (source_path_hash, source_path, last_import_at, last_imported_updated_at, items_imported, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT (source_path_hash) DO UPDATE SET
			source_path = EXCLUDED.source_path,
			last_import_at = EXCLUDED.last_import_at,
			last_imported_updated_at = EXCLUDED.last_imported_updated_at,
			items_imported = EXCLUDED.items_imported,
			updated_at = EXCLUDED.updated_at
	`, hash, sourcePath, now, maxUpdatedAt, itemsImported, now)

	if err != nil {
		return fmt.Errorf("save import metadata: %w", err)
	}

	return nil
}

// hashSourcePath creates a SHA256 hash of the absolute source path.
func hashSourcePath(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	hash := sha256.Sum256([]byte(absPath))
	return hex.EncodeToString(hash[:])
}
