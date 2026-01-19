package claude

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// ImportLock represents an exclusive lock for import operations.
// Only one import process can hold the lock at a time.
type ImportLock struct {
	file *os.File
}

// DefaultLockPath returns the default lock file path (~/.claude/.zombiekit-import.lock).
func DefaultLockPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home directory: %w", err)
	}
	return filepath.Join(home, ".claude", ".zombiekit-import.lock"), nil
}

// AcquireLock attempts to acquire an exclusive import lock.
// Returns an error if another process holds the lock.
// The lock is automatically released when the process terminates.
func AcquireLock(lockPath string) (*ImportLock, error) {
	// Ensure the directory exists
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create lock directory: %w", err)
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("open lock file: %w", err)
	}

	// Try to acquire exclusive non-blocking lock
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("import already in progress (another process holds the lock)")
	}

	return &ImportLock{file: f}, nil
}

// Release releases the import lock.
// Safe to call multiple times.
func (l *ImportLock) Release() error {
	if l == nil || l.file == nil {
		return nil
	}

	// Unlock the file
	_ = syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)

	// Close the file
	err := l.file.Close()
	l.file = nil
	return err
}
