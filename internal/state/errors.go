package state

import "errors"

// ErrInvalidDBPath is returned when the database path is empty
// or the parent directory cannot be created.
var ErrInvalidDBPath = errors.New("invalid database path")
