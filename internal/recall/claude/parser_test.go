package claude

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseFile_ValidUserMessage(t *testing.T) {
	content := `{"type":"user","uuid":"abc123","sessionId":"sess1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"Hello world"},"isMeta":false}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	entries, err := ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Type != "user" {
		t.Errorf("expected type 'user', got %q", e.Type)
	}
	if e.UUID != "abc123" {
		t.Errorf("expected UUID 'abc123', got %q", e.UUID)
	}
	if e.SessionID != "sess1" {
		t.Errorf("expected sessionId 'sess1', got %q", e.SessionID)
	}
}

func TestParseFile_ValidAssistantMessage(t *testing.T) {
	content := `{"type":"assistant","uuid":"def456","sessionId":"sess1","timestamp":"2024-01-15T10:01:00Z","message":{"role":"assistant","content":"Hi there!"},"isMeta":false}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	entries, err := ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	e := entries[0]
	if e.Type != "assistant" {
		t.Errorf("expected type 'assistant', got %q", e.Type)
	}
	if e.Message.Role != "assistant" {
		t.Errorf("expected role 'assistant', got %q", e.Message.Role)
	}
}

func TestParseFile_MalformedJSON(t *testing.T) {
	// Mix valid and invalid lines
	content := `{"type":"user","uuid":"abc123","sessionId":"sess1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"First"},"isMeta":false}
{malformed json line
{"type":"user","uuid":"def456","sessionId":"sess1","timestamp":"2024-01-15T10:01:00Z","message":{"role":"user","content":"Third"},"isMeta":false}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	entries, err := ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseFile should not fail on malformed lines: %v", err)
	}

	// Should have parsed 2 valid entries, skipping the malformed one
	if len(entries) != 2 {
		t.Errorf("expected 2 entries (skipping malformed), got %d", len(entries))
	}
}

func TestParseFile_EmptyFile(t *testing.T) {
	tmpFile := createTempFile(t, "")
	defer os.Remove(tmpFile)

	entries, err := ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseFile failed on empty file: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty file, got %d", len(entries))
	}
}

func TestParseFile_LargeLine(t *testing.T) {
	// Create a message with large content (1MB)
	largeContent := make([]byte, 1024*1024)
	for i := range largeContent {
		largeContent[i] = 'x'
	}

	content := `{"type":"user","uuid":"abc123","sessionId":"sess1","timestamp":"2024-01-15T10:00:00Z","message":{"role":"user","content":"` + string(largeContent) + `"},"isMeta":false}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	entries, err := ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseFile failed on large line: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
}

func TestFilterImportable_SkipsIsMeta(t *testing.T) {
	entries := []HistoryEntry{
		{Type: "user", UUID: "1", IsMeta: false, Message: &MessageContent{Role: "user", Content: "normal"}},
		{Type: "user", UUID: "2", IsMeta: true, Message: &MessageContent{Role: "user", Content: "meta"}},
		{Type: "assistant", UUID: "3", IsMeta: false, Message: &MessageContent{Role: "assistant", Content: "reply"}},
	}

	result := FilterImportable(entries)

	if len(result) != 2 {
		t.Errorf("expected 2 entries (skipping isMeta), got %d", len(result))
	}

	for _, e := range result {
		if e.IsMeta {
			t.Errorf("isMeta entry should have been filtered out")
		}
	}
}

func TestFilterImportable_IncludesSidechain(t *testing.T) {
	entries := []HistoryEntry{
		{Type: "user", UUID: "1", IsSidechain: false, Message: &MessageContent{Role: "user", Content: "main"}},
		{Type: "user", UUID: "2", IsSidechain: true, Message: &MessageContent{Role: "user", Content: "branch"}},
	}

	result := FilterImportable(entries)

	if len(result) != 2 {
		t.Errorf("expected 2 entries (including sidechain), got %d", len(result))
	}
}

func TestFilterImportable_SkipsNonUserAssistant(t *testing.T) {
	entries := []HistoryEntry{
		{Type: "user", UUID: "1", Message: &MessageContent{Role: "user", Content: "user msg"}},
		{Type: "assistant", UUID: "2", Message: &MessageContent{Role: "assistant", Content: "assistant msg"}},
		{Type: "summary", UUID: "3", Message: &MessageContent{Role: "system", Content: "summary"}},
		{Type: "system", UUID: "4", Message: &MessageContent{Role: "system", Content: "system"}},
	}

	result := FilterImportable(entries)

	if len(result) != 2 {
		t.Errorf("expected 2 entries (user and assistant only), got %d", len(result))
	}

	for _, e := range result {
		if e.Type != "user" && e.Type != "assistant" {
			t.Errorf("non user/assistant type should have been filtered: %s", e.Type)
		}
	}
}

func TestExtractContent_StringContent(t *testing.T) {
	entry := HistoryEntry{
		Message: &MessageContent{
			Role:    "user",
			Content: "Hello, world!",
		},
	}

	content := ExtractContent(entry)

	if content != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got %q", content)
	}
}

func TestExtractContent_ContentBlocks(t *testing.T) {
	entry := HistoryEntry{
		Message: &MessageContent{
			Role: "assistant",
			Content: []any{
				map[string]any{"type": "text", "text": "First part."},
				map[string]any{"type": "text", "text": "Second part."},
			},
		},
	}

	content := ExtractContent(entry)

	expected := "First part.\nSecond part."
	if content != expected {
		t.Errorf("expected %q, got %q", expected, content)
	}
}

func TestExtractContent_TextBlock(t *testing.T) {
	entry := HistoryEntry{
		Message: &MessageContent{
			Role: "assistant",
			Content: []any{
				map[string]any{"type": "text", "text": "The answer is 42."},
			},
		},
	}

	content := ExtractContent(entry)

	if content != "The answer is 42." {
		t.Errorf("expected 'The answer is 42.', got %q", content)
	}
}

func TestExtractContent_ThinkingBlock(t *testing.T) {
	entry := HistoryEntry{
		Message: &MessageContent{
			Role: "assistant",
			Content: []any{
				map[string]any{"type": "thinking", "thinking": "Let me think about this..."},
				map[string]any{"type": "text", "text": "The answer is 42."},
			},
		},
	}

	content := ExtractContent(entry)

	expected := "Let me think about this...\nThe answer is 42."
	if content != expected {
		t.Errorf("expected %q, got %q", expected, content)
	}
}

func TestExtractContent_SkipsToolBlocks(t *testing.T) {
	entry := HistoryEntry{
		Message: &MessageContent{
			Role: "assistant",
			Content: []any{
				map[string]any{"type": "text", "text": "Let me run that command."},
				map[string]any{"type": "tool_use", "name": "bash", "input": map[string]any{"command": "ls"}},
				map[string]any{"type": "tool_result", "content": "file1.txt\nfile2.txt"},
				map[string]any{"type": "text", "text": "Done!"},
			},
		},
	}

	content := ExtractContent(entry)

	expected := "Let me run that command.\nDone!"
	if content != expected {
		t.Errorf("expected %q (no tool blocks), got %q", expected, content)
	}
}

func TestExtractContent_NilMessage(t *testing.T) {
	entry := HistoryEntry{
		Message: nil,
	}

	content := ExtractContent(entry)

	if content != "" {
		t.Errorf("expected empty string for nil message, got %q", content)
	}
}

func TestParseFile_PreservesTimestamp(t *testing.T) {
	content := `{"type":"user","uuid":"abc123","sessionId":"sess1","timestamp":"2024-01-15T10:30:45Z","message":{"role":"user","content":"test"},"isMeta":false}`

	tmpFile := createTempFile(t, content)
	defer os.Remove(tmpFile)

	entries, err := ParseFile(tmpFile)
	if err != nil {
		t.Fatalf("ParseFile failed: %v", err)
	}

	expected := time.Date(2024, 1, 15, 10, 30, 45, 0, time.UTC)
	if !entries[0].Timestamp.Equal(expected) {
		t.Errorf("expected timestamp %v, got %v", expected, entries[0].Timestamp)
	}
}

// createTempFile creates a temporary file with the given content.
func createTempFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.jsonl")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	return tmpFile
}
