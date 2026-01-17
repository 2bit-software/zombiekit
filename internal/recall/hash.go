package recall

import (
	"crypto/sha256"
	"encoding/hex"
)

// ContentHash returns a SHA-256 hash of the content for duplicate detection.
func ContentHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}
