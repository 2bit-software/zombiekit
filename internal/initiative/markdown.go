package initiative

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// StepStatus represents the status of a workflow step.
type StepStatus string

const (
	StepPending    StepStatus = "pending"
	StepInProgress StepStatus = "in_progress"
	StepCompleted  StepStatus = "completed"
	StepSkipped    StepStatus = "skipped"
)

// ParsedStep represents a step parsed from the INITIATIVE.md step table.
type ParsedStep struct {
	Name    string     // Step name (e.g., "spec", "plan")
	Profile string     // Profile name(s) for this step (empty for legacy 3-column format)
	Status  StepStatus // Step status
	Updated string     // Timestamp or "-"
}

// ParsedInitiative represents an INITIATIVE.md parsed into structured data.
type ParsedInitiative struct {
	Name    string       // Initiative name slug
	Type    string       // Initiative type ("feature", "bug", "refactor")
	Status  string       // Status from header
	Created time.Time    // Created timestamp
	Steps   []ParsedStep // Steps in this initiative (flat, no cycles)
}

// CurrentStep returns the current in-progress step, or nil if none.
func (p *ParsedInitiative) CurrentStep() *ParsedStep {
	for i := range p.Steps {
		if p.Steps[i].Status == StepInProgress {
			return &p.Steps[i]
		}
	}
	return nil
}

// NextStep returns the next pending step after the current step.
func (p *ParsedInitiative) NextStep() *ParsedStep {
	foundCurrent := false
	for i := range p.Steps {
		if p.Steps[i].Status == StepInProgress {
			foundCurrent = true
			continue
		}
		if foundCurrent && p.Steps[i].Status == StepPending {
			return &p.Steps[i]
		}
	}
	// If no in-progress step, find first pending
	if !foundCurrent {
		for i := range p.Steps {
			if p.Steps[i].Status == StepPending {
				return &p.Steps[i]
			}
		}
	}
	return nil
}

var (
	// Matches 4-column step table rows: | step | profile | status | updated |
	stepRowRe4Col = regexp.MustCompile(`^\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|`)
	// Matches 3-column step table rows: | step | status | updated |
	stepRowRe3Col = regexp.MustCompile(`^\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|$`)
	// Matches header metadata like **Type**: feature
	metadataRe = regexp.MustCompile(`^\*\*(\w+)\*\*:\s*(.+)$`)
	// Matches initiative header: # Initiative: name
	titleRe = regexp.MustCompile(`^#\s+Initiative:\s*(.+)$`)
)

// parseMetadataField applies a parsed metadata key-value pair to the initiative.
func parseMetadataField(parsed *ParsedInitiative, key, value string) {
	switch key {
	case "Type":
		parsed.Type = value
	case "Status":
		parsed.Status = value
	case "Created":
		if t, err := time.Parse("2006-01-02", value); err == nil {
			parsed.Created = t
		} else if t, err := time.Parse(time.RFC3339, value); err == nil {
			parsed.Created = t
		}
	}
}

// parseStepRow attempts to parse a markdown table row as a step.
// Supports both 4-column (Step|Profile|Status|Updated) and 3-column (Step|Status|Updated) formats.
// Returns nil if the line is not a valid step row.
func parseStepRow(line string) *ParsedStep {
	// Try 4-column first: | Step | Profile | Status | Updated |
	if matches := stepRowRe4Col.FindStringSubmatch(line); matches != nil {
		stepName := strings.TrimSpace(matches[1])
		if stepName == "Step" || stepName == "step" {
			return nil
		}
		return &ParsedStep{
			Name:    stepName,
			Profile: strings.TrimSpace(matches[2]),
			Status:  parseStepStatus(strings.TrimSpace(matches[3])),
			Updated: strings.TrimSpace(matches[4]),
		}
	}

	// Fall back to 3-column: | Step | Status | Updated |
	if matches := stepRowRe3Col.FindStringSubmatch(line); matches != nil {
		stepName := strings.TrimSpace(matches[1])
		if stepName == "Step" || stepName == "step" {
			return nil
		}
		return &ParsedStep{
			Name:    stepName,
			Status:  parseStepStatus(strings.TrimSpace(matches[2])),
			Updated: strings.TrimSpace(matches[3]),
		}
	}

	return nil
}

// scanLine processes a single line from the initiative file, updating the parsed state.
// Returns the updated inStepTable flag.
func scanLine(parsed *ParsedInitiative, line string, inStepTable bool) bool {
	if matches := titleRe.FindStringSubmatch(line); matches != nil {
		parsed.Name = strings.TrimSpace(matches[1])
		return inStepTable
	}

	if matches := metadataRe.FindStringSubmatch(line); matches != nil {
		parseMetadataField(parsed, matches[1], strings.TrimSpace(matches[2]))
		return inStepTable
	}

	if strings.HasPrefix(line, "| Step ") {
		return true
	}

	if strings.HasPrefix(line, "|---") || strings.HasPrefix(line, "| ---") {
		return inStepTable
	}

	if !inStepTable {
		return false
	}

	if step := parseStepRow(line); step != nil {
		parsed.Steps = append(parsed.Steps, *step)
		return true
	}

	return line != "" && strings.HasPrefix(line, "|")
}

// ParseInitiativeMD parses an INITIATIVE.md file into structured data.
func ParseInitiativeMD(path string) (*ParsedInitiative, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening initiative file: %w", err)
	}
	defer file.Close()

	parsed := &ParsedInitiative{}
	scanner := bufio.NewScanner(file)
	inStepTable := false

	for scanner.Scan() {
		inStepTable = scanLine(parsed, scanner.Text(), inStepTable)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading initiative file: %w", err)
	}

	return parsed, nil
}

func parseStepStatus(s string) StepStatus {
	switch strings.ToLower(s) {
	case "pending":
		return StepPending
	case "in_progress", "in-progress":
		return StepInProgress
	case "completed", "complete":
		return StepCompleted
	case "skipped":
		return StepSkipped
	default:
		return StepPending
	}
}

// UpdateStepStatus updates the status of a step.
// Returns an error if the step is not found.
func (p *ParsedInitiative) UpdateStepStatus(stepName string, status StepStatus, timestamp string) error {
	for i := range p.Steps {
		if p.Steps[i].Name == stepName {
			p.Steps[i].Status = status
			p.Steps[i].Updated = timestamp
			return nil
		}
	}
	return fmt.Errorf("step not found: %s", stepName)
}

// AddStep inserts a new step after the specified step.
// If afterStep is empty, the step is added at the beginning.
func (p *ParsedInitiative) AddStep(afterStep string, newStep ParsedStep) error {
	if afterStep == "" {
		// Add at the beginning
		p.Steps = append([]ParsedStep{newStep}, p.Steps...)
		return nil
	}
	for i := range p.Steps {
		if p.Steps[i].Name == afterStep {
			// Insert after this step
			newSteps := make([]ParsedStep, 0, len(p.Steps)+1)
			newSteps = append(newSteps, p.Steps[:i+1]...)
			newSteps = append(newSteps, newStep)
			newSteps = append(newSteps, p.Steps[i+1:]...)
			p.Steps = newSteps
			return nil
		}
	}
	return fmt.Errorf("step not found: %s", afterStep)
}

// replaceStepsSection replaces the Steps/Cycles section in the given lines with new step content.
// Returns the resulting lines and whether the section was found.
func replaceStepsSection(lines []string, stepLines []string) ([]string, bool) {
	var result []string
	inStepsSection := false
	stepsWritten := false

	for _, line := range lines {
		if strings.HasPrefix(line, "## Steps") || strings.HasPrefix(line, "## Cycles") {
			result = append(result, "## Steps", "")
			result = append(result, stepLines...)
			inStepsSection = true
			stepsWritten = true
			continue
		}

		if inStepsSection {
			if strings.HasPrefix(line, "## ") {
				inStepsSection = false
				result = append(result, line)
			}
			continue
		}

		result = append(result, line)
	}

	return result, stepsWritten
}

// WriteTo writes the parsed initiative to a file, preserving non-step sections.
// Uses atomic write (temp file + rename) for safety.
func (p *ParsedInitiative) WriteTo(path string) error {
	originalContent, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading original file: %w", err)
	}

	lines := strings.Split(string(originalContent), "\n")
	result, stepsWritten := replaceStepsSection(lines, p.formatSteps())

	if !stepsWritten && len(p.Steps) > 0 {
		result = append(result, "", "## Steps", "")
		result = append(result, p.formatSteps()...)
	}

	tempPath := path + ".tmp"
	content := strings.Join(result, "\n")
	if err := os.WriteFile(tempPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}

	return nil
}

// hasProfiles returns true if any step has a non-empty Profile field.
func (p *ParsedInitiative) hasProfiles() bool {
	for _, step := range p.Steps {
		if step.Profile != "" {
			return true
		}
	}
	return false
}

// formatSteps formats steps as a markdown table.
// Uses 4-column format (with Profile) if any step has a profile set,
// otherwise uses 3-column format for backwards compatibility.
func (p *ParsedInitiative) formatSteps() []string {
	var lines []string

	if p.hasProfiles() {
		lines = append(lines, "| Step | Profile | Status | Updated |")
		lines = append(lines, "|------|---------|--------|---------|")
		for _, step := range p.Steps {
			row := fmt.Sprintf("| %s | %s | %s | %s |", step.Name, step.Profile, step.Status, step.Updated)
			lines = append(lines, row)
		}
	} else {
		lines = append(lines, "| Step | Status | Updated |")
		lines = append(lines, "|------|--------|---------|")
		for _, step := range p.Steps {
			row := fmt.Sprintf("| %s | %s | %s |", step.Name, step.Status, step.Updated)
			lines = append(lines, row)
		}
	}
	lines = append(lines, "")

	return lines
}
