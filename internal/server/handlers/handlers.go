package handlers

import (
	"github.com/2bit-software/zombiekit/gen/zombiekit/brains/llm/v1/llmv1connect"
)

type LLMService struct {
	llmv1connect.UnimplementedLLMServiceHandler
}
