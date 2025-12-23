// Package memory provides persistent memory storage functionality.
package memory

// SanitizeName sanitizes a memory name to contain only valid characters.
// Valid characters: a-z, A-Z, 0-9, -, _, .
// Invalid characters are replaced with underscores.
// Empty names become "unnamed".
//
// This implementation is compatible with mcp-genie's sanitizeName function.
func SanitizeName(name string) string {
	if name == "" {
		return "unnamed"
	}

	runes := []rune(name)
	result := make([]rune, len(runes))

	for i, r := range runes {
		// Keep valid characters: alphanumeric, underscore, hyphen, dot
		if isValidNameChar(r) {
			result[i] = r
		} else {
			result[i] = '_'
		}
	}

	return string(result)
}

// isValidNameChar returns true if the character is valid for a memory name.
func isValidNameChar(r rune) bool {
	return (r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') ||
		r == '_' ||
		r == '-' ||
		r == '.'
}

// MaxNameLength is the maximum allowed length for a memory name.
const MaxNameLength = 255

// MaxContentSize is the maximum allowed size for memory content in bytes.
const MaxContentSize = 1048576 // 1MB

// ValidateName checks if a name is valid after sanitization.
func ValidateName(name string) error {
	if len(name) > MaxNameLength {
		return ErrNameTooLong
	}
	return nil
}

// ValidateContent checks if content is valid.
func ValidateContent(content string) error {
	if len(content) > MaxContentSize {
		return ErrContentTooLarge
	}
	return nil
}
