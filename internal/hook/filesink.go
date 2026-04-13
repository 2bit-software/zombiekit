package hook

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// FileSink appends audit records as newline-delimited JSON to a file
// under ~/.zombiekit/logs/hooks.jsonl.
type FileSink struct {
	path string
}

// NewFileSink constructs a FileSink rooted at homeDir/.zombiekit/logs/hooks.jsonl.
func NewFileSink(homeDir string) *FileSink {
	return &FileSink{
		path: filepath.Join(homeDir, ".zombiekit", "logs", "hooks.jsonl"),
	}
}

// Path reports the file path the sink writes to.
func (s *FileSink) Path() string { return s.path }

// Write appends a single JSON-encoded record followed by a newline.
func (s *FileSink) Write(rec AuditRecord) (err error) {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	return json.NewEncoder(f).Encode(rec)
}
