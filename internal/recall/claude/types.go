// Package claude implements Claude Code conversation history parsing and import.
package claude

import "time"

// HistoryEntry represents a single entry in a Claude Code history JSONL file.
type HistoryEntry struct {
	Type        string          `json:"type"`        // "user", "assistant", "summary", etc.
	UUID        string          `json:"uuid"`        // Unique message ID
	ParentUUID  *string         `json:"parentUuid"`  // Parent message for threading
	SessionID   string          `json:"sessionId"`   // Conversation/session ID
	Timestamp   time.Time       `json:"timestamp"`   // Message timestamp
	Message     *MessageContent `json:"message,omitempty"`
	IsMeta      bool            `json:"isMeta"`      // Skip if true
	IsSidechain bool            `json:"isSidechain"` // Alternate branch, still imported
	CWD         string          `json:"cwd,omitempty"`
	GitBranch   string          `json:"gitBranch,omitempty"`
	Version     string          `json:"version,omitempty"`
}

// MessageContent wraps the role and content of a message.
type MessageContent struct {
	Role    string      `json:"role"`    // "user" or "assistant"
	Content any `json:"content"` // string OR []ContentBlock
}

// ContentBlock represents a single block within a message.
type ContentBlock struct {
	Type     string `json:"type"`               // "text", "thinking", "tool_use", "tool_result"
	Text     string `json:"text,omitempty"`     // For text blocks
	Thinking string `json:"thinking,omitempty"` // For thinking blocks
}
