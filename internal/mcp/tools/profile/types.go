// Package profile provides MCP tools for profile composition and management.
package profile

// SaveRequest represents a request to save a profile to disk.
type SaveRequest struct {
	Name      string `json:"name"`                // Profile name (used as filename)
	Content   string `json:"content"`             // Full profile content including frontmatter
	Location  string `json:"location"`            // "local" or "global"
	Overwrite bool   `json:"overwrite,omitempty"` // Allow overwriting existing profile
}

// SaveResponse is returned for profile-save.
type SaveResponse struct {
	Success bool   `json:"success"`
	Path    string `json:"path,omitempty"`    // Absolute path to written file
	Error   string `json:"error,omitempty"`   // Error code if failed
	Message string `json:"message,omitempty"` // Human-readable error message
	Hint    string `json:"hint,omitempty"`    // Actionable suggestion
}
