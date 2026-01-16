package initiative

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CreateCycle creates a new cycle within an initiative.
// It creates the cycle directory and audit subdirectory.
func (s *Service) CreateCycle(initPath string, cycleType CycleType, name string) (*Cycle, error) {
	// Validate cycle type
	if !cycleType.IsValid() {
		return nil, &InitiativeError{
			Code:    "INVALID_CYCLE_TYPE",
			Message: fmt.Sprintf("invalid cycle type '%s'", cycleType),
			Hint:    "Type must be one of: feat, ref, fix",
		}
	}

	// Normalize name
	normalizedName := normalizeName(name)
	if err := validateName(normalizedName); err != nil {
		return nil, err
	}

	// Get next cycle number
	cycleNum, err := s.getNextCycleNumber(initPath)
	if err != nil {
		return nil, err
	}

	// Generate cycle ID
	cycleID := generateCycleID(cycleType, normalizedName)
	cyclePath := filepath.Join(initPath, cycleID)

	// Create cycle directory
	if err := os.MkdirAll(cyclePath, 0755); err != nil {
		return nil, fmt.Errorf("creating cycle directory: %w", err)
	}

	// Create audit subdirectory
	auditPath := filepath.Join(cyclePath, "audit")
	if err := os.MkdirAll(auditPath, 0755); err != nil {
		return nil, fmt.Errorf("creating audit directory: %w", err)
	}

	now := time.Now()
	cycle := &Cycle{
		ID:           cycleID,
		Type:         cycleType,
		Name:         normalizedName,
		Path:         cyclePath,
		Status:       CycleStatusTemplate,
		InitiativeID: filepath.Base(initPath),
		Number:       cycleNum,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	return cycle, nil
}

// generateCycleID generates a unique cycle ID in format: {hex-timestamp}-{cycle-type}-{name}
func generateCycleID(cycleType CycleType, name string) string {
	timestamp := fmt.Sprintf("%08x", time.Now().Unix())
	return fmt.Sprintf("%s-%s-%s", timestamp, cycleType, name)
}

// getNextCycleNumber counts existing cycle directories and returns the next number.
func (s *Service) getNextCycleNumber(initPath string) (int, error) {
	entries, err := os.ReadDir(initPath)
	if err != nil {
		// If directory doesn't exist yet, this is the first cycle
		if os.IsNotExist(err) {
			return 1, nil
		}
		return 0, fmt.Errorf("reading initiative directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		// Count subdirectories that look like cycles (exclude audit, INITIATIVE.md, etc.)
		if entry.IsDir() && entry.Name() != "audit" {
			count++
		}
	}
	return count + 1, nil
}
