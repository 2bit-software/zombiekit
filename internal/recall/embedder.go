package recall

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/ollama/ollama/api"
)

// errContextLength is the error substring from Ollama when input exceeds context.
const errContextLength = "the input length exceeds the context length"

// EmbedPurpose indicates how the text will be used.
type EmbedPurpose int

const (
	// PurposeDocument is for content being stored.
	PurposeDocument EmbedPurpose = iota
	// PurposeQuery is for search queries.
	PurposeQuery
)

// ExpectedEmbeddingDimension is the expected dimension for nomic-embed-text.
const ExpectedEmbeddingDimension = 768

// Embedder generates vector embeddings for text.
type Embedder interface {
	// Embed returns the embedding vector for the given text.
	// Implementations should handle any necessary prefixes.
	Embed(ctx context.Context, text string, purpose EmbedPurpose) ([]float32, error)
}

// OllamaEmbedder implements Embedder using the Ollama API.
type OllamaEmbedder struct {
	client    *api.Client
	model     string
	validated bool
}

// NewOllamaEmbedder creates a new OllamaEmbedder.
func NewOllamaEmbedder(ollamaURL, model string) (*OllamaEmbedder, error) {
	base, err := url.Parse(ollamaURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama URL: %w", err)
	}

	client := api.NewClient(base, http.DefaultClient)
	return &OllamaEmbedder{
		client: client,
		model:  model,
	}, nil
}

// Embed generates an embedding for the given text.
// If the text exceeds the model's context length, it recursively trims
// sentences until it fits or the text is empty.
func (e *OllamaEmbedder) Embed(ctx context.Context, text string, purpose EmbedPurpose) ([]float32, error) {
	return e.embedWithRetry(ctx, text, purpose)
}

// embedWithRetry attempts to embed text, trimming sentences on context overflow.
func (e *OllamaEmbedder) embedWithRetry(ctx context.Context, text string, purpose EmbedPurpose) ([]float32, error) {
	// Apply task prefix per nomic-embed-text best practices
	var prefixed string
	switch purpose {
	case PurposeDocument:
		prefixed = "search_document: " + text
	case PurposeQuery:
		prefixed = "search_query: " + text
	default:
		prefixed = text
	}

	resp, err := e.client.Embed(ctx, &api.EmbedRequest{
		Model: e.model,
		Input: prefixed,
	})
	if err != nil {
		// Check for context length overflow
		if strings.Contains(err.Error(), errContextLength) {
			trimmed := trimLastSentence(text)
			if trimmed == "" {
				return nil, fmt.Errorf("text too long for embedding even after trimming: %w", err)
			}
			// Retry with shorter text
			return e.embedWithRetry(ctx, trimmed, purpose)
		}
		return nil, fmt.Errorf("ollama embed: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("ollama returned no embeddings")
	}

	embedding := resp.Embeddings[0]

	// Validate dimension on first call
	if !e.validated {
		if len(embedding) != ExpectedEmbeddingDimension {
			return nil, fmt.Errorf("embedding model %q produces %d dimensions, expected %d (nomic-embed-text)",
				e.model, len(embedding), ExpectedEmbeddingDimension)
		}
		e.validated = true
	}

	return embedding, nil
}

// trimLastSentence removes the last sentence from text.
// Returns empty string if no sentence boundary found or text is too short.
func trimLastSentence(text string) string {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return ""
	}

	// Find the last sentence boundary (before the final sentence)
	// Look for ". ", "? ", "! ", ".\n", "?\n", "!\n"
	lastBoundary := -1
	for i := len(text) - 2; i >= 0; i-- {
		if i+1 < len(text) && isSentenceEnd(text[i], text[i+1]) {
			lastBoundary = i + 1
			break
		}
	}

	if lastBoundary <= 0 {
		// No sentence boundary found - try paragraph boundary
		if idx := strings.LastIndex(text, "\n\n"); idx > 0 {
			return strings.TrimSpace(text[:idx])
		}
		// No boundary at all - can't safely trim
		return ""
	}

	return strings.TrimSpace(text[:lastBoundary])
}

// isSentenceEnd checks if char followed by next forms a sentence ending.
func isSentenceEnd(char, next byte) bool {
	return (char == '.' || char == '?' || char == '!') && (next == ' ' || next == '\n')
}

// CheckAvailable verifies the Ollama service is reachable.
func (e *OllamaEmbedder) CheckAvailable(ctx context.Context) error {
	_, err := e.client.List(ctx)
	if err != nil {
		return fmt.Errorf("cannot connect to Ollama: %w", err)
	}
	return nil
}
