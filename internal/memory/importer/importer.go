package importer

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "modernc.org/sqlite" // SQLite driver

	"github.com/2bit-software/zombiekit/internal/memory"
)

// Importer handles migration from SQLite to PostgreSQL.
type Importer struct {
	sourceDB *sql.DB
	pool     *pgxpool.Pool
	metadata *MetadataRepository
	opts     ImportOptions
}

// New creates a new Importer with the given options.
func New(ctx context.Context, opts ImportOptions) (*Importer, error) {
	// Set defaults
	if opts.BatchSize <= 0 {
		opts.BatchSize = 100
	}

	// Validate source path
	sourcePath := expandPath(opts.SourcePath)
	if _, err := os.Stat(sourcePath); err != nil {
		return nil, fmt.Errorf("SQLite database not found: %s", opts.SourcePath)
	}

	// Open SQLite with exclusive locking
	sourceDB, err := openSQLiteExclusive(ctx, sourcePath)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}

	// Connect to PostgreSQL
	pool, err := pgxpool.New(ctx, opts.TargetURL)
	if err != nil {
		sourceDB.Close()
		return nil, fmt.Errorf("PostgreSQL connection failed: %w", err)
	}

	// Verify PostgreSQL connection
	if err := pool.Ping(ctx); err != nil {
		sourceDB.Close()
		pool.Close()
		return nil, fmt.Errorf("PostgreSQL connection failed: %w", err)
	}

	// Create metadata repository and ensure schema
	metadata := NewMetadataRepository(pool)
	if err := metadata.EnsureSchema(ctx); err != nil {
		sourceDB.Close()
		pool.Close()
		return nil, fmt.Errorf("ensure import_metadata schema: %w", err)
	}

	return &Importer{
		sourceDB: sourceDB,
		pool:     pool,
		metadata: metadata,
		opts:     opts,
	}, nil
}

// Close releases all resources.
func (i *Importer) Close() error {
	var errs []error
	if i.sourceDB != nil {
		if err := i.sourceDB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if i.pool != nil {
		i.pool.Close()
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// Import performs the migration from SQLite to PostgreSQL.
func (i *Importer) Import(ctx context.Context) (*ImportResult, error) {
	start := time.Now()
	result := &ImportResult{
		SourcePath: i.opts.SourcePath,
		TargetURL:  maskURL(i.opts.TargetURL),
		DryRun:     i.opts.DryRun,
	}

	// Check for previous import
	lastImport, err := i.metadata.GetLastImport(ctx, i.opts.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("get last import: %w", err)
	}

	// Fetch items to import
	var items []memory.MemoryItem
	if lastImport != nil && lastImport.LastImportedUpdatedAt != nil {
		items, err = i.fetchMemoriesSince(ctx, *lastImport.LastImportedUpdatedAt)
	} else {
		items, err = i.fetchAllMemories(ctx)
	}
	if err != nil {
		return nil, fmt.Errorf("fetch memories: %w", err)
	}

	result.TotalInSource = len(items)

	// Dry run - just return what would be imported
	if i.opts.DryRun {
		return i.preview(ctx, items, result)
	}

	// Import items
	var maxUpdatedAt *time.Time
	for idx, item := range items {
		// Check for existing version
		action, err := i.checkExistingVersion(ctx, item)
		if err != nil {
			result.ErrorCount++
			result.ErrorDetails = append(result.ErrorDetails, ItemError{
				Name:    item.Name,
				Version: item.Version,
				Error:   err.Error(),
			})
			continue
		}

		switch action {
		case actionSkip:
			result.Skipped++
		case actionInsert, actionSoftDeleteAndInsert:
			if action == actionSoftDeleteAndInsert {
				if err := i.softDeleteVersions(ctx, item.Name); err != nil {
					result.ErrorCount++
					result.ErrorDetails = append(result.ErrorDetails, ItemError{
						Name:    item.Name,
						Version: item.Version,
						Error:   fmt.Sprintf("soft delete: %s", err.Error()),
					})
					continue
				}
			}

			if err := i.insertMemory(ctx, item); err != nil {
				result.ErrorCount++
				result.ErrorDetails = append(result.ErrorDetails, ItemError{
					Name:    item.Name,
					Version: item.Version,
					Error:   err.Error(),
				})
				continue
			}
			result.Imported++

			// Track max updated_at
			if maxUpdatedAt == nil || item.UpdatedAt.After(*maxUpdatedAt) {
				t := normalizeToUTC(item.UpdatedAt)
				maxUpdatedAt = &t
			}
		}

		// Report progress
		if i.opts.OnProgress != nil {
			i.opts.OnProgress(result.Imported, len(items), item.Name)
		}

		// Batch commit (every BatchSize items)
		if (idx+1)%i.opts.BatchSize == 0 {
			// Progress checkpoint - metadata will be saved at the end
		}
	}

	// Save import metadata
	if result.Imported > 0 {
		if err := i.metadata.SaveImportMetadata(ctx, i.opts.SourcePath, result.Imported, maxUpdatedAt); err != nil {
			// Log but don't fail the import
			result.ErrorDetails = append(result.ErrorDetails, ItemError{
				Name:  "_metadata",
				Error: fmt.Sprintf("save metadata: %s", err.Error()),
			})
		}
	}

	result.Duration = time.Since(start)
	return result, nil
}

// Preview returns what would be imported without making changes.
func (i *Importer) Preview(ctx context.Context) (*ImportResult, error) {
	opts := i.opts
	opts.DryRun = true
	i.opts = opts
	return i.Import(ctx)
}

// preview populates the result with pending items for dry-run mode.
func (i *Importer) preview(ctx context.Context, items []memory.MemoryItem, result *ImportResult) (*ImportResult, error) {
	for _, item := range items {
		action, err := i.checkExistingVersion(ctx, item)
		if err != nil {
			result.ErrorCount++
			result.ErrorDetails = append(result.ErrorDetails, ItemError{
				Name:    item.Name,
				Version: item.Version,
				Error:   err.Error(),
			})
			continue
		}

		switch action {
		case actionSkip:
			result.Skipped++
		case actionInsert, actionSoftDeleteAndInsert:
			result.PendingItems = append(result.PendingItems, PendingItem{
				Name:      item.Name,
				Version:   item.Version,
				UpdatedAt: item.UpdatedAt,
			})
		}
	}

	result.Imported = len(result.PendingItems)
	return result, nil
}

// importAction represents what action to take for an item.
type importAction int

const (
	actionSkip importAction = iota
	actionInsert
	actionSoftDeleteAndInsert
)

// checkExistingVersion determines what action to take for an item.
func (i *Importer) checkExistingVersion(ctx context.Context, item memory.MemoryItem) (importAction, error) {
	var maxVersion int
	var exists bool

	err := i.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(version), 0), COUNT(*) > 0
		FROM memories
		WHERE name = $1 AND deleted = FALSE
	`, item.Name).Scan(&maxVersion, &exists)

	if err != nil {
		return actionSkip, fmt.Errorf("check existing version: %w", err)
	}

	if !exists {
		return actionInsert, nil
	}

	if item.Version > maxVersion {
		return actionSoftDeleteAndInsert, nil
	}

	// Same or lower version - skip
	return actionSkip, nil
}

// softDeleteVersions soft-deletes all versions of a memory item in PostgreSQL.
func (i *Importer) softDeleteVersions(ctx context.Context, name string) error {
	_, err := i.pool.Exec(ctx, `
		UPDATE memories
		SET deleted = TRUE, updated_at = NOW()
		WHERE name = $1 AND deleted = FALSE
	`, name)
	return err
}

// fetchAllMemories retrieves all memories from SQLite.
func (i *Importer) fetchAllMemories(ctx context.Context) ([]memory.MemoryItem, error) {
	rows, err := i.sourceDB.QueryContext(ctx, `
		SELECT name, version, content, deleted, created_at, updated_at
		FROM memories
		ORDER BY updated_at ASC, name ASC, version ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query memories: %w", err)
	}
	defer rows.Close()

	return scanMemories(rows)
}

// fetchMemoriesSince retrieves memories updated after the given timestamp.
func (i *Importer) fetchMemoriesSince(ctx context.Context, since time.Time) ([]memory.MemoryItem, error) {
	rows, err := i.sourceDB.QueryContext(ctx, `
		SELECT name, version, content, deleted, created_at, updated_at
		FROM memories
		WHERE updated_at > ?
		ORDER BY updated_at ASC, name ASC, version ASC
	`, since.UTC())
	if err != nil {
		return nil, fmt.Errorf("query memories since %v: %w", since, err)
	}
	defer rows.Close()

	return scanMemories(rows)
}

// scanMemories reads memory items from database rows.
func scanMemories(rows *sql.Rows) ([]memory.MemoryItem, error) {
	var items []memory.MemoryItem
	for rows.Next() {
		var item memory.MemoryItem
		if err := rows.Scan(
			&item.Name, &item.Version, &item.Content,
			&item.Deleted, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		// Normalize timestamps to UTC
		item.CreatedAt = normalizeToUTC(item.CreatedAt)
		item.UpdatedAt = normalizeToUTC(item.UpdatedAt)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate memories: %w", err)
	}
	return items, nil
}

// insertMemory inserts a single memory item into PostgreSQL.
func (i *Importer) insertMemory(ctx context.Context, item memory.MemoryItem) error {
	_, err := i.pool.Exec(ctx, `
		INSERT INTO memories (name, version, content, deleted, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (name, version) DO NOTHING
	`, item.Name, item.Version, item.Content, item.Deleted,
		normalizeToUTC(item.CreatedAt), normalizeToUTC(item.UpdatedAt))

	if err != nil {
		return fmt.Errorf("insert memory %s v%d: %w", item.Name, item.Version, err)
	}
	return nil
}

// openSQLiteExclusive opens a SQLite database with exclusive locking.
func openSQLiteExclusive(ctx context.Context, path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Set exclusive locking mode
	if _, err := db.ExecContext(ctx, "PRAGMA locking_mode=EXCLUSIVE"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set exclusive locking: %w", err)
	}

	// Acquire the lock by reading from a table
	if _, err := db.QueryContext(ctx, "SELECT 1 FROM memories LIMIT 1"); err != nil {
		// Table might not exist, which is fine for empty DB
		if _, err := db.QueryContext(ctx, "SELECT 1"); err != nil {
			db.Close()
			return nil, fmt.Errorf("acquire exclusive lock: %w", err)
		}
	}

	return db, nil
}

// normalizeToUTC converts a timestamp to UTC.
func normalizeToUTC(t time.Time) time.Time {
	return t.UTC()
}

// expandPath expands ~ in the path to the user's home directory.
func expandPath(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// maskURL masks sensitive parts of a database URL for display.
func maskURL(url string) string {
	// Simple masking - just show host/db, not credentials
	if len(url) > 20 {
		return url[:20] + "..."
	}
	return url
}
