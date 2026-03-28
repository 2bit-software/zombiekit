package state

import "errors"

var (
	// ErrInvalidDBPath is returned when the database path is empty
	// or the parent directory cannot be created.
	ErrInvalidDBPath = errors.New("invalid database path")

	// ErrJobExists is returned when CreateJob is called with a ticket ID that already exists.
	ErrJobExists = errors.New("job already exists")

	// ErrJobNotFound is returned when an operation targets a job that does not exist.
	ErrJobNotFound = errors.New("job not found")
)
