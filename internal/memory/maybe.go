package memory

// Maybe represents an optional value. It can contain a value (Just) or be empty (Nothing).
type Maybe[T any] struct {
	value T
	valid bool
}

// Just creates a Maybe containing the given value.
func Just[T any](v T) Maybe[T] {
	return Maybe[T]{value: v, valid: true}
}

// Nothing creates an empty Maybe with no value.
func Nothing[T any]() Maybe[T] {
	return Maybe[T]{valid: false}
}

// HasValue returns true if this Maybe contains a value.
func (m Maybe[T]) HasValue() bool {
	return m.valid
}

// Value returns the contained value.
// The caller must check HasValue() first - calling Value() on Nothing is undefined.
func (m Maybe[T]) Value() T {
	return m.value
}

// ValueOr returns the contained value if present, otherwise returns the provided default.
func (m Maybe[T]) ValueOr(defaultValue T) T {
	if m.valid {
		return m.value
	}
	return defaultValue
}
