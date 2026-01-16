// Package profile provides MCP tools for profile composition and management.
package profile

// WriteRequest represents a request to write a profile to disk.
type WriteRequest struct {
	Name      string `json:"name"`                // Profile name (used as filename)
	Content   string `json:"content"`             // Full profile content including frontmatter
	Location  string `json:"location"`            // "local" or "global"
	Overwrite bool   `json:"overwrite,omitempty"` // Allow overwriting existing profile
}

// WriteResponse is returned for profile-write.
type WriteResponse struct {
	Success bool   `json:"success"`
	Path    string `json:"path,omitempty"`    // Absolute path to written file
	Error   string `json:"error,omitempty"`   // Error code if failed
	Message string `json:"message,omitempty"` // Human-readable error message
	Hint    string `json:"hint,omitempty"`    // Actionable suggestion
}
