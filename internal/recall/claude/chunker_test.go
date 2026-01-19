package claude

import (
	"strings"
	"testing"
)

func TestChunkMessage_ShortMessage(t *testing.T) {
	content := "This is a short message."

	chunks := ChunkMessage(content)

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk, got %d", len(chunks))
	}
	if chunks[0] != content {
		t.Errorf("expected unchanged content, got %q", chunks[0])
	}
}

func TestChunkMessage_ExactLimit(t *testing.T) {
	// Create content exactly at MaxChunkSize
	content := strings.Repeat("x", MaxChunkSize)

	chunks := ChunkMessage(content)

	if len(chunks) != 1 {
		t.Errorf("expected 1 chunk for exact limit, got %d", len(chunks))
	}
	if len(chunks[0]) != MaxChunkSize {
		t.Errorf("expected length %d, got %d", MaxChunkSize, len(chunks[0]))
	}
}

func TestChunkMessage_SplitsAtSentence(t *testing.T) {
	// Create content that exceeds MaxChunkSize with sentence boundaries
	part1 := strings.Repeat("x", MaxChunkSize-100) + ". "
	part2 := strings.Repeat("y", 200) + "."
	content := part1 + part2

	chunks := ChunkMessage(content)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// First chunk should end with "."
	if !strings.HasSuffix(strings.TrimSpace(chunks[0]), ".") {
		t.Errorf("first chunk should end at sentence boundary, got: ...%q", chunks[0][len(chunks[0])-20:])
	}
}

func TestChunkMessage_SplitsAtNewline(t *testing.T) {
	// Create content with newline after period
	part1 := strings.Repeat("x", MaxChunkSize-100) + ".\n"
	part2 := strings.Repeat("y", 200)
	content := part1 + part2

	chunks := ChunkMessage(content)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// First chunk should end with "."
	if !strings.HasSuffix(strings.TrimSpace(chunks[0]), ".") {
		t.Errorf("first chunk should end at sentence boundary with newline")
	}
}

func TestChunkMessage_ForceCut(t *testing.T) {
	// Create content with no sentence boundaries
	content := strings.Repeat("x", MaxChunkSize+1000)

	chunks := ChunkMessage(content)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// First chunk should be exactly MaxChunkSize (force cut)
	if len(chunks[0]) != MaxChunkSize {
		t.Errorf("expected force cut at %d, got %d", MaxChunkSize, len(chunks[0]))
	}
}

func TestChunkMessage_MultipleChunks(t *testing.T) {
	// Create content that requires 3+ chunks
	content := strings.Repeat("x", MaxChunkSize*2+1000)

	chunks := ChunkMessage(content)

	if len(chunks) < 3 {
		t.Errorf("expected at least 3 chunks, got %d", len(chunks))
	}

	// Verify all content is preserved
	total := 0
	for _, c := range chunks {
		total += len(c)
	}
	if total != len(content) {
		t.Errorf("content length mismatch: original %d, chunked total %d", len(content), total)
	}
}

func TestChunkMessage_PreservesContent(t *testing.T) {
	// Create content with sentence boundaries (no trailing spaces to avoid trim differences)
	sentences := []string{
		"First sentence is here.",
		" Second sentence follows.",
		" Third sentence ends.",
	}
	// Repeat to exceed MaxChunkSize
	var builder strings.Builder
	for builder.Len() < MaxChunkSize+1000 {
		for _, s := range sentences {
			builder.WriteString(s)
		}
	}
	content := builder.String()

	chunks := ChunkMessage(content)

	// Rejoin with space (since chunks are trimmed) and verify total length is close
	totalLen := 0
	for _, c := range chunks {
		totalLen += len(c)
	}

	// Content should be mostly preserved (within ~10 chars due to trimming)
	if totalLen < len(content)-50 || totalLen > len(content) {
		t.Errorf("content length significantly changed: original %d, chunked %d", len(content), totalLen)
	}
}

func TestChunkSourceID_SingleChunk(t *testing.T) {
	result := ChunkSourceID("uuid-abc123", 0, 1)

	if result != "uuid-abc123" {
		t.Errorf("expected original UUID for single chunk, got %q", result)
	}
}

func TestChunkSourceID_MultipleChunks(t *testing.T) {
	testCases := []struct {
		uuid     string
		index    int
		total    int
		expected string
	}{
		{"abc123", 0, 3, "abc123-0"},
		{"abc123", 1, 3, "abc123-1"},
		{"abc123", 2, 3, "abc123-2"},
		{"def-456", 0, 2, "def-456-0"},
		{"def-456", 1, 2, "def-456-1"},
	}

	for _, tc := range testCases {
		result := ChunkSourceID(tc.uuid, tc.index, tc.total)
		if result != tc.expected {
			t.Errorf("ChunkSourceID(%q, %d, %d) = %q, expected %q",
				tc.uuid, tc.index, tc.total, result, tc.expected)
		}
	}
}

func TestChunkMessage_QuestionMark(t *testing.T) {
	// Test splitting at question mark
	part1 := strings.Repeat("x", MaxChunkSize-100) + "? "
	part2 := strings.Repeat("y", 200)
	content := part1 + part2

	chunks := ChunkMessage(content)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// First chunk should end with "?"
	if !strings.HasSuffix(strings.TrimSpace(chunks[0]), "?") {
		t.Errorf("first chunk should end at question mark boundary")
	}
}

func TestChunkMessage_ExclamationMark(t *testing.T) {
	// Test splitting at exclamation mark
	part1 := strings.Repeat("x", MaxChunkSize-100) + "! "
	part2 := strings.Repeat("y", 200)
	content := part1 + part2

	chunks := ChunkMessage(content)

	if len(chunks) < 2 {
		t.Fatalf("expected at least 2 chunks, got %d", len(chunks))
	}

	// First chunk should end with "!"
	if !strings.HasSuffix(strings.TrimSpace(chunks[0]), "!") {
		t.Errorf("first chunk should end at exclamation mark boundary")
	}
}
