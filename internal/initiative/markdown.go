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

// ParsedCycle represents a cycle parsed from INITIATIVE.md.
type ParsedCycle struct {
	Number int          // Cycle number (1, 2, 3...)
	Type   string       // Cycle type ("feat", "ref", "fix")
	Name   string       // Cycle name slug
	Status string       // "active" or "completed"
	Steps  []ParsedStep // Steps in this cycle
}

// ParsedInitiative represents an INITIATIVE.md parsed into structured data.
type ParsedInitiative struct {
	Name    string        // Initiative name slug
	Type    string        // Initiative type ("feature", "bug", "refactor")
	Status  string        // Status from header
	Created time.Time     // Created timestamp
	Cycles  []ParsedCycle // All cycles in this initiative
}

// ActiveCycle returns the currently active cycle, or nil if none.
func (p *ParsedInitiative) ActiveCycle() *ParsedCycle {
	for i := range p.Cycles {
		if p.Cycles[i].Status == "active" {
			return &p.Cycles[i]
		}
	}
	return nil
}

// CurrentStep returns the current in-progress step, or nil if none.
func (p *ParsedInitiative) CurrentStep() *ParsedStep {
	cycle := p.ActiveCycle()
	if cycle == nil {
		return nil
	}
	for i := range cycle.Steps {
		if cycle.Steps[i].Status == StepInProgress {
			return &cycle.Steps[i]
		}
	}
	return nil
}

// NextStep returns the next pending step after the current step.
func (p *ParsedInitiative) NextStep() *ParsedStep {
	cycle := p.ActiveCycle()
	if cycle == nil {
		return nil
	}
	foundCurrent := false
	for i := range cycle.Steps {
		if cycle.Steps[i].Status == StepInProgress {
			foundCurrent = true
			continue
		}
		if foundCurrent && cycle.Steps[i].Status == StepPending {
			return &cycle.Steps[i]
		}
	}
	// If no in-progress step, find first pending
	if !foundCurrent {
		for i := range cycle.Steps {
			if cycle.Steps[i].Status == StepPending {
				return &cycle.Steps[i]
			}
		}
	}
	return nil
}

var (
	// Matches cycle headers like: ### 1. feat/user-auth (active)
	cycleHeaderRe = regexp.MustCompile(`^###\s+(\d+)\.\s+(\w+)/([^\s]+)\s+\((\w+)\)`)
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

	var currentCycle *ParsedCycle
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

		// Parse cycle header: ### 1. feat/user-auth (active)
		if matches := cycleHeaderRe.FindStringSubmatch(line); matches != nil {
			// Save previous cycle if exists
			if currentCycle != nil {
				parsed.Cycles = append(parsed.Cycles, *currentCycle)
			}

			number := 0
			fmt.Sscanf(matches[1], "%d", &number)

			currentCycle = &ParsedCycle{
				Number: number,
				Type:   matches[2],
				Name:   matches[3],
				Status: matches[4],
				Steps:  []ParsedStep{},
			}
			inStepTable = false
			continue
		}

		// Detect step table header
		if currentCycle != nil && strings.HasPrefix(line, "| Step ") {
			inStepTable = true
			continue
		}

		// Skip table separator
		if strings.HasPrefix(line, "|---") || strings.HasPrefix(line, "| ---") {
			continue
		}

		// Parse step table row
		if currentCycle != nil && inStepTable {
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
				currentCycle.Steps = append(currentCycle.Steps, step)
				continue
			} else if line == "" || !strings.HasPrefix(line, "|") {
				// End of table
				inStepTable = false
			}
		}
	}

	// Don't forget the last cycle
	if currentCycle != nil {
		parsed.Cycles = append(parsed.Cycles, *currentCycle)
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

// UpdateStepStatus updates the status of a step in a specific cycle.
// Returns an error if the cycle or step is not found.
func (p *ParsedInitiative) UpdateStepStatus(cycleNum int, stepName string, status StepStatus, timestamp string) error {
	for i := range p.Cycles {
		if p.Cycles[i].Number == cycleNum {
			for j := range p.Cycles[i].Steps {
				if p.Cycles[i].Steps[j].Name == stepName {
					p.Cycles[i].Steps[j].Status = status
					p.Cycles[i].Steps[j].Updated = timestamp
					return nil
				}
			}
			return fmt.Errorf("step not found: %s in cycle %d", stepName, cycleNum)
		}
	}
	return fmt.Errorf("cycle not found: %d", cycleNum)
}

// AddStep inserts a new step after the specified step in a cycle.
// If afterStep is empty, the step is added at the beginning.
func (p *ParsedInitiative) AddStep(cycleNum int, afterStep string, newStep ParsedStep) error {
	for i := range p.Cycles {
		if p.Cycles[i].Number == cycleNum {
			if afterStep == "" {
				// Add at the beginning
				p.Cycles[i].Steps = append([]ParsedStep{newStep}, p.Cycles[i].Steps...)
				return nil
			}
			for j := range p.Cycles[i].Steps {
				if p.Cycles[i].Steps[j].Name == afterStep {
					// Insert after this step
					newSteps := make([]ParsedStep, 0, len(p.Cycles[i].Steps)+1)
					newSteps = append(newSteps, p.Cycles[i].Steps[:j+1]...)
					newSteps = append(newSteps, newStep)
					newSteps = append(newSteps, p.Cycles[i].Steps[j+1:]...)
					p.Cycles[i].Steps = newSteps
					return nil
				}
			}
			return fmt.Errorf("step not found: %s in cycle %d", afterStep, cycleNum)
		}
	}
	return fmt.Errorf("cycle not found: %d", cycleNum)
}

// WriteTo writes the parsed initiative to a file, preserving non-cycle sections.
// Uses atomic write (temp file + rename) for safety.
func (p *ParsedInitiative) WriteTo(path string) error {
	// Read the original file to preserve non-cycle content
	originalContent, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading original file: %w", err)
	}

	// Find the Cycles section and replace it
	lines := strings.Split(string(originalContent), "\n")
	var result []string
	inCyclesSection := false
	cyclesWritten := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Detect "## Cycles" section
		if strings.HasPrefix(line, "## Cycles") {
			result = append(result, line)
			result = append(result, "")
			inCyclesSection = true

			// Write all cycles
			for _, cycle := range p.Cycles {
				result = append(result, p.formatCycle(&cycle)...)
			}
			cyclesWritten = true
			continue
		}

		// If in cycles section, skip until next ## section
		if inCyclesSection {
			if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "## Cycles") {
				inCyclesSection = false
				result = append(result, line)
			}
			// Skip lines in cycles section (they've been replaced)
			continue
		}

		result = append(result, line)
	}

	// If no cycles section was found, append one
	if !cyclesWritten && len(p.Cycles) > 0 {
		result = append(result, "")
		result = append(result, "## Cycles")
		result = append(result, "")
		for _, cycle := range p.Cycles {
			result = append(result, p.formatCycle(&cycle)...)
		}
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

// formatCycle formats a cycle as markdown lines.
func (p *ParsedInitiative) formatCycle(cycle *ParsedCycle) []string {
	var lines []string

	// Cycle header: ### 1. feat/user-auth (active)
	header := fmt.Sprintf("### %d. %s/%s (%s)", cycle.Number, cycle.Type, cycle.Name, cycle.Status)
	lines = append(lines, header)
	lines = append(lines, "")

	// Step table
	lines = append(lines, "| Step | Status | Updated |")
	lines = append(lines, "|------|--------|---------|")
	for _, step := range cycle.Steps {
		row := fmt.Sprintf("| %s | %s | %s |", step.Name, step.Status, step.Updated)
		lines = append(lines, row)
	}
	lines = append(lines, "")

	return lines
}
