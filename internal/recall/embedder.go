package recall

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"
)

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
func (e *OllamaEmbedder) Embed(ctx context.Context, text string, purpose EmbedPurpose) ([]float32, error) {
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

// CheckAvailable verifies the Ollama service is reachable.
func (e *OllamaEmbedder) CheckAvailable(ctx context.Context) error {
	_, err := e.client.List(ctx)
	if err != nil {
		return fmt.Errorf("cannot connect to Ollama: %w", err)
	}
	return nil
}
