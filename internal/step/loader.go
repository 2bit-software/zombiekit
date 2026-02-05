package step

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BrainsDir is the name of the brains configuration directory.
const BrainsDir = ".brains"

// StepsDir is the subdirectory name for step definitions.
const StepsDir = "steps"

// Loader loads step definitions from local and global sources.
type Loader struct {
	workDir   string
	globalDir string
}

// NewLoader creates a new step loader for the given working directory.
// If workDir is empty, the current working directory is used.
func NewLoader(workDir string) *Loader {
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	// Default global directory is ~/.brains
	homeDir, _ := os.UserHomeDir()
	globalDir := filepath.Join(homeDir, BrainsDir)

	return &Loader{
		workDir:   workDir,
		globalDir: globalDir,
	}
}

// SetGlobalDir overrides the global directory (useful for testing).
func (l *Loader) SetGlobalDir(dir string) {
	l.globalDir = dir
}

// Get loads a step by name, searching in order: local -> global.
// Returns an error with code UNKNOWN_STEP if the step is not found.
func (l *Loader) Get(name string) (*Step, error) {
	// 1. Try local (.brains/steps/{name}.md)
	localPath := filepath.Join(l.workDir, BrainsDir, StepsDir, name+".md")
	if step, err := l.loadFromFile(localPath, name, SourceLocal); err == nil {
		return step, nil
	}

	// 2. Try global (~/.brains/steps/{name}.md)
	globalPath := filepath.Join(l.globalDir, StepsDir, name+".md")
	if step, err := l.loadFromFile(globalPath, name, SourceGlobal); err == nil {
		return step, nil
	}

	return nil, &StepError{
		Code:    "UNKNOWN_STEP",
		Message: fmt.Sprintf("step '%s' not found", name),
		Hint:    "Steps are loaded from .brains/steps/ (local) or ~/.brains/steps/ (global)",
	}
}

// List returns all available steps from all sources.
// Steps from higher-precedence sources shadow those from lower-precedence sources.
func (l *Loader) List() ([]*Step, error) {
	// Use a map to deduplicate by name (first wins = highest precedence)
	stepMap := make(map[string]*Step)

	// 2. Global (lower precedence)
	globalPath := filepath.Join(l.globalDir, StepsDir)
	if global, err := l.loadAllFromDir(globalPath, SourceGlobal); err == nil {
		for name, step := range global {
			stepMap[name] = step
		}
	}

	// 1. Local (highest precedence)
	localPath := filepath.Join(l.workDir, BrainsDir, StepsDir)
	if local, err := l.loadAllFromDir(localPath, SourceLocal); err == nil {
		for name, step := range local {
			stepMap[name] = step
		}
	}

	// Convert map to slice
	steps := make([]*Step, 0, len(stepMap))
	for _, step := range stepMap {
		steps = append(steps, step)
	}

	return steps, nil
}

// loadFromFile loads a single step from a file path.
func (l *Loader) loadFromFile(path, name string, source StepSource) (*Step, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return ParseStep(content, name, path, source)
}

// loadAllFromDir loads all step definitions from a directory.
func (l *Loader) loadAllFromDir(dirPath string, source StepSource) (map[string]*Step, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	steps := make(map[string]*Step)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		filePath := filepath.Join(dirPath, entry.Name())

		step, err := l.loadFromFile(filePath, name, source)
		if err != nil {
			// Skip files with parse errors
			continue
		}

		steps[name] = step
	}

	return steps, nil
}

// StepError represents an error in step operations with an error code.
type StepError struct {
	Code    string
	Message string
	Hint    string
}

func (e *StepError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("%s: %s (%s)", e.Code, e.Message, e.Hint)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}
