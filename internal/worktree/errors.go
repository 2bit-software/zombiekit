package worktree

import (
	"errors"
	"strings"
)

// ErrorKind classifies worktree operation errors.
type ErrorKind int

const (
	ErrPathExists     ErrorKind = iota + 1 // worktree path already exists
	ErrBranchExists                        // branch name already in use
	ErrNotAWorktree                        // path is not a known worktree
	ErrWorktreeLocked                      // worktree is locked
	ErrBranchInUse                         // branch has an active worktree
	ErrBranchNotFound                      // branch does not exist
	ErrGitUnavailable                      // git not found on PATH
	ErrNotARepository                      // directory is not a git repository
	ErrGitCommand                          // unexpected git failure
)

// Error represents a worktree operation error with classification.
type Error struct {
	Kind    ErrorKind
	Message string
	Err     error
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }

func newError(kind ErrorKind, msg string, cause error) *Error {
	return &Error{Kind: kind, Message: msg, Err: cause}
}

// stderrClassifications maps git stderr substrings to ErrorKinds.
// Order matters: first match wins (e.g. "already used by worktree" before "already exists").
var stderrClassifications = []struct {
	substr string
	kind   ErrorKind
}{
	{"already used by worktree", ErrBranchExists},
	{"a branch named", ErrBranchExists},
	{"already exists", ErrPathExists},
	{"is not a working tree", ErrNotAWorktree},
	{"cannot remove a locked", ErrWorktreeLocked},
	{"contains modified or untracked", ErrGitCommand},
	{"cannot delete branch", ErrBranchInUse},
	{"not found", ErrBranchNotFound},
}

// classifyError maps git stderr output to an ErrorKind.
func classifyError(stderr string) ErrorKind {
	for _, c := range stderrClassifications {
		if strings.Contains(stderr, c.substr) {
			return c.kind
		}
	}
	return ErrGitCommand
}

// IsPathExists reports whether err is a worktree path-already-exists error.
func IsPathExists(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrPathExists
}

// IsBranchExists reports whether err is a branch-already-exists error.
func IsBranchExists(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrBranchExists
}

// IsNotAWorktree reports whether err indicates the path is not a worktree.
func IsNotAWorktree(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrNotAWorktree
}

// IsWorktreeLocked reports whether err indicates a locked worktree.
func IsWorktreeLocked(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrWorktreeLocked
}

// IsBranchInUse reports whether err indicates the branch has an active worktree.
func IsBranchInUse(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrBranchInUse
}

// IsBranchNotFound reports whether err indicates the branch does not exist.
func IsBranchNotFound(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrBranchNotFound
}

// IsNotARepository reports whether err indicates the dir is not a git repo.
func IsNotARepository(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Kind == ErrNotARepository
}

