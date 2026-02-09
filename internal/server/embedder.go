package server

import (
	"context"

	"github.com/zombiekit/brains/internal/recall"
)

type OllamaEmbedderAdapter struct {
	embedder *recall.OllamaEmbedder
}

func NewOllamaEmbedderAdapter(ollamaURL, model string) (*OllamaEmbedderAdapter, error) {
	e, err := recall.NewOllamaEmbedder(ollamaURL, model)
	if err != nil {
		return nil, err
	}
	return &OllamaEmbedderAdapter{embedder: e}, nil
}

func (a *OllamaEmbedderAdapter) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return a.embedder.Embed(ctx, text, recall.PurposeQuery)
}
