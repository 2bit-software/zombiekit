package rules

import (
	"os"
	"path/filepath"
)

// StatFunc abstracts filesystem existence checks so gate evaluation can
// run against a fake filesystem in tests.
type StatFunc func(string) (os.FileInfo, error)

// GateResolver decides whether a rule's file-existence gates are satisfied
// for a given hook event. It walks from the event's working directory up
// to the enclosing repo root once, so rules gated on `Taskfile.yml` still
// fire when the bash command was issued from a subdirectory.
type GateResolver struct {
	cwd      string
	repoRoot string
	stat     StatFunc
}

// NewGateResolver prepares a resolver rooted at cwd, using the real
// filesystem. The repo root is the nearest ancestor containing a `.git`
// directory; if none exists, the resolver walks only `cwd` itself.
func NewGateResolver(cwd string) *GateResolver {
	return newGateResolver(cwd, os.Stat)
}

// newGateResolver is the testable constructor that accepts an injected
// stat implementation.
func newGateResolver(cwd string, stat StatFunc) *GateResolver {
	absCwd := cwd
	if abs, err := filepath.Abs(cwd); err == nil {
		absCwd = abs
	}
	return &GateResolver{
		cwd:      absCwd,
		repoRoot: findGitRootWithStat(absCwd, stat),
		stat:     stat,
	}
}

// Passes reports whether a rule's file-existence gates are all satisfied
// for the resolver's working directory. A rule with no gates always passes.
func (g *GateResolver) Passes(rule *Rule) bool {
	for _, rel := range rule.RequiresFiles {
		if g.resolve(rel) == "" {
			return false
		}
	}
	for _, rel := range rule.RequiresFilesAbsent {
		if g.resolve(rel) != "" {
			return false
		}
	}
	return true
}

// resolve walks from cwd up to repoRoot looking for rel and returns the
// first absolute path that exists, or empty string if nothing was found.
func (g *GateResolver) resolve(rel string) string {
	current := g.cwd
	for {
		candidate := filepath.Join(current, rel)
		if _, err := g.stat(candidate); err == nil {
			return candidate
		}
		if g.repoRoot != "" && current == g.repoRoot {
			return ""
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}

// findGitRootWithStat locates the nearest ancestor of start that contains
// a `.git` directory, or returns empty string when none exists.
func findGitRootWithStat(start string, stat StatFunc) string {
	current := start
	for {
		if info, err := stat(filepath.Join(current, ".git")); err == nil && info.IsDir() {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			return ""
		}
		current = parent
	}
}
