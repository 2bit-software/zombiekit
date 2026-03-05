package remote

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	commonv1 "github.com/zombiekit/brains/gen/zombiekit/brains/common/v1"
	searchv1 "github.com/zombiekit/brains/gen/zombiekit/brains/search/v1"
	"github.com/zombiekit/brains/gen/zombiekit/brains/search/v1/searchv1connect"
	"github.com/zombiekit/brains/internal/proxy/handlers"
)

type serverConnection interface {
	IsConfigured() bool
	Search() searchv1connect.SearchServiceClient
}

func NewRecallListHandler(conn serverConnection) handlers.Handler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		if !conn.IsConfigured() {
			return "", errors.New("server not configured: recall-list-conversations requires a ZK server connection. Set ZK_SERVER_URL to enable.")
		}

		limit := intArg(args, "limit", 20, 100)
		page := intArg(args, "page", 1, 0)
		project := stringArg(args, "project", "")

		resp, err := conn.Search().ListConversations(ctx,
			connect.NewRequest(&searchv1.ListConversationsRequest{
				Pagination:    &commonv1.PageRequest{PageSize: int32(limit)},
				ProjectFilter: project,
			}))
		if err != nil {
			return "", fmt.Errorf("server unreachable: %w", err)
		}

		return formatListResponse(resp.Msg, page, limit)
	}
}

func NewRecallReadHandler(conn serverConnection) handlers.Handler {
	return func(ctx context.Context, args map[string]any) (string, error) {
		if !conn.IsConfigured() {
			return "", errors.New("server not configured: recall-read-conversation requires a ZK server connection. Set ZK_SERVER_URL to enable.")
		}

		convID := stringArg(args, "conversation_id", "")
		if convID == "" {
			return toJSON(map[string]string{"error": "conversation_id is required"})
		}

		limit := intArg(args, "limit", 20, 100)
		page := intArg(args, "page", 1, 0)

		resp, err := conn.Search().GetConversation(ctx,
			connect.NewRequest(&searchv1.GetConversationRequest{
				ConversationId: convID,
				Pagination:     &commonv1.PageRequest{PageSize: int32(limit)},
			}))
		if err != nil {
			return "", fmt.Errorf("server unreachable: %w", err)
		}

		return formatReadResponse(resp.Msg, convID, page, limit)
	}
}

func formatListResponse(msg *searchv1.ListConversationsResponse, page, limit int) (string, error) {
	type summary struct {
		ConversationID string `json:"conversation_id"`
		Project        string `json:"project"`
		Summary        string `json:"summary"`
		TotalChunks    int32  `json:"total_chunks"`
		CreatedAt      string `json:"created_at"`
		UpdatedAt      string `json:"updated_at"`
	}

	items := make([]summary, 0, len(msg.GetConversations()))
	for _, c := range msg.GetConversations() {
		s := summary{
			ConversationID: c.GetId(),
			Project:        c.GetProject(),
			Summary:        c.GetSummary(),
			TotalChunks:    c.GetTotalChunks(),
		}
		if c.GetCreatedAt() != nil {
			s.CreatedAt = c.GetCreatedAt().AsTime().Format("2006-01-02T15:04:05Z07:00")
		}
		if c.GetUpdatedAt() != nil {
			s.UpdatedAt = c.GetUpdatedAt().AsTime().Format("2006-01-02T15:04:05Z07:00")
		}
		items = append(items, s)
	}

	hasMore := msg.GetPagination() != nil && msg.GetPagination().GetNextPageToken() != ""

	return toJSON(map[string]any{
		"page":     page,
		"limit":    limit,
		"has_more": hasMore,
		"items":    items,
	})
}

func formatReadResponse(msg *searchv1.GetConversationResponse, convID string, page, limit int) (string, error) {
	type chunk struct {
		ID       string `json:"id"`
		Content  string `json:"content"`
		Sequence int32  `json:"sequence"`
	}

	items := make([]chunk, 0, len(msg.GetChunks()))
	for _, c := range msg.GetChunks() {
		items = append(items, chunk{
			ID:       c.GetId(),
			Content:  c.GetContent(),
			Sequence: c.GetSequence(),
		})
	}

	hasMore := msg.GetPagination() != nil && msg.GetPagination().GetNextPageToken() != ""

	return toJSON(map[string]any{
		"conversation_id": convID,
		"page":            page,
		"limit":           limit,
		"has_more":        hasMore,
		"items":           items,
	})
}

func intArg(args map[string]any, key string, defaultVal, maxVal int) int {
	v, ok := args[key].(float64)
	if !ok || v < 1 {
		return defaultVal
	}
	n := int(v)
	if maxVal > 0 && n > maxVal {
		return maxVal
	}
	return n
}

func stringArg(args map[string]any, key, defaultVal string) string {
	v, ok := args[key].(string)
	if !ok || v == "" {
		return defaultVal
	}
	return v
}

func toJSON(v any) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("marshal response: %w", err)
	}
	return string(data), nil
}
