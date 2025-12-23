// Package integration provides end-to-end integration tests.
package integration

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	_ "modernc.org/sqlite"

	"github.com/zombiekit/brains/internal/memory/importer"
)

// TestImportE2E_FullWorkflow tests the complete import workflow from SQLite to PostgreSQL.
// This test requires a running PostgreSQL instance.
// T043 [P]
func TestImportE2E_FullWorkflow(t *testing.T) {
	// Get PostgreSQL URL from environment or skip
	pgURL := os.Getenv("TEST_POSTGRES_URL")
	if pgURL == "" {
		pgURL = "postgres://postgres:postgres@localhost:5432/brains_test?sslmode=disable"
	}

	// Verify PostgreSQL is available
	ctx := context.Background()
	pgPool, err := pgxpool.New(ctx, pgURL)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	defer pgPool.Close()

	if err := pgPool.Ping(ctx); err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	// Create temp SQLite database
	tmpDir := t.TempDir()
	sqlitePath := filepath.Join(tmpDir, "test.db")

	sqliteDB, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer sqliteDB.Close()

	// Initialize SQLite schema
	_, err = sqliteDB.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			name TEXT NOT NULL,
			version INTEGER NOT NULL,
			content TEXT NOT NULL,
			deleted BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,
			PRIMARY KEY (name, version)
		)
	`)
	if err != nil {
		t.Fatalf("create sqlite schema: %v", err)
	}

	// Clean up PostgreSQL
	_, _ = pgPool.Exec(ctx, "DROP TABLE IF EXISTS memories")
	_, _ = pgPool.Exec(ctx, "DROP TABLE IF EXISTS import_metadata")

	// Create PostgreSQL schema
	_, err = pgPool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS memories (
			name TEXT NOT NULL,
			version INTEGER NOT NULL,
			content TEXT NOT NULL,
			deleted BOOLEAN NOT NULL DEFAULT FALSE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (name, version)
		)
	`)
	if err != nil {
		t.Fatalf("create postgres schema: %v", err)
	}

	// Scenario 1: First-time import
	t.Run("FirstTimeImport", func(t *testing.T) {
		// Insert test data into SQLite
		now := time.Now().UTC()
		for i := 1; i <= 5; i++ {
			_, err := sqliteDB.Exec(`
				INSERT INTO memories (name, version, content, deleted, created_at, updated_at)
				VALUES (?, ?, ?, FALSE, ?, ?)
			`, fmt.Sprintf("memory-%02d", i), 1, fmt.Sprintf("Content %d", i), now, now)
			if err != nil {
				t.Fatalf("insert sqlite memory: %v", err)
			}
		}

		// Run import
		imp, err := importer.New(ctx, importer.ImportOptions{
			SourcePath: sqlitePath,
			TargetURL:  pgURL,
		})
		if err != nil {
			t.Fatalf("create importer: %v", err)
		}
		defer imp.Close()

		result, err := imp.Import(ctx)
		if err != nil {
			t.Fatalf("import failed: %v", err)
		}

		if result.Imported != 5 {
			t.Errorf("expected 5 imported, got %d", result.Imported)
		}
		if result.ErrorCount != 0 {
			t.Errorf("expected 0 errors, got %d: %v", result.ErrorCount, result.ErrorDetails)
		}

		// Verify data in PostgreSQL
		var count int
		err = pgPool.QueryRow(ctx, "SELECT COUNT(*) FROM memories WHERE deleted = FALSE").Scan(&count)
		if err != nil {
			t.Fatalf("count postgres memories: %v", err)
		}
		if count != 5 {
			t.Errorf("expected 5 items in PostgreSQL, got %d", count)
		}
	})

	// Scenario 2: Incremental import
	t.Run("IncrementalImport", func(t *testing.T) {
		// Add more items to SQLite
		time.Sleep(50 * time.Millisecond) // Ensure different timestamp
		now := time.Now().UTC()
		for i := 6; i <= 8; i++ {
			_, err := sqliteDB.Exec(`
				INSERT INTO memories (name, version, content, deleted, created_at, updated_at)
				VALUES (?, ?, ?, FALSE, ?, ?)
			`, fmt.Sprintf("memory-%02d", i), 1, fmt.Sprintf("Content %d", i), now, now)
			if err != nil {
				t.Fatalf("insert sqlite memory: %v", err)
			}
		}

		// Run import again
		imp, err := importer.New(ctx, importer.ImportOptions{
			SourcePath: sqlitePath,
			TargetURL:  pgURL,
		})
		if err != nil {
			t.Fatalf("create importer: %v", err)
		}
		defer imp.Close()

		result, err := imp.Import(ctx)
		if err != nil {
			t.Fatalf("import failed: %v", err)
		}

		// Should only import the 3 new items
		if result.Imported != 3 {
			t.Errorf("expected 3 imported, got %d", result.Imported)
		}

		// Total should now be 8
		var count int
		err = pgPool.QueryRow(ctx, "SELECT COUNT(*) FROM memories WHERE deleted = FALSE").Scan(&count)
		if err != nil {
			t.Fatalf("count postgres memories: %v", err)
		}
		if count != 8 {
			t.Errorf("expected 8 items in PostgreSQL, got %d", count)
		}
	})

	// Scenario 3: Dry-run mode
	t.Run("DryRunMode", func(t *testing.T) {
		// Add more items to SQLite
		time.Sleep(50 * time.Millisecond)
		now := time.Now().UTC()
		for i := 9; i <= 10; i++ {
			_, err := sqliteDB.Exec(`
				INSERT INTO memories (name, version, content, deleted, created_at, updated_at)
				VALUES (?, ?, ?, FALSE, ?, ?)
			`, fmt.Sprintf("memory-%02d", i), 1, fmt.Sprintf("Content %d", i), now, now)
			if err != nil {
				t.Fatalf("insert sqlite memory: %v", err)
			}
		}

		// Count before dry-run
		var countBefore int
		err = pgPool.QueryRow(ctx, "SELECT COUNT(*) FROM memories WHERE deleted = FALSE").Scan(&countBefore)
		if err != nil {
			t.Fatalf("count postgres memories: %v", err)
		}

		// Run dry-run import
		imp, err := importer.New(ctx, importer.ImportOptions{
			SourcePath: sqlitePath,
			TargetURL:  pgURL,
			DryRun:     true,
		})
		if err != nil {
			t.Fatalf("create importer: %v", err)
		}
		defer imp.Close()

		result, err := imp.Import(ctx)
		if err != nil {
			t.Fatalf("dry-run import failed: %v", err)
		}

		if !result.DryRun {
			t.Error("expected DryRun to be true")
		}
		if result.Imported != 2 {
			t.Errorf("expected 2 would be imported, got %d", result.Imported)
		}
		if len(result.PendingItems) != 2 {
			t.Errorf("expected 2 pending items, got %d", len(result.PendingItems))
		}

		// Count should be unchanged
		var countAfter int
		err = pgPool.QueryRow(ctx, "SELECT COUNT(*) FROM memories WHERE deleted = FALSE").Scan(&countAfter)
		if err != nil {
			t.Fatalf("count postgres memories: %v", err)
		}
		if countAfter != countBefore {
			t.Errorf("dry-run should not change data: before=%d, after=%d", countBefore, countAfter)
		}
	})

	// Scenario 4: Version upgrade
	t.Run("VersionUpgrade", func(t *testing.T) {
		// Clear SQLite and add version 2 of existing item
		_, _ = sqliteDB.Exec("DELETE FROM memories")
		time.Sleep(50 * time.Millisecond)
		now := time.Now().UTC()

		_, err := sqliteDB.Exec(`
			INSERT INTO memories (name, version, content, deleted, created_at, updated_at)
			VALUES (?, ?, ?, FALSE, ?, ?)
		`, "memory-01", 2, "Version 2 content", now, now)
		if err != nil {
			t.Fatalf("insert sqlite memory v2: %v", err)
		}

		// Run import
		imp, err := importer.New(ctx, importer.ImportOptions{
			SourcePath: sqlitePath,
			TargetURL:  pgURL,
		})
		if err != nil {
			t.Fatalf("create importer: %v", err)
		}
		defer imp.Close()

		result, err := imp.Import(ctx)
		if err != nil {
			t.Fatalf("import failed: %v", err)
		}

		if result.Imported != 1 {
			t.Errorf("expected 1 imported, got %d", result.Imported)
		}

		// Version 2 should exist and version 1 should be soft-deleted
		var content string
		err = pgPool.QueryRow(ctx, `
			SELECT content FROM memories
			WHERE name = 'memory-01' AND version = 2 AND deleted = FALSE
		`).Scan(&content)
		if err != nil {
			t.Errorf("version 2 not found: %v", err)
		}
		if content != "Version 2 content" {
			t.Errorf("wrong content: %s", content)
		}

		// Version 1 should be soft-deleted
		var deleted bool
		err = pgPool.QueryRow(ctx, `
			SELECT deleted FROM memories
			WHERE name = 'memory-01' AND version = 1
		`).Scan(&deleted)
		if err != nil {
			t.Errorf("version 1 not found: %v", err)
		}
		if !deleted {
			t.Error("version 1 should be soft-deleted")
		}
	})
}
