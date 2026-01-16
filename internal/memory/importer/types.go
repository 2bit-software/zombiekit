// Package importer provides SQLite to PostgreSQL memory data migration.
package importer

import "time"

// ImportOptions configures import behavior.
type ImportOptions struct {
	// SourcePath is the path to the SQLite database file.
	SourcePath string

	// TargetURL is the PostgreSQL connection URL.
	TargetURL string

	// BatchSize is the number of items per batch transaction.
	// Default: 100
	BatchSize int

	// DryRun previews import without making changes.
	DryRun bool

	// OnProgress is called after each item is processed.
	// Parameters: imported count, total count, current item name.
	OnProgress ProgressFunc
}

// ProgressFunc is called during import to report progress.
type ProgressFunc func(imported, total int, currentItem string)

// ImportResult contains the results of an import operation.
type ImportResult struct {
	// SourcePath is the SQLite database that was imported.
	SourcePath string `json:"source"`

	// TargetURL is the PostgreSQL database (masked for display).
	TargetURL string `json:"target"`

	// DryRun indicates if this was a preview run.
	DryRun bool `json:"dry_run"`

	// Imported is the count of successfully imported items.
	Imported int `json:"imported"`

	// Skipped is the count of items that already exist.
	Skipped int `json:"skipped"`

	// Errors is the count of items that failed to import.
	ErrorCount int `json:"errors"`

	// ErrorDetails contains per-item error information.
	ErrorDetails []ItemError `json:"error_details,omitempty"`

	// Duration is how long the import took.
	Duration time.Duration `json:"duration_ms"`

	// PendingItems lists items that would be imported (dry-run only).
	PendingItems []PendingItem `json:"pending_items,omitempty"`

	// TotalInSource is the total count of items in the source database.
	TotalInSource int `json:"total_in_source,omitempty"`
}

// ItemError describes a single item import failure.
type ItemError struct {
	Name    string `json:"name"`
	Version int    `json:"version"`
	Error   string `json:"error"`
}

// PendingItem represents an item that would be imported (for dry-run preview).
type PendingItem struct {
	Name      string    `json:"name"`
	Version   int       `json:"version"`
	UpdatedAt time.Time `json:"updated_at"`
}
