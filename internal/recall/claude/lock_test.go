package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAcquireLock_Success(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	lock, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}
	defer lock.Release()

	// Lock file should exist
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("lock file should exist")
	}
}

func TestAcquireLock_AlreadyHeld(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	// Acquire first lock
	lock1, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("first AcquireLock failed: %v", err)
	}
	defer lock1.Release()

	// Try to acquire second lock - should fail
	lock2, err := AcquireLock(lockPath)
	if err == nil {
		lock2.Release()
		t.Error("expected error when lock already held")
	}
}

func TestRelease_MultipleCalls(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	lock, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireLock failed: %v", err)
	}

	// First release should succeed
	if err := lock.Release(); err != nil {
		t.Errorf("first Release failed: %v", err)
	}

	// Second release should be safe (no error)
	if err := lock.Release(); err != nil {
		t.Errorf("second Release should be safe: %v", err)
	}
}

func TestRelease_AllowsNewLock(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "test.lock")

	// Acquire and release first lock
	lock1, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("first AcquireLock failed: %v", err)
	}
	lock1.Release()

	// Should be able to acquire a new lock
	lock2, err := AcquireLock(lockPath)
	if err != nil {
		t.Errorf("should be able to acquire lock after release: %v", err)
	}
	if lock2 != nil {
		lock2.Release()
	}
}

func TestAcquireLock_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "nested", "dir", "test.lock")

	lock, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireLock should create parent directories: %v", err)
	}
	defer lock.Release()

	// Directory should exist
	dir := filepath.Dir(lockPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Error("parent directory should exist")
	}
}

func TestDefaultLockPath(t *testing.T) {
	path, err := DefaultLockPath()
	if err != nil {
		t.Fatalf("DefaultLockPath failed: %v", err)
	}

	if path == "" {
		t.Error("lock path should not be empty")
	}

	// Should contain .claude in the path
	if filepath.Base(filepath.Dir(path)) != ".claude" {
		t.Errorf("expected lock in .claude directory, got %s", path)
	}
}

func TestRelease_NilLock(t *testing.T) {
	var lock *ImportLock
	// Should not panic
	if err := lock.Release(); err != nil {
		t.Errorf("Release on nil lock should be safe: %v", err)
	}
}
