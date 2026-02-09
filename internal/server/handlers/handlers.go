package handlers

import (
	"github.com/zombiekit/brains/gen/zombiekit/brains/llm/v1/llmv1connect"
)

type LLMService struct {
	llmv1connect.UnimplementedLLMServiceHandler
}
