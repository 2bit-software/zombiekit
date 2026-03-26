package state

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewSQLiteStore(context.Background(), dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { store.Close() })
	return store
}

func TestNewSQLiteStore_FirstRun(t *testing.T) {
	store := setupTestStore(t)

	expectedTables := []string{"jobs", "comment_watermarks", "concurrency_slots", "schema_migrations"}
	for _, table := range expectedTables {
		var count int
		err := store.DB().QueryRowContext(
			context.Background(),
			"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
			table,
		).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count, "table %s should exist", table)
	}
}

func TestNewSQLiteStore_JobsTableColumns(t *testing.T) {
	store := setupTestStore(t)

	rows, err := store.DB().QueryContext(context.Background(), "PRAGMA table_info(jobs)")
	require.NoError(t, err)
	defer rows.Close()

	columns := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var dflt *string
		err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk)
		require.NoError(t, err)
		columns[name] = true
	}
	require.NoError(t, rows.Err())

	expected := []string{"ticket_id", "worktree_path", "cmux_session", "pr_number", "status", "created_at", "updated_at"}
	for _, col := range expected {
		assert.True(t, columns[col], "jobs table should have column %s", col)
	}
}

func TestNewSQLiteStore_IdempotentRestart(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	ctx := context.Background()

	// First open — creates tables
	store1, err := NewSQLiteStore(ctx, dbPath)
	require.NoError(t, err)

	// Insert a row to verify data survives restart
	_, err = store1.DB().ExecContext(ctx,
		"INSERT INTO jobs (ticket_id, worktree_path, cmux_session, status) VALUES (?, ?, ?, ?)",
		"DEV-999", "/tmp/wt", "session-1", "queued",
	)
	require.NoError(t, err)
	require.NoError(t, store1.Close())

	// Second open — should not error, data preserved
	store2, err := NewSQLiteStore(ctx, dbPath)
	require.NoError(t, err)
	defer store2.Close()

	var ticketID string
	err = store2.DB().QueryRowContext(ctx,
		"SELECT ticket_id FROM jobs WHERE ticket_id = ?", "DEV-999",
	).Scan(&ticketID)
	require.NoError(t, err)
	assert.Equal(t, "DEV-999", ticketID)
}

func TestNewSQLiteStore_EmptyPath(t *testing.T) {
	_, err := NewSQLiteStore(context.Background(), "")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrInvalidDBPath))
}

func TestNewSQLiteStore_UnwritablePath(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("skipping unwritable path test when running as root")
	}

	_, err := NewSQLiteStore(context.Background(), "/root/forbidden/test.db")
	require.Error(t, err)
}

func TestNewSQLiteStore_PragmasSet(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	var journalMode string
	err := store.DB().QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode)
	require.NoError(t, err)
	assert.Equal(t, "wal", journalMode)

	var foreignKeys int
	err = store.DB().QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys)
	require.NoError(t, err)
	assert.Equal(t, 1, foreignKeys)

	var synchronous int
	err = store.DB().QueryRowContext(ctx, "PRAGMA synchronous").Scan(&synchronous)
	require.NoError(t, err)
	assert.Equal(t, 1, synchronous) // NORMAL = 1
}

func TestRunMigrations_Idempotent(t *testing.T) {
	store := setupTestStore(t)
	ctx := context.Background()

	// Migrations already ran in constructor. Run again explicitly.
	err := store.Migrate(ctx)
	require.NoError(t, err)

	// Verify schema_migrations has exactly one entry
	var count int
	err = store.DB().QueryRowContext(ctx,
		"SELECT COUNT(*) FROM schema_migrations",
	).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}
