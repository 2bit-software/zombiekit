package claude

import (
	"fmt"
	"strings"
)

// MaxChunkSize is the maximum size in characters for a single chunk.
const MaxChunkSize = 8000

// ChunkMessage splits a long message into smaller chunks at sentence boundaries.
// Short messages (under MaxChunkSize) are returned unchanged.
func ChunkMessage(content string) []string {
	if len(content) <= MaxChunkSize {
		return []string{content}
	}

	var chunks []string
	remaining := content

	for len(remaining) > MaxChunkSize {
		// Find last sentence boundary before MaxChunkSize
		cutPoint := findSentenceBoundary(remaining[:MaxChunkSize])
		if cutPoint == 0 {
			// No boundary found, force cut at MaxChunkSize
			cutPoint = MaxChunkSize
		}

		chunks = append(chunks, strings.TrimSpace(remaining[:cutPoint]))
		remaining = strings.TrimSpace(remaining[cutPoint:])
	}

	if len(remaining) > 0 {
		chunks = append(chunks, remaining)
	}

	return chunks
}

// ChunkSourceID generates a unique source_id for each chunk of a message.
// For single-chunk messages: returns original UUID unchanged.
// For multi-chunk messages: appends chunk index (e.g., "abc123-0", "abc123-1").
func ChunkSourceID(originalUUID string, chunkIndex int, totalChunks int) string {
	if totalChunks == 1 {
		return originalUUID
	}
	return fmt.Sprintf("%s-%d", originalUUID, chunkIndex)
}

// findSentenceBoundary finds the last sentence boundary in the text.
// Returns the position after the boundary, or 0 if no boundary found.
func findSentenceBoundary(text string) int {
	// Look for sentence-ending punctuation from the end
	for i := len(text) - 1; i > 0; i-- {
		if isSentenceEnd(text, i) {
			return i + 1
		}
	}
	return 0
}

// isSentenceEnd checks if position i is a sentence ending.
func isSentenceEnd(text string, i int) bool {
	if i >= len(text)-1 {
		return false
	}

	char := text[i]
	nextChar := text[i+1]

	// Check for ". ", ".\n", "? ", "?\n", "! ", "!\n"
	if (char == '.' || char == '?' || char == '!') && (nextChar == ' ' || nextChar == '\n') {
		return true
	}

	return false
}
