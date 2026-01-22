package cli

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadEnvFile_Success(t *testing.T) {
	// Create temp env file
	tmpFile, err := os.CreateTemp("", "test-*.env")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("BRAINS_TEST_VAR_1=hello\nBRAINS_TEST_VAR_2=world\n")
	require.NoError(t, err)
	tmpFile.Close()

	// Clear any existing values
	os.Unsetenv("BRAINS_TEST_VAR_1")
	os.Unsetenv("BRAINS_TEST_VAR_2")
	defer os.Unsetenv("BRAINS_TEST_VAR_1")
	defer os.Unsetenv("BRAINS_TEST_VAR_2")

	// Load the file
	err = loadEnvFile(tmpFile.Name())
	require.NoError(t, err)

	// Verify values were set
	assert.Equal(t, "hello", os.Getenv("BRAINS_TEST_VAR_1"))
	assert.Equal(t, "world", os.Getenv("BRAINS_TEST_VAR_2"))
}

func TestLoadEnvFile_NotOverride(t *testing.T) {
	// Set existing env var
	os.Setenv("BRAINS_TEST_NO_OVERRIDE", "original")
	defer os.Unsetenv("BRAINS_TEST_NO_OVERRIDE")

	// Create env file with different value
	tmpFile, err := os.CreateTemp("", "test-*.env")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString("BRAINS_TEST_NO_OVERRIDE=new_value\n")
	require.NoError(t, err)
	tmpFile.Close()

	// Load should not override existing value
	err = loadEnvFile(tmpFile.Name())
	require.NoError(t, err)
	assert.Equal(t, "original", os.Getenv("BRAINS_TEST_NO_OVERRIDE"))
}

func TestLoadEnvFile_MissingFile(t *testing.T) {
	err := loadEnvFile("/nonexistent/path/to/file.env")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "env file not found")
}

func TestLoadEnvFile_IsDirectory(t *testing.T) {
	// Use temp directory (guaranteed to exist and be a directory)
	tmpDir := t.TempDir()

	err := loadEnvFile(tmpDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "is a directory")
}
