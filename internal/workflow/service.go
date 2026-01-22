// Package workflow provides services for loading workflow definitions.
// Workflows are entry points for starting work (e.g., "new" workflow that routes to feature/bug/refactor).
package workflow

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Workflow represents a workflow definition.
type Workflow struct {
	Name        string
	Description string
	Content     string // The body content (markdown)
	Path        string // Source path for debugging
	Source      string // "local", "global", or "embedded"
}

// Service provides workflow loading operations.
type Service struct {
	workingDir string
	homeDir    string
}

// Global registry for embedded workflows filesystem
var (
	globalEmbeddedFS fs.FS
	globalEmbeddedMu sync.RWMutex
)

// SetEmbeddedFS registers an embedded filesystem containing default workflows.
func SetEmbeddedFS(fsys fs.FS) {
	globalEmbeddedMu.Lock()
	defer globalEmbeddedMu.Unlock()
	globalEmbeddedFS = fsys
}

// GetEmbeddedFS returns the registered embedded filesystem.
func GetEmbeddedFS() fs.FS {
	globalEmbeddedMu.RLock()
	defer globalEmbeddedMu.RUnlock()
	return globalEmbeddedFS
}

// ResetEmbeddedFS clears the registered embedded filesystem (for testing).
func ResetEmbeddedFS() {
	globalEmbeddedMu.Lock()
	defer globalEmbeddedMu.Unlock()
	globalEmbeddedFS = nil
}

// NewService creates a new workflow Service.
// If workingDir is empty, it uses the current working directory.
func NewService(workingDir string) (*Service, error) {
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	homeDir, _ := os.UserHomeDir()

	return &Service{
		workingDir: workingDir,
		homeDir:    homeDir,
	}, nil
}

// Load loads a workflow by name.
// Resolution order: local (.brains/workflows/) > global (~/.brains/workflows/) > embedded.
func (s *Service) Load(name string) (*Workflow, error) {
	// Try local first
	if wf := s.loadFromDir(filepath.Join(s.workingDir, ".brains", "workflows"), name, "local"); wf != nil {
		return wf, nil
	}

	// Try global
	if s.homeDir != "" {
		if wf := s.loadFromDir(filepath.Join(s.homeDir, ".brains", "workflows"), name, "global"); wf != nil {
			return wf, nil
		}
	}

	// Try embedded
	if wf := s.loadFromEmbedded(name); wf != nil {
		return wf, nil
	}

	return nil, &WorkflowNotFoundError{Name: name}
}

// loadFromDir loads a workflow from a filesystem directory.
func (s *Service) loadFromDir(dir, name, source string) *Workflow {
	filePath := filepath.Join(dir, name+".md")
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil
	}

	wf, err := parseWorkflow(content, name, filePath, source)
	if err != nil {
		return nil
	}

	return wf
}

// loadFromEmbedded loads a workflow from the embedded filesystem.
func (s *Service) loadFromEmbedded(name string) *Workflow {
	globalEmbeddedMu.RLock()
	fsys := globalEmbeddedFS
	globalEmbeddedMu.RUnlock()

	if fsys == nil {
		return nil
	}

	filePath := "workflows/" + name + ".md"
	content, err := fs.ReadFile(fsys, filePath)
	if err != nil {
		return nil
	}

	wf, err := parseWorkflow(content, name, "[embedded]/"+name+".md", "embedded")
	if err != nil {
		return nil
	}

	return wf
}

// parseWorkflow parses workflow content from markdown with YAML frontmatter.
func parseWorkflow(content []byte, name, path, source string) (*Workflow, error) {
	str := string(content)

	// Check for frontmatter
	if !strings.HasPrefix(str, "---") {
		// No frontmatter, treat entire content as body
		return &Workflow{
			Name:    name,
			Content: str,
			Path:    path,
			Source:  source,
		}, nil
	}

	// Find end of frontmatter
	endIndex := strings.Index(str[3:], "---")
	if endIndex == -1 {
		return nil, fmt.Errorf("unterminated frontmatter in %s", path)
	}

	frontmatter := str[3 : endIndex+3]
	body := strings.TrimSpace(str[endIndex+6:])

	// Parse frontmatter
	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &meta); err != nil {
		return nil, fmt.Errorf("parsing frontmatter in %s: %w", path, err)
	}

	// Use filename-derived name if not specified in frontmatter
	workflowName := meta.Name
	if workflowName == "" {
		workflowName = name
	}

	return &Workflow{
		Name:        workflowName,
		Description: meta.Description,
		Content:     body,
		Path:        path,
		Source:      source,
	}, nil
}

// List returns all available workflows from all sources.
// Resolution order: local > global > embedded (higher precedence overwrites).
func (s *Service) List() ([]*Workflow, error) {
	workflowMap := make(map[string]*Workflow)

	// Load in reverse precedence order
	// 3. Embedded (lowest)
	if GetEmbeddedFS() != nil {
		s.loadAllFromEmbedded(workflowMap)
	}

	// 2. Global
	if s.homeDir != "" {
		s.loadAllFromDir(filepath.Join(s.homeDir, ".brains", "workflows"), "global", workflowMap)
	}

	// 1. Local (highest)
	s.loadAllFromDir(filepath.Join(s.workingDir, ".brains", "workflows"), "local", workflowMap)

	// Convert to slice
	workflows := make([]*Workflow, 0, len(workflowMap))
	for _, wf := range workflowMap {
		workflows = append(workflows, wf)
	}

	return workflows, nil
}

func (s *Service) loadAllFromDir(dir, source string, out map[string]*Workflow) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		if wf := s.loadFromDir(dir, name, source); wf != nil {
			out[name] = wf
		}
	}
}

func (s *Service) loadAllFromEmbedded(out map[string]*Workflow) {
	globalEmbeddedMu.RLock()
	fsys := globalEmbeddedFS
	globalEmbeddedMu.RUnlock()

	if fsys == nil {
		return
	}

	entries, err := fs.ReadDir(fsys, "workflows")
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".md")
		if wf := s.loadFromEmbedded(name); wf != nil {
			out[name] = wf
		}
	}
}

// WorkflowNotFoundError is returned when a workflow cannot be found.
type WorkflowNotFoundError struct {
	Name string
}

func (e *WorkflowNotFoundError) Error() string {
	return fmt.Sprintf("workflow %q not found", e.Name)
}
