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
	// StatusActive represents an active initiative.
	StatusActive InitiativeStatus = "active"
	// StatusCompleted represents a completed initiative.
	StatusCompleted InitiativeStatus = "completed"
)

// IsValid returns true if the initiative status is valid.
func (s InitiativeStatus) IsValid() bool {
	switch s {
	case StatusActive, StatusCompleted:
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
// NOTE: This is a pointer-only structure. Status is NOT stored here;
// it should be read from INITIATIVE.md frontmatter.
type InitiativeState struct {
	// Initiative is the relative path to active initiative (from project root).
	Initiative string `json:"initiative,omitempty"`
	// Cycle is the relative path to active cycle within the initiative.
	Cycle string `json:"cycle,omitempty"`
	// Started is when this initiative became active.
	Started time.Time `json:"started,omitempty"`
	// LastActivity is the last step execution time.
	LastActivity time.Time `json:"last_activity,omitempty"`
	// CurrentStep is the last executed step.
	CurrentStep string `json:"current_step,omitempty"`
}

// IsEmpty returns true if there is no active initiative.
func (s *InitiativeState) IsEmpty() bool {
	return s.Initiative == ""
}

// CycleType represents the type of cycle within an initiative.
type CycleType string

const (
	// CycleFeat represents a feature cycle.
	CycleFeat CycleType = "feat"
	// CycleRef represents a refactor cycle.
	CycleRef CycleType = "ref"
	// CycleFix represents a bug fix cycle.
	CycleFix CycleType = "fix"
)

// IsValid returns true if the cycle type is valid.
func (c CycleType) IsValid() bool {
	switch c {
	case CycleFeat, CycleRef, CycleFix:
		return true
	default:
		return false
	}
}

// String returns the string representation of the cycle type.
func (c CycleType) String() string {
	return string(c)
}

// CycleStatus represents the status of a cycle.
type CycleStatus string

const (
	// CycleStatusTemplate means blank templates have been created.
	CycleStatusTemplate CycleStatus = "template"
	// CycleStatusInProgress means the workflow is executing.
	CycleStatusInProgress CycleStatus = "in_progress"
	// CycleStatusAudited means the cycle passed audit.
	CycleStatusAudited CycleStatus = "audited"
	// CycleStatusApproved means the user approved the cycle.
	CycleStatusApproved CycleStatus = "approved"
)

// IsValid returns true if the cycle status is valid.
func (s CycleStatus) IsValid() bool {
	switch s {
	case CycleStatusTemplate, CycleStatusInProgress, CycleStatusAudited, CycleStatusApproved:
		return true
	default:
		return false
	}
}

// String returns the string representation of the cycle status.
func (s CycleStatus) String() string {
	return string(s)
}

// Cycle represents a single workflow pass within an initiative.
type Cycle struct {
	// ID is the unique identifier (e.g., "675d8a40-feat-user-auth").
	ID string `json:"id"`
	// Type is the cycle type (feat, ref, fix).
	Type CycleType `json:"type"`
	// Name is the human-readable name slug.
	Name string `json:"name"`
	// Path is the absolute path to the cycle folder.
	Path string `json:"path"`
	// Status is the current cycle status.
	Status CycleStatus `json:"status"`
	// InitiativeID is the parent initiative ID.
	InitiativeID string `json:"initiative_id"`
	// Number is the cycle number within the initiative (1, 2, 3...).
	Number int `json:"number"`
	// CreatedAt is when the cycle was created.
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the last update timestamp.
	UpdatedAt time.Time `json:"updated_at"`
}
