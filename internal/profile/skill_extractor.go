package profile

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractPendingSkills scans profilesDir for *.skill ZIP files and extracts any
// whose target subdirectory does not yet exist. Returns all errors encountered
// (non-fatal — caller should decide how to handle them).
func ExtractPendingSkills(profilesDir string) []error {
	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return []error{fmt.Errorf("reading profiles dir %s: %w", profilesDir, err)}
	}

	var errs []error
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".skill") {
			continue
		}

		skillName := strings.TrimSuffix(entry.Name(), ".skill")
		targetDir := filepath.Join(profilesDir, skillName)

		if _, statErr := os.Stat(targetDir); statErr == nil {
			continue // already extracted
		}

		skillPath := filepath.Join(profilesDir, entry.Name())
		if extractErr := ExtractSkillFile(skillPath, targetDir); extractErr != nil {
			errs = append(errs, fmt.Errorf("%s: %w", entry.Name(), extractErr))
		}
	}
	return errs
}

// ExtractSkillFile extracts a .skill ZIP to targetDir.
// Uses a temp directory and atomic rename for safety.
// Returns an error if the ZIP contains no SKILL.md.
func ExtractSkillFile(skillPath, targetDir string) error {
	r, err := zip.OpenReader(skillPath)
	if err != nil {
		return fmt.Errorf("opening zip: %w", err)
	}
	defer r.Close()

	prefix := detectTopLevelPrefix(r.File)

	tmpDir, err := os.MkdirTemp(filepath.Dir(targetDir), filepath.Base(targetDir)+".tmp.")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}

	if err := extractZipFiles(r.File, tmpDir, prefix, targetDir); err != nil {
		os.RemoveAll(tmpDir)
		return err
	}

	if err := validateHasSkillMD(tmpDir); err != nil {
		os.RemoveAll(tmpDir)
		return err
	}

	if err := os.Rename(tmpDir, targetDir); err != nil {
		os.RemoveAll(tmpDir)
		return fmt.Errorf("finalizing extraction: %w", err)
	}

	return nil
}

// detectTopLevelPrefix returns the single common top-level directory prefix
// shared by all non-empty ZIP entries (e.g. "epic-planner/"). Returns "" if
// entries are flat or if multiple top-level directories exist.
func detectTopLevelPrefix(files []*zip.File) string {
	var first string
	for _, f := range files {
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) < 2 || parts[1] == "" {
			// Root-level file or bare directory entry — no single prefix
			if parts[0] != "" && !strings.HasSuffix(f.Name, "/") {
				return ""
			}
			continue
		}
		if first == "" {
			first = parts[0]
		} else if parts[0] != first {
			return ""
		}
	}
	if first != "" {
		return first + "/"
	}
	return ""
}

func validateHasSkillMD(dir string) error {
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); os.IsNotExist(err) {
		return fmt.Errorf("no SKILL.md found in skill archive")
	}
	return nil
}

func extractZipFiles(files []*zip.File, tmpDir, prefix, targetDir string) error {
	// Zip-slip check is relative to tmpDir (where files are actually written).
	targetAbs, err := filepath.Abs(tmpDir)
	if err != nil {
		return fmt.Errorf("resolving target path: %w", err)
	}
	_ = targetDir // retained in signature for clarity; unused after slip-check fix

	for _, f := range files {
		name := strings.TrimPrefix(f.Name, prefix)
		if name == "" || name == "/" {
			continue
		}

		destPath := filepath.Join(tmpDir, filepath.FromSlash(name))
		destAbs, err := filepath.Abs(destPath)
		if err != nil {
			return fmt.Errorf("resolving path %s: %w", name, err)
		}

		// Zip-slip mitigation: destination must be under targetDir
		rel, err := filepath.Rel(targetAbs, destAbs)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("unsafe path in zip: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			if mkErr := os.MkdirAll(destPath, f.Mode()); mkErr != nil {
				return fmt.Errorf("creating dir %s: %w", name, mkErr)
			}
			continue
		}

		if mkErr := os.MkdirAll(filepath.Dir(destPath), 0o755); mkErr != nil {
			return fmt.Errorf("creating parent for %s: %w", name, mkErr)
		}

		if writeErr := writeZipEntry(f, destPath); writeErr != nil {
			return fmt.Errorf("writing %s: %w", name, writeErr)
		}
	}
	return nil
}

func writeZipEntry(f *zip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}
