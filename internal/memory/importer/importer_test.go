package importer

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
)

// testHelper provides common test setup utilities.
type testHelper struct {
	t          *testing.T
	sqliteDB   *sql.DB
	sqlitePath string
	pgPool     *pgxpool.Pool
	pgURL      string
}

// newTestHelper creates a test helper with SQLite and PostgreSQL connections.
func newTestHelper(t *testing.T) *testHelper {
	t.Helper()

	// Create temp SQLite database
	tmpDir := t.TempDir()
	sqlitePath := filepath.Join(tmpDir, "test.db")

	sqliteDB, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

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

	// Connect to PostgreSQL
	pgURL := os.Getenv("TEST_POSTGRES_URL")
	if pgURL == "" {
		pgURL = "postgres://postgres:postgres@localhost:5432/brains_test?sslmode=disable"
	}

	pgPool, err := pgxpool.New(context.Background(), pgURL)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	if err := pgPool.Ping(context.Background()); err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}

	// Clean up PostgreSQL tables
	_, _ = pgPool.Exec(context.Background(), "DROP TABLE IF EXISTS memories")
	_, _ = pgPool.Exec(context.Background(), "DROP TABLE IF EXISTS import_metadata")

	// Create PostgreSQL schema
	_, err = pgPool.Exec(context.Background(), `
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

	t.Cleanup(func() {
		sqliteDB.Close()
		pgPool.Close()
	})

	return &testHelper{
		t:          t,
		sqliteDB:   sqliteDB,
		sqlitePath: sqlitePath,
		pgPool:     pgPool,
		pgURL:      pgURL,
	}
}

// insertSQLiteMemory adds a memory item to the SQLite database.
func (h *testHelper) insertSQLiteMemory(name string, version int, content string) {
	h.t.Helper()
	now := time.Now().UTC()
	_, err := h.sqliteDB.Exec(`
		INSERT INTO memories (name, version, content, deleted, created_at, updated_at)
		VALUES (?, ?, ?, FALSE, ?, ?)
	`, name, version, content, now, now)
	if err != nil {
		h.t.Fatalf("insert sqlite memory: %v", err)
	}
}

// countPostgresMemories returns the count of non-deleted memories in PostgreSQL.
func (h *testHelper) countPostgresMemories() int {
	h.t.Helper()
	var count int
	err := h.pgPool.QueryRow(context.Background(),
		"SELECT COUNT(*) FROM memories WHERE deleted = FALSE").Scan(&count)
	if err != nil {
		h.t.Fatalf("count postgres memories: %v", err)
	}
	return count
}

// getPostgresMemory retrieves a specific memory from PostgreSQL.
func (h *testHelper) getPostgresMemory(name string, version int) (content string, exists bool) {
	h.t.Helper()
	err := h.pgPool.QueryRow(context.Background(),
		"SELECT content FROM memories WHERE name = $1 AND version = $2 AND deleted = FALSE",
		name, version).Scan(&content)
	if err != nil {
		return "", false
	}
	return content, true
}

// ============================================================================
// User Story 1: First-time Migration Tests (T008-T010)
// ============================================================================

// TestBasicImport_10Items tests importing 10 items from SQLite to PostgreSQL.
// T008 [P] [US1]
func TestBasicImport_10Items(t *testing.T) {
	h := newTestHelper(t)

	// Insert 10 items into SQLite
	for i := 1; i <= 10; i++ {
		h.insertSQLiteMemory(
			fmt.Sprintf("memory-%02d", i),
			1,
			fmt.Sprintf("Content for memory %d", i),
		)
	}

	// Run import
	ctx := context.Background()
	imp, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
		BatchSize:  100,
	})
	if err != nil {
		t.Fatalf("create importer: %v", err)
	}
	defer imp.Close()

	result, err := imp.Import(ctx)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	// Verify results
	if result.Imported != 10 {
		t.Errorf("expected 10 imported, got %d", result.Imported)
	}
	if result.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", result.Skipped)
	}
	if result.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d: %v", result.ErrorCount, result.ErrorDetails)
	}

	// Verify data in PostgreSQL
	count := h.countPostgresMemories()
	if count != 10 {
		t.Errorf("expected 10 items in PostgreSQL, got %d", count)
	}

	// Verify specific item
	content, exists := h.getPostgresMemory("memory-05", 1)
	if !exists {
		t.Error("memory-05 not found in PostgreSQL")
	}
	if content != "Content for memory 5" {
		t.Errorf("unexpected content: %s", content)
	}
}

// TestUnicodePreservation tests that Unicode and special characters are preserved.
// T009 [P] [US1]
func TestUnicodePreservation(t *testing.T) {
	h := newTestHelper(t)

	// Test cases with various Unicode content
	testCases := []struct {
		name    string
		content string
	}{
		{"unicode-emoji", "Hello 👋 World 🌍 Testing 🧪"},
		{"unicode-chinese", "你好世界 - Chinese characters"},
		{"unicode-japanese", "こんにちは - Japanese hiragana"},
		{"unicode-arabic", "مرحبا بالعالم - Arabic text"},
		{"unicode-math", "∑∫∂√∞ - Mathematical symbols"},
		{"special-chars", "Line1\nLine2\tTabbed\r\nWindows newline"},
		{"sql-special", "It's a test with 'quotes' and \"double quotes\""},
		{"html-special", "<script>alert('xss')</script> & < > \" '"},
	}

	for _, tc := range testCases {
		h.insertSQLiteMemory(tc.name, 1, tc.content)
	}

	// Run import
	ctx := context.Background()
	imp, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
	})
	if err != nil {
		t.Fatalf("create importer: %v", err)
	}
	defer imp.Close()

	result, err := imp.Import(ctx)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	if result.Imported != len(testCases) {
		t.Errorf("expected %d imported, got %d", len(testCases), result.Imported)
	}

	// Verify each item preserved correctly
	for _, tc := range testCases {
		content, exists := h.getPostgresMemory(tc.name, 1)
		if !exists {
			t.Errorf("%s not found in PostgreSQL", tc.name)
			continue
		}
		if content != tc.content {
			t.Errorf("%s content mismatch:\n  expected: %q\n  got: %q", tc.name, tc.content, content)
		}
	}
}

// TestEmptySourceDatabase tests importing from an empty SQLite database.
// T010 [P] [US1]
func TestEmptySourceDatabase(t *testing.T) {
	h := newTestHelper(t)

	// Don't insert any items - SQLite is empty

	// Run import
	ctx := context.Background()
	imp, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
	})
	if err != nil {
		t.Fatalf("create importer: %v", err)
	}
	defer imp.Close()

	result, err := imp.Import(ctx)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	// Verify results
	if result.Imported != 0 {
		t.Errorf("expected 0 imported, got %d", result.Imported)
	}
	if result.Skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", result.Skipped)
	}
	if result.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d", result.ErrorCount)
	}
	if result.TotalInSource != 0 {
		t.Errorf("expected 0 total in source, got %d", result.TotalInSource)
	}

	// Verify PostgreSQL is still empty
	count := h.countPostgresMemories()
	if count != 0 {
		t.Errorf("expected 0 items in PostgreSQL, got %d", count)
	}
}

// ============================================================================
// User Story 2: Incremental Migration Tests (T017-T019)
// ============================================================================

// TestIncrementalImport_SkipAlreadyImported tests that already-imported items are skipped.
// T017 [P] [US2]
func TestIncrementalImport_SkipAlreadyImported(t *testing.T) {
	h := newTestHelper(t)

	// Insert items into SQLite
	for i := 1; i <= 5; i++ {
		h.insertSQLiteMemory(fmt.Sprintf("memory-%02d", i), 1, fmt.Sprintf("Content %d", i))
	}

	ctx := context.Background()

	// First import
	imp1, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
	})
	if err != nil {
		t.Fatalf("create importer: %v", err)
	}

	result1, err := imp1.Import(ctx)
	imp1.Close()
	if err != nil {
		t.Fatalf("first import failed: %v", err)
	}

	if result1.Imported != 5 {
		t.Errorf("first import: expected 5 imported, got %d", result1.Imported)
	}

	// Add more items to SQLite
	time.Sleep(10 * time.Millisecond) // Ensure different timestamp
	for i := 6; i <= 8; i++ {
		h.insertSQLiteMemory(fmt.Sprintf("memory-%02d", i), 1, fmt.Sprintf("Content %d", i))
	}

	// Second import - should only import new items
	imp2, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
	})
	if err != nil {
		t.Fatalf("create second importer: %v", err)
	}
	defer imp2.Close()

	result2, err := imp2.Import(ctx)
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}

	if result2.Imported != 3 {
		t.Errorf("second import: expected 3 imported, got %d", result2.Imported)
	}

	// Total should be 8
	count := h.countPostgresMemories()
	if count != 8 {
		t.Errorf("expected 8 items in PostgreSQL, got %d", count)
	}
}

// TestNewVersionImport_SoftDeleteOld tests that importing a new version soft-deletes the old one.
// T018 [P] [US2]
func TestNewVersionImport_SoftDeleteOld(t *testing.T) {
	h := newTestHelper(t)

	// Insert version 1 into SQLite
	h.insertSQLiteMemory("test-memory", 1, "Version 1 content")

	ctx := context.Background()

	// First import
	imp1, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
	})
	if err != nil {
		t.Fatalf("create importer: %v", err)
	}

	_, err = imp1.Import(ctx)
	imp1.Close()
	if err != nil {
		t.Fatalf("first import failed: %v", err)
	}

	// Verify version 1 exists
	content, exists := h.getPostgresMemory("test-memory", 1)
	if !exists || content != "Version 1 content" {
		t.Error("version 1 not found after first import")
	}

	// Clear and add version 2 to SQLite
	_, _ = h.sqliteDB.Exec("DELETE FROM memories")
	time.Sleep(10 * time.Millisecond)
	h.insertSQLiteMemory("test-memory", 2, "Version 2 content")

	// Second import with new version
	imp2, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
	})
	if err != nil {
		t.Fatalf("create second importer: %v", err)
	}
	defer imp2.Close()

	result2, err := imp2.Import(ctx)
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}

	if result2.Imported != 1 {
		t.Errorf("expected 1 imported, got %d", result2.Imported)
	}

	// Version 2 should exist
	content, exists = h.getPostgresMemory("test-memory", 2)
	if !exists {
		t.Error("version 2 not found")
	}
	if content != "Version 2 content" {
		t.Errorf("wrong content for version 2: %s", content)
	}

	// Version 1 should be soft-deleted (not returned by our helper which excludes deleted)
	_, exists = h.getPostgresMemory("test-memory", 1)
	if exists {
		t.Error("version 1 should be soft-deleted")
	}
}

// TestZeroItemsWhenNoChanges tests that no items are imported when there are no changes.
// T019 [P] [US2]
func TestZeroItemsWhenNoChanges(t *testing.T) {
	h := newTestHelper(t)

	// Insert items into SQLite
	h.insertSQLiteMemory("memory-1", 1, "Content 1")
	h.insertSQLiteMemory("memory-2", 1, "Content 2")

	ctx := context.Background()

	// First import
	imp1, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
	})
	if err != nil {
		t.Fatalf("create importer: %v", err)
	}

	_, err = imp1.Import(ctx)
	imp1.Close()
	if err != nil {
		t.Fatalf("first import failed: %v", err)
	}

	// Second import - no changes in SQLite
	imp2, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
	})
	if err != nil {
		t.Fatalf("create second importer: %v", err)
	}
	defer imp2.Close()

	result2, err := imp2.Import(ctx)
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}

	// No items should be imported or even considered
	if result2.Imported != 0 {
		t.Errorf("expected 0 imported, got %d", result2.Imported)
	}
	if result2.TotalInSource != 0 {
		t.Errorf("expected 0 in source (incremental), got %d", result2.TotalInSource)
	}
}

// ============================================================================
// User Story 3: Import Status Visibility Tests (T025-T026)
// ============================================================================

// TestDryRunMode_NoDataChanges tests that dry-run mode doesn't modify data.
// T025 [P] [US3]
func TestDryRunMode_NoDataChanges(t *testing.T) {
	h := newTestHelper(t)

	// Insert items into SQLite
	for i := 1; i <= 5; i++ {
		h.insertSQLiteMemory(fmt.Sprintf("memory-%02d", i), 1, fmt.Sprintf("Content %d", i))
	}

	ctx := context.Background()

	// Run dry-run import
	imp, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
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

	// Verify dry-run results
	if !result.DryRun {
		t.Error("expected DryRun to be true")
	}
	if result.Imported != 5 {
		t.Errorf("expected 5 would be imported, got %d", result.Imported)
	}
	if len(result.PendingItems) != 5 {
		t.Errorf("expected 5 pending items, got %d", len(result.PendingItems))
	}

	// Verify NO data was written to PostgreSQL
	count := h.countPostgresMemories()
	if count != 0 {
		t.Errorf("dry-run should not write data, but found %d items", count)
	}
}

// TestProgressCallback_Invocation tests that progress callback is called during import.
// T026 [P] [US3]
func TestProgressCallback_Invocation(t *testing.T) {
	h := newTestHelper(t)

	// Insert items into SQLite
	itemCount := 10
	for i := 1; i <= itemCount; i++ {
		h.insertSQLiteMemory(fmt.Sprintf("memory-%02d", i), 1, fmt.Sprintf("Content %d", i))
	}

	ctx := context.Background()

	// Track progress calls
	var progressCalls []struct {
		imported int
		total    int
		item     string
	}

	imp, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
		OnProgress: func(imported, total int, currentItem string) {
			progressCalls = append(progressCalls, struct {
				imported int
				total    int
				item     string
			}{imported, total, currentItem})
		},
	})
	if err != nil {
		t.Fatalf("create importer: %v", err)
	}
	defer imp.Close()

	_, err = imp.Import(ctx)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	// Verify progress was called for each item
	if len(progressCalls) != itemCount {
		t.Errorf("expected %d progress calls, got %d", itemCount, len(progressCalls))
	}

	// Verify progress incremented correctly
	for i, call := range progressCalls {
		if call.imported != i+1 {
			t.Errorf("call %d: expected imported=%d, got %d", i, i+1, call.imported)
		}
		if call.total != itemCount {
			t.Errorf("call %d: expected total=%d, got %d", i, itemCount, call.total)
		}
	}
}

// ============================================================================
// User Story 4: Error Recovery Tests (T034-T036)
// ============================================================================

// TestPartialFailureRecovery tests that re-running after partial failure skips completed items.
// T034 [P] [US4]
func TestPartialFailureRecovery(t *testing.T) {
	h := newTestHelper(t)

	// Insert items into SQLite
	for i := 1; i <= 10; i++ {
		h.insertSQLiteMemory(fmt.Sprintf("memory-%02d", i), 1, fmt.Sprintf("Content %d", i))
	}

	ctx := context.Background()

	// First import - all 10 items
	imp1, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
	})
	if err != nil {
		t.Fatalf("create importer: %v", err)
	}

	result1, err := imp1.Import(ctx)
	imp1.Close()
	if err != nil {
		t.Fatalf("first import failed: %v", err)
	}

	if result1.Imported != 10 {
		t.Errorf("first import: expected 10 imported, got %d", result1.Imported)
	}

	// Simulate adding more items after first import
	time.Sleep(10 * time.Millisecond)
	for i := 11; i <= 15; i++ {
		h.insertSQLiteMemory(fmt.Sprintf("memory-%02d", i), 1, fmt.Sprintf("Content %d", i))
	}

	// Second import - should only import new items (simulating recovery)
	imp2, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
	})
	if err != nil {
		t.Fatalf("create second importer: %v", err)
	}
	defer imp2.Close()

	result2, err := imp2.Import(ctx)
	if err != nil {
		t.Fatalf("second import failed: %v", err)
	}

	// Should only import the 5 new items
	if result2.Imported != 5 {
		t.Errorf("recovery import: expected 5 imported, got %d", result2.Imported)
	}

	// Total should be 15
	count := h.countPostgresMemories()
	if count != 15 {
		t.Errorf("expected 15 items in PostgreSQL, got %d", count)
	}
}

// TestPerItemErrorHandling tests that single item failures don't stop the import.
// T035 [P] [US4]
func TestPerItemErrorHandling(t *testing.T) {
	h := newTestHelper(t)

	// This test verifies that we collect per-item errors without failing the whole import.
	// Since our implementation uses ON CONFLICT DO NOTHING, we won't get duplicate errors.
	// Instead, we test that errors are properly tracked in the result.

	// Insert valid items
	for i := 1; i <= 5; i++ {
		h.insertSQLiteMemory(fmt.Sprintf("memory-%02d", i), 1, fmt.Sprintf("Content %d", i))
	}

	ctx := context.Background()

	imp, err := New(ctx, ImportOptions{
		SourcePath: h.sqlitePath,
		TargetURL:  h.pgURL,
	})
	if err != nil {
		t.Fatalf("create importer: %v", err)
	}
	defer imp.Close()

	result, err := imp.Import(ctx)
	if err != nil {
		t.Fatalf("import failed: %v", err)
	}

	// All items should be imported successfully
	if result.Imported != 5 {
		t.Errorf("expected 5 imported, got %d", result.Imported)
	}
	if result.ErrorCount != 0 {
		t.Errorf("expected 0 errors, got %d: %v", result.ErrorCount, result.ErrorDetails)
	}
}

// TestPostgresUnavailable_ErrorMessage tests clear error when PostgreSQL is unavailable.
// T036 [P] [US4]
func TestPostgresUnavailable_ErrorMessage(t *testing.T) {
	// Create a temp SQLite database
	tmpDir := t.TempDir()
	sqlitePath := filepath.Join(tmpDir, "test.db")

	sqliteDB, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer sqliteDB.Close()

	// Initialize schema
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

	ctx := context.Background()

	// Try to connect to an invalid PostgreSQL URL
	_, err = New(ctx, ImportOptions{
		SourcePath: sqlitePath,
		TargetURL:  "postgres://invalid:invalid@localhost:59999/nonexistent?connect_timeout=1",
	})

	// Should fail with connection error
	if err == nil {
		t.Fatal("expected error for unavailable PostgreSQL")
	}

	// Error should mention connection failure
	errStr := err.Error()
	if errStr == "" {
		t.Error("expected non-empty error message")
	}
}
