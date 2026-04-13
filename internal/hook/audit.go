package hook

import "time"

// AuditRecord is the per-invocation trace written to the hook log.
// One record is emitted per zk hook CLI invocation.
type AuditRecord struct {
	Timestamp      time.Time     `json:"ts"`
	Event          string        `json:"event"`
	SessionID      string        `json:"session"`
	Agent          string        `json:"agent"`
	CWD            string        `json:"cwd,omitempty"`
	Source         string        `json:"source,omitempty"`
	ToolName       string        `json:"tool_name,omitempty"`
	Command        string        `json:"command,omitempty"`
	FilePaths      []string      `json:"file_paths,omitempty"`
	MatchedRules   []MatchedRule `json:"matched_rules,omitempty"`
	SkippedRules   []MatchedRule `json:"skipped_rules,omitempty"`
	OutputBytes    int           `json:"output_bytes"`
	DurationMicros int64         `json:"duration_us"`
	Err            string        `json:"err,omitempty"`
}

// AuditSink persists hook audit records. Implementations are not required to
// be safe for concurrent use; zk hook is a short-lived CLI process.
type AuditSink interface {
	Write(rec AuditRecord) error
}

// NopSink discards audit records. Used in tests and when auditing is disabled.
type NopSink struct{}

// Write reports success without persisting the record.
func (NopSink) Write(AuditRecord) error { return nil }
