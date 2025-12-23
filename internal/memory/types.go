// Package memory provides persistent memory storage functionality.
package memory

import "time"

// MemoryItem represents a single memory entry.
type MemoryItem struct {
	Name      string    `json:"name" db:"name"`
	Content   string    `json:"content" db:"content"`
	Version   int       `json:"version" db:"version"`
	Deleted   bool      `json:"deleted" db:"deleted"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// MemoryMetadata contains metadata about a memory item (for list operations).
type MemoryMetadata struct {
	Name      string    `json:"name" db:"name"`
	Size      int       `json:"size" db:"-"`
	Version   int       `json:"version" db:"version"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}
