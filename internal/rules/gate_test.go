package rules

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// fakeFileInfo is a minimal os.FileInfo used by the fake stat implementation.
type fakeFileInfo struct {
	name string
	dir  bool
}

func (f fakeFileInfo) Name() string       { return f.name }
func (f fakeFileInfo) Size() int64        { return 0 }
func (f fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (f fakeFileInfo) IsDir() bool        { return f.dir }
func (f fakeFileInfo) Sys() any           { return nil }

// fakeFS builds a StatFunc backed by an in-memory path set. Entries ending
// in "/" are treated as directories.
func fakeFS(entries ...string) StatFunc {
	set := make(map[string]bool, len(entries))
	for _, e := range entries {
		set[filepath.Clean(e)] = true
	}
	return func(path string) (os.FileInfo, error) {
		clean := filepath.Clean(path)
		if set[clean] {
			return fakeFileInfo{name: filepath.Base(clean), dir: true}, nil
		}
		return nil, &os.PathError{Op: "stat", Path: path, Err: syscall.ENOENT}
	}
}

func TestGateResolver_NoGates(t *testing.T) {
	g := newGateResolver("/repo", fakeFS("/repo/.git"))
	rule := &Rule{}
	assert.True(t, g.Passes(rule))
}

func TestGateResolver_RequiresFiles_Present(t *testing.T) {
	stat := fakeFS("/repo/.git", "/repo/Taskfile.yml")
	g := newGateResolver("/repo", stat)
	rule := &Rule{RequiresFiles: []string{"Taskfile.yml"}}
	assert.True(t, g.Passes(rule))
}

func TestGateResolver_RequiresFiles_Missing(t *testing.T) {
	stat := fakeFS("/repo/.git")
	g := newGateResolver("/repo", stat)
	rule := &Rule{RequiresFiles: []string{"Taskfile.yml"}}
	assert.False(t, g.Passes(rule))
}

func TestGateResolver_RequiresFilesAbsent_Absent(t *testing.T) {
	stat := fakeFS("/repo/.git")
	g := newGateResolver("/repo", stat)
	rule := &Rule{RequiresFilesAbsent: []string{"Taskfile.yml"}}
	assert.True(t, g.Passes(rule))
}

func TestGateResolver_RequiresFilesAbsent_Present(t *testing.T) {
	stat := fakeFS("/repo/.git", "/repo/Taskfile.yml")
	g := newGateResolver("/repo", stat)
	rule := &Rule{RequiresFilesAbsent: []string{"Taskfile.yml"}}
	assert.False(t, g.Passes(rule))
}

func TestGateResolver_BothGates_BothSatisfied(t *testing.T) {
	stat := fakeFS("/repo/.git", "/repo/Taskfile.yml")
	g := newGateResolver("/repo", stat)
	rule := &Rule{
		RequiresFiles:       []string{"Taskfile.yml"},
		RequiresFilesAbsent: []string{"go.mod"},
	}
	assert.True(t, g.Passes(rule))
}

func TestGateResolver_BothGates_AbsentGateFails(t *testing.T) {
	stat := fakeFS("/repo/.git", "/repo/Taskfile.yml", "/repo/go.mod")
	g := newGateResolver("/repo", stat)
	rule := &Rule{
		RequiresFiles:       []string{"Taskfile.yml"},
		RequiresFilesAbsent: []string{"go.mod"},
	}
	assert.False(t, g.Passes(rule))
}

func TestGateResolver_WalksUpToRepoRoot(t *testing.T) {
	// Taskfile lives at repo root; event cwd is two dirs deep.
	stat := fakeFS("/repo/.git", "/repo/Taskfile.yml")
	g := newGateResolver("/repo/pkg/sub", stat)
	rule := &Rule{RequiresFiles: []string{"Taskfile.yml"}}
	assert.True(t, g.Passes(rule))
}

func TestGateResolver_DoesNotWalkAboveRepoRoot(t *testing.T) {
	// File exists above the repo root — must not be found.
	stat := fakeFS("/repo/.git", "/Taskfile.yml")
	g := newGateResolver("/repo/pkg", stat)
	rule := &Rule{RequiresFiles: []string{"Taskfile.yml"}}
	assert.False(t, g.Passes(rule))
}

func TestGateResolver_MultipleRequiredFiles_AllMustExist(t *testing.T) {
	stat := fakeFS("/repo/.git", "/repo/Taskfile.yml")
	g := newGateResolver("/repo", stat)
	rule := &Rule{RequiresFiles: []string{"Taskfile.yml", "go.mod"}}
	assert.False(t, g.Passes(rule))
}
