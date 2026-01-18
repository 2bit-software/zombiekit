package claude

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ParseFile reads a JSONL file and returns all history entries.
// Malformed lines are skipped gracefully.
func ParseFile(path string) ([]HistoryEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	var entries []HistoryEntry
	scanner := bufio.NewScanner(file)
	// Allow up to 10MB per line for large messages
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry HistoryEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			// Skip malformed lines (graceful degradation)
			continue
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan file: %w", err)
	}

	return entries, nil
}

// FilterImportable returns only user/assistant messages that should be imported.
// Filters out: isMeta messages, non-user/assistant types.
// Includes: isSidechain messages (valid conversation branches per Decision 7).
func FilterImportable(entries []HistoryEntry) []HistoryEntry {
	var result []HistoryEntry
	for _, e := range entries {
		if e.IsMeta {
			continue
		}
		if e.Type != "user" && e.Type != "assistant" {
			continue
		}
		if e.Message == nil {
			continue
		}
		// Sidechain messages are included - they represent valid conversation branches
		result = append(result, e)
	}
	return result
}

// ExtractContent extracts searchable text content from a history entry.
// Handles both string content and []ContentBlock content.
// Extracts text and thinking blocks; skips tool_use/tool_result.
func ExtractContent(entry HistoryEntry) string {
	if entry.Message == nil {
		return ""
	}

	switch c := entry.Message.Content.(type) {
	case string:
		return c
	case []any:
		var texts []string
		for _, block := range c {
			if m, ok := block.(map[string]any); ok {
				blockType, _ := m["type"].(string)
				switch blockType {
				case "text":
					if t, ok := m["text"].(string); ok {
						texts = append(texts, t)
					}
				case "thinking":
					// Include thinking blocks - valuable for search
					if t, ok := m["thinking"].(string); ok {
						texts = append(texts, t)
					}
				// tool_use, tool_result: skip (not searchable prose)
				}
			}
		}
		return strings.Join(texts, "\n")
	default:
		return fmt.Sprintf("%v", c)
	}
}
