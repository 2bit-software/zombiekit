package step

import (
	"io/fs"
	"sync"
)

// Global registry for embedded template filesystem (spec-template.md, research-template.md, etc.)
var (
	globalTemplateFS fs.FS
	globalTemplateMu sync.RWMutex
)

// SetTemplateFS registers an embedded filesystem containing template files
// (spec-template.md, research-template.md, etc.).
// This should be called during application initialization.
func SetTemplateFS(fsys fs.FS) {
	globalTemplateMu.Lock()
	defer globalTemplateMu.Unlock()
	globalTemplateFS = fsys
}

// GetTemplateFS returns the registered embedded template filesystem.
// Returns nil if no template filesystem has been registered.
func GetTemplateFS() fs.FS {
	globalTemplateMu.RLock()
	defer globalTemplateMu.RUnlock()
	return globalTemplateFS
}
