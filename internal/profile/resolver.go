package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolvedDirectory represents a .brains/profiles/ directory that was found.
type ResolvedDirectory struct {
	Path   string        // Absolute path to the profiles directory
	Source ProfileSource // The source type (local, parent, global)
}

// Resolver finds and loads profiles from .brains/profiles/ directories.
type Resolver struct {
	workingDir string
	homeDir    string
}

// NewResolver creates a new Resolver starting from the given working directory.
// If workingDir is empty, it uses the current working directory.
func NewResolver(workingDir string) (*Resolver, error) {
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("getting current directory: %w", err)
		}
	}

	// Ensure we have an absolute path
	absWorkingDir, err := filepath.Abs(workingDir)
	if err != nil {
		return nil, fmt.Errorf("resolving absolute path: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Home directory inaccessible is not fatal; we'll skip global profiles
		homeDir = ""
	}

	return &Resolver{
		workingDir: absWorkingDir,
		homeDir:    homeDir,
	}, nil
}

// FindProfileDirs finds all .brains/profiles/ directories from the working
// directory up to the git root (or filesystem root), plus the global directory.
// Returns directories in precedence order: local first, parent directories,
// then global last.
func (r *Resolver) FindProfileDirs() ([]ResolvedDirectory, error) {
	dirs := r.walkAncestorProfileDirs()

	if globalDir, ok := r.globalProfileDir(); ok {
		dirs = append(dirs, globalDir)
	}

	return dirs, nil
}

// walkAncestorProfileDirs walks from the working directory up to the git root
// (or filesystem root), collecting any .brains/profiles/ directories found.
func (r *Resolver) walkAncestorProfileDirs() []ResolvedDirectory {
	var dirs []ResolvedDirectory
	gitRoot := r.findGitRoot()
	current := r.workingDir
	isFirst := true

	for {
		profilesPath := filepath.Join(current, ".brains", "profiles")
		if info, err := os.Stat(profilesPath); err == nil && info.IsDir() {
			source := SourceParent
			if isFirst {
				source = SourceLocal
			}
			dirs = append(dirs, ResolvedDirectory{
				Path:   profilesPath,
				Source: source,
			})
		}
		isFirst = false

		if gitRoot != "" && current == gitRoot {
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return dirs
}

// globalProfileDir returns the global profiles directory if it exists.
func (r *Resolver) globalProfileDir() (ResolvedDirectory, bool) {
	if r.homeDir == "" {
		return ResolvedDirectory{}, false
	}
	globalPath := filepath.Join(r.homeDir, ".brains", "profiles")
	info, err := os.Stat(globalPath)
	if err != nil || !info.IsDir() {
		return ResolvedDirectory{}, false
	}
	return ResolvedDirectory{Path: globalPath, Source: SourceGlobal}, true
}

// findGitRoot finds the git repository root by looking for .git directory.
// Returns empty string if not in a git repository.
func (r *Resolver) findGitRoot() string {
	current := r.workingDir
	for {
		gitPath := filepath.Join(current, ".git")
		if info, err := os.Stat(gitPath); err == nil && info.IsDir() {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached filesystem root without finding .git
			return ""
		}
		current = parent
	}
}

// LoadProfiles loads all profile files from the given directories.
// Profiles are keyed by name, with earlier directories taking precedence
// (shadowing later ones).
func (r *Resolver) LoadProfiles(dirs []ResolvedDirectory) (map[string]*Profile, error) {
	profiles := make(map[string]*Profile)

	// Process directories in order (first one wins for each name)
	for _, dir := range dirs {
		dirProfiles, err := r.loadProfilesFromDir(dir)
		if err != nil {
			// Log warning but continue with other directories
			continue
		}

		for name, profile := range dirProfiles {
			if _, exists := profiles[name]; !exists {
				profiles[name] = profile
			}
		}
	}

	return profiles, nil
}

// LoadAllProfiles loads all profiles including shadowed ones.
// Returns profiles grouped by name, with all versions from different sources.
func (r *Resolver) LoadAllProfiles(dirs []ResolvedDirectory) (map[string][]*Profile, error) {
	profiles := make(map[string][]*Profile)

	for _, dir := range dirs {
		dirProfiles, err := r.loadProfilesFromDir(dir)
		if err != nil {
			continue
		}

		for name, profile := range dirProfiles {
			profiles[name] = append(profiles[name], profile)
		}
	}

	return profiles, nil
}

// loadProfilesFromDir loads all profiles from a single profiles directory.
// Supports two layouts: flat name.md files and skill-directory name/SKILL.md.
// Any *.skill ZIP files found are auto-extracted before loading (idempotent).
func (r *Resolver) loadProfilesFromDir(dir ResolvedDirectory) (map[string]*Profile, error) {
	profiles := make(map[string]*Profile)

	// Pass 1: extract any pending .skill ZIPs (idempotent — skips existing dirs).
	// Errors are non-fatal; continue loading whatever is already on disk.
	ExtractPendingSkills(dir.Path) //nolint:errcheck

	// Pass 2: re-read directory to pick up newly extracted subdirs.
	entries, err := os.ReadDir(dir.Path)
	if err != nil {
		return nil, fmt.Errorf("reading directory %s: %w", dir.Path, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			skillDir := filepath.Join(dir.Path, entry.Name())
			if !IsSkillDirectory(skillDir) {
				continue
			}
			p, loadErr := LoadSkillProfile(skillDir, dir.Source)
			if loadErr != nil {
				continue
			}
			// Conflict: directory skill and flat .md share the same name.
			// Directory is processed first; flat .md will be skipped below.
			if _, exists := profiles[p.Name]; exists {
				continue
			}
			profiles[p.Name] = p
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, ".skill") {
			continue // handled by ExtractPendingSkills above
		}
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		// Extract profile name from filename (without .md extension)
		profileName := strings.TrimSuffix(name, ".md")

		// Skip if a skill directory with the same name was already loaded.
		if _, exists := profiles[profileName]; exists {
			continue
		}

		filePath := filepath.Join(dir.Path, name)

		content, err := os.ReadFile(filePath)
		if err != nil {
			// Skip files we can't read
			continue
		}

		profile, err := ParseProfile(content, profileName, filePath, dir.Source)
		if err != nil {
			// Skip profiles with parse errors during loading
			continue
		}

		profiles[profileName] = profile
	}

	return profiles, nil
}

// ResolveProfile finds a specific profile by name from all sources.
// Returns the highest-precedence version of the profile.
func (r *Resolver) ResolveProfile(name string) (*Profile, error) {
	dirs, err := r.FindProfileDirs()
	if err != nil {
		return nil, err
	}

	profiles, err := r.LoadProfiles(dirs)
	if err != nil {
		return nil, err
	}

	profile, exists := profiles[name]
	if !exists {
		return nil, fmt.Errorf("profile %q not found", name)
	}

	return profile, nil
}

// GetInheritanceChain returns all versions of a profile from global to local
// (for inheritance resolution). Returns profiles in order: global first,
// then parent directories from root to CWD, then local.
func (r *Resolver) GetInheritanceChain(name string) ([]*Profile, error) {
	dirs, err := r.FindProfileDirs()
	if err != nil {
		return nil, err
	}

	allProfiles, err := r.LoadAllProfiles(dirs)
	if err != nil {
		return nil, err
	}

	versions := allProfiles[name]
	if len(versions) == 0 {
		return nil, nil
	}

	// Reverse the order: we collected in precedence order (local first),
	// but for inheritance we need global first
	chain := make([]*Profile, len(versions))
	for i, p := range versions {
		chain[len(versions)-1-i] = p
	}

	return chain, nil
}
