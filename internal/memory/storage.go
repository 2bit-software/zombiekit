// Package memory provides persistent memory storage functionality.
package memory

import (
	"context"

	"github.com/zombiekit/brains/internal/mo"
)

// Storage defines the interface for storing and retrieving memory items.
// This interface is compatible with mcp-genie's stickymemory implementation.
type Storage interface {
	// Set stores a memory item (creates new version).
	Set(ctx context.Context, name, content string) error

	// Get retrieves the latest non-deleted version of a memory item.
	// Returns Nothing if the item doesn't exist or is deleted.
	Get(ctx context.Context, name string) (mo.Maybe[MemoryItem], error)

	// Delete soft-deletes all versions of a memory item.
	Delete(ctx context.Context, name string) error

	// List returns all items, optionally filtered by search query.
	// If search is empty, returns all items.
	// Results are ordered by updated_at descending.
	List(ctx context.Context, search string) ([]MemoryMetadata, error)

	// Clear removes all items and returns the count of distinct names deleted.
	Clear(ctx context.Context) (int, error)

	// Close closes any resources held by the storage.
	Close() error
}
