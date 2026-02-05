// Package initiative provides initiative management for the step framework.
// An initiative represents a unit of work (feature, bug, refactor) being tracked.
package initiative

import (
	"time"
)

// InitiativeType represents the type of initiative.
type InitiativeType string

const (
	// TypeFeature represents a new feature initiative.
	TypeFeature InitiativeType = "feature"
	// TypeBug represents a bug fix initiative.
	TypeBug InitiativeType = "bug"
	// TypeRefactor represents a refactoring initiative.
	TypeRefactor InitiativeType = "refactor"
)

// ValidTypes returns all valid initiative types.
func ValidTypes() []InitiativeType {
	return []InitiativeType{TypeFeature, TypeBug, TypeRefactor}
}

// IsValid returns true if the initiative type is valid.
func (t InitiativeType) IsValid() bool {
	switch t {
	case TypeFeature, TypeBug, TypeRefactor:
		return true
	default:
		return false
	}
}

// String returns the string representation of the initiative type.
func (t InitiativeType) String() string {
	return string(t)
}

// InitiativeStatus represents the status of an initiative.
type InitiativeStatus string

const (
	// StatusInProgress represents an in-progress initiative.
	StatusInProgress InitiativeStatus = "in_progress"
	// StatusComplete represents a completed initiative.
	StatusComplete InitiativeStatus = "complete"
)

// IsValid returns true if the initiative status is valid.
func (s InitiativeStatus) IsValid() bool {
	switch s {
	case StatusInProgress, StatusComplete:
		return true
	default:
		return false
	}
}

// String returns the string representation of the initiative status.
func (s InitiativeStatus) String() string {
	return string(s)
}

// Initiative represents a unit of work (feature, bug, refactor) being tracked.
type Initiative struct {
	// ID is the unique identifier (e.g., "675d8a3f-feature-user-auth").
	ID string `json:"id"`
	// Type is the initiative type (feature, bug, refactor).
	Type InitiativeType `json:"type"`
	// Name is the human-readable name slug (e.g., "user-auth").
	Name string `json:"name"`
	// Path is the absolute path to the initiative folder.
	Path string `json:"path"`
	// Status is the current status (active, completed).
	Status InitiativeStatus `json:"status"`
	// CreatedAt is when the initiative was created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the last activity timestamp.
	UpdatedAt time.Time `json:"updated_at"`
}

// InitiativeState tracks the currently active initiative for a project.
// Stored in .brains/active.json.
// NOTE: This is a minimal pointer structure. Step/phase state is tracked
// in INITIATIVE.md, not here.
type InitiativeState struct {
	// Initiative is the relative path to active initiative (from project root).
	Initiative string `json:"initiative,omitempty"`
	// Started is when this initiative became active.
	Started time.Time `json:"started,omitempty"`
	// Status is the initiative status (in_progress, complete).
	Status InitiativeStatus `json:"status,omitempty"`
}

// IsEmpty returns true if there is no active initiative.
func (s *InitiativeState) IsEmpty() bool {
	return s.Initiative == ""
}
