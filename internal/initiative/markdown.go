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
	// Matches step table rows: | step | status | updated |
	stepRowRe = regexp.MustCompile(`^\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|\s*([^|]+)\s*\|`)
	// Matches header metadata like **Type**: feature
	metadataRe = regexp.MustCompile(`^\*\*(\w+)\*\*:\s*(.+)$`)
	// Matches initiative header: # Initiative: name
	titleRe = regexp.MustCompile(`^#\s+Initiative:\s*(.+)$`)
)

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
		line := scanner.Text()

		// Parse title: # Initiative: name
		if matches := titleRe.FindStringSubmatch(line); matches != nil {
			parsed.Name = strings.TrimSpace(matches[1])
			continue
		}

		// Parse metadata: **Key**: value
		if matches := metadataRe.FindStringSubmatch(line); matches != nil {
			key := matches[1]
			value := strings.TrimSpace(matches[2])
			switch key {
			case "Type":
				parsed.Type = value
			case "Status":
				parsed.Status = value
			case "Created":
				// Try to parse the date
				if t, err := time.Parse("2006-01-02", value); err == nil {
					parsed.Created = t
				} else if t, err := time.Parse(time.RFC3339, value); err == nil {
					parsed.Created = t
				}
			}
			continue
		}

		// Detect step table header (## Steps section or | Step | header row)
		if strings.HasPrefix(line, "| Step ") {
			inStepTable = true
			continue
		}

		// Skip table separator
		if strings.HasPrefix(line, "|---") || strings.HasPrefix(line, "| ---") {
			continue
		}

		// Parse step table row
		if inStepTable {
			if matches := stepRowRe.FindStringSubmatch(line); matches != nil {
				stepName := strings.TrimSpace(matches[1])
				statusStr := strings.TrimSpace(matches[2])
				updated := strings.TrimSpace(matches[3])

				// Skip if this looks like a header row
				if stepName == "Step" || stepName == "step" {
					continue
				}

				step := ParsedStep{
					Name:    stepName,
					Status:  parseStepStatus(statusStr),
					Updated: updated,
				}
				parsed.Steps = append(parsed.Steps, step)
				continue
			} else if line == "" || !strings.HasPrefix(line, "|") {
				// End of table
				inStepTable = false
			}
		}
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

// WriteTo writes the parsed initiative to a file, preserving non-step sections.
// Uses atomic write (temp file + rename) for safety.
func (p *ParsedInitiative) WriteTo(path string) error {
	// Read the original file to preserve non-step content
	originalContent, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading original file: %w", err)
	}

	// Find the Steps section and replace it
	lines := strings.Split(string(originalContent), "\n")
	var result []string
	inStepsSection := false
	stepsWritten := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Detect "## Steps" or legacy "## Cycles" section
		if strings.HasPrefix(line, "## Steps") || strings.HasPrefix(line, "## Cycles") {
			result = append(result, "## Steps")
			result = append(result, "")
			inStepsSection = true

			// Write all steps
			result = append(result, p.formatSteps()...)
			stepsWritten = true
			continue
		}

		// If in steps section, skip until next ## section
		if inStepsSection {
			if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "## Steps") && !strings.HasPrefix(line, "## Cycles") {
				inStepsSection = false
				result = append(result, line)
			}
			// Skip lines in steps section (they've been replaced)
			continue
		}

		result = append(result, line)
	}

	// If no steps section was found, append one
	if !stepsWritten && len(p.Steps) > 0 {
		result = append(result, "")
		result = append(result, "## Steps")
		result = append(result, "")
		result = append(result, p.formatSteps()...)
	}

	// Write atomically
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

// formatSteps formats steps as a markdown table.
func (p *ParsedInitiative) formatSteps() []string {
	var lines []string

	lines = append(lines, "| Step | Status | Updated |")
	lines = append(lines, "|------|--------|---------|")
	for _, step := range p.Steps {
		row := fmt.Sprintf("| %s | %s | %s |", step.Name, step.Status, step.Updated)
		lines = append(lines, row)
	}
	lines = append(lines, "")

	return lines
}
