package recall

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/zombiekit/brains/internal/recall"
)

const (
	// DefaultPageLimit is the default number of items per page.
	DefaultPageLimit = 20
	// MaxPageLimit is the maximum number of items per page.
	MaxPageLimit = 100
)

// Tool provides MCP tools for conversation retrieval.
type Tool struct {
	storage recall.Storage
}

// NewTool creates a recall tool with the given storage backend.
func NewTool(storage recall.Storage) *Tool {
	return &Tool{storage: storage}
}

// ListConversations handles recall-list-conversations tool calls.
func (t *Tool) ListConversations(ctx context.Context, args map[string]any) (string, error) {
	page, limit := normalizePageParams(args)
	offset := calculateOffset(page, limit)

	// Extract project filter
	project := ""
	if p, ok := args["project"].(string); ok {
		project = p
	}

	// Fetch limit+1 to detect has_more
	items, err := t.storage.ListConversations(ctx, limit+1, offset, project)
	if err != nil {
		return "", fmt.Errorf("list conversations: %w", err)
	}

	hasMore := len(items) > limit
	if hasMore {
		items = items[:limit]
	}

	// Convert to response format
	summaries := make([]ConversationSummary, len(items))
	for i, item := range items {
		summaries[i] = ConversationSummary{
			ConversationID: item.ConversationID,
			Title:          item.Title,
			MessageCount:   item.MessageCount,
			FirstMessage:   item.FirstMessage.Format("2006-01-02T15:04:05Z07:00"),
			LastMessage:    item.LastMessage.Format("2006-01-02T15:04:05Z07:00"),
			Source:         item.Source,
			Project:        item.Project,
		}
	}

	response := ListResponse{
		Page:    page,
		Limit:   limit,
		HasMore: hasMore,
		Items:   summaries,
	}

	return toJSON(response)
}

// ReadConversation handles recall-read-conversation tool calls.
func (t *Tool) ReadConversation(ctx context.Context, args map[string]any) (string, error) {
	// Validate required conversation_id
	convID, ok := args["conversation_id"].(string)
	if !ok || convID == "" {
		return toJSON(ErrorResponse{Error: "conversation_id is required"})
	}

	// Validate UUID format
	if _, err := uuid.Parse(convID); err != nil {
		return toJSON(ErrorResponse{Error: "invalid conversation_id format"})
	}

	// Check conversation exists
	exists, err := t.storage.ConversationExists(ctx, convID)
	if err != nil {
		return "", fmt.Errorf("check conversation exists: %w", err)
	}
	if !exists {
		return toJSON(ErrorResponse{Error: "conversation not found"})
	}

	page, limit := normalizePageParams(args)
	offset := calculateOffset(page, limit)

	// Fetch limit+1 to detect has_more
	chunks, err := t.storage.GetConversationChunks(ctx, convID, limit+1, offset)
	if err != nil {
		return "", fmt.Errorf("get conversation chunks: %w", err)
	}

	hasMore := len(chunks) > limit
	if hasMore {
		chunks = chunks[:limit]
	}

	// Convert to response format
	outputs := make([]ChunkOutput, len(chunks))
	for i, chunk := range chunks {
		output := ChunkOutput{
			ID:      chunk.ID,
			Content: chunk.Content,
		}

		if chunk.Metadata != nil {
			output.Role = chunk.Metadata.Role
			if !chunk.Metadata.Timestamp.IsZero() {
				output.Timestamp = chunk.Metadata.Timestamp.Format("2006-01-02T15:04:05Z07:00")
			}
			output.Project = chunk.Metadata.CWD
			output.GitBranch = chunk.Metadata.GitBranch
		}

		outputs[i] = output
	}

	response := ReadResponse{
		ConversationID: convID,
		Page:           page,
		Limit:          limit,
		HasMore:        hasMore,
		Items:          outputs,
	}

	return toJSON(response)
}

// normalizePageParams extracts and normalizes page and limit parameters.
func normalizePageParams(args map[string]any) (page, limit int) {
	page = 1
	if p, ok := args["page"].(float64); ok && p >= 1 {
		page = int(p)
	}

	limit = DefaultPageLimit
	if l, ok := args["limit"].(float64); ok {
		if l > 0 {
			limit = int(l)
		}
		if limit > MaxPageLimit {
			limit = MaxPageLimit
		}
	}
	return page, limit
}

// calculateOffset computes the offset from page and limit.
func calculateOffset(page, limit int) int {
	return (page - 1) * limit
}

// toJSON marshals a value to JSON string.
func toJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal response: %w", err)
	}
	return string(data), nil
}
