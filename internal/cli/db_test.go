package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func runDBCmd(t *testing.T, dbPath string, args ...string) (string, error) {
	t.Helper()

	var buf bytes.Buffer
	app := &cli.App{
		Name:   "brains",
		Writer: &buf,
		Commands: []*cli.Command{
			newDBCommand(),
		},
	}

	// Set the db-path in env for the command to pick up
	os.Setenv("BRAINS_SQLITE_PATH", dbPath)
	defer os.Unsetenv("BRAINS_SQLITE_PATH")

	allArgs := append([]string{"brains", "db"}, args...)
	err := app.Run(allArgs)

	return buf.String(), err
}

func TestDBMigrate_FreshDatabase_SQLite(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	output, err := runDBCmd(t, dbPath, "migrate")
	require.NoError(t, err)

	// Should report migrations applied
	assert.Contains(t, strings.ToLower(output), "applied")
}

func TestDBMigrate_AlreadyApplied_NoChange(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Run migrations first time
	_, err := runDBCmd(t, dbPath, "migrate")
	require.NoError(t, err)

	// Run again - should report no pending migrations
	output, err := runDBCmd(t, dbPath, "migrate")
	require.NoError(t, err)

	assert.Contains(t, strings.ToLower(output), "no pending")
}

func TestDBStatus_ShowsApplied(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Run migrations first
	_, err := runDBCmd(t, dbPath, "migrate")
	require.NoError(t, err)

	// Check status
	output, err := runDBCmd(t, dbPath, "status")
	require.NoError(t, err)

	// Should show stickymemory migration as applied
	assert.Contains(t, output, "stickymemory")
	assert.Contains(t, strings.ToLower(output), "applied")
}

func TestDBStatus_ShowsPending(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Don't run migrations - database doesn't exist yet
	// This should show pending migrations

	output, err := runDBCmd(t, dbPath, "status")
	require.NoError(t, err)

	// Should show stickymemory migration as pending
	assert.Contains(t, output, "stickymemory")
	assert.Contains(t, strings.ToLower(output), "pending")
}

func TestDBStatus_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Run migrations first
	_, err := runDBCmd(t, dbPath, "migrate")
	require.NoError(t, err)

	// Get status in JSON format
	output, err := runDBCmd(t, dbPath, "status", "--format", "json")
	require.NoError(t, err)

	// Should be valid JSON array
	var statuses []interface{}
	err = json.Unmarshal([]byte(output), &statuses)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(statuses), 1)
}

func TestDBStatus_EmptyDatabase(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "empty.db")

	// Create empty database file
	f, err := os.Create(dbPath)
	require.NoError(t, err)
	f.Close()

	// Check status - should show all migrations as pending
	output, err := runDBCmd(t, dbPath, "status")
	require.NoError(t, err)

	assert.Contains(t, strings.ToLower(output), "pending")
}

func TestDBMigrate_CreatesParentDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "subdir", "nested", "test.db")

	// Parent directory doesn't exist - migrate should create it
	output, err := runDBCmd(t, dbPath, "migrate")
	require.NoError(t, err)

	assert.Contains(t, strings.ToLower(output), "applied")

	// Verify database file was created
	_, err = os.Stat(dbPath)
	assert.NoError(t, err)
}
