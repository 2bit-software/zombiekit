package server

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/2bit-software/zombiekit/internal/recall"
	commonv1 "github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/common/v1"
	searchv1 "github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/search/v1"
	"github.com/2bit-software/zombiekit/proto/gen/zombiekit/brains/search/v1/searchv1connect"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type SearchService struct {
	searchv1connect.UnimplementedSearchServiceHandler
	storage  recall.Storage
	embedder Embedder
}

type Embedder interface {
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}

func NewSearchService(storage recall.Storage, embedder Embedder) *SearchService {
	return &SearchService{storage: storage, embedder: embedder}
}

func (s *SearchService) Search(
	ctx context.Context,
	req *connect.Request[searchv1.SearchRequest],
) (*connect.Response[searchv1.SearchResponse], error) {
	if s.storage == nil || s.embedder == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("search not configured"))
	}

	msg := req.Msg
	if msg.Query == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("query is required"))
	}

	limit := int(msg.Limit)
	if limit <= 0 {
		limit = 10
	}

	embedding, err := s.embedder.EmbedQuery(ctx, msg.Query)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	results, err := s.storage.Search(ctx, embedding, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &searchv1.SearchResponse{
		Results: make([]*searchv1.SearchResult, 0, len(results)),
	}
	for _, r := range results {
		resp.Results = append(resp.Results, &searchv1.SearchResult{
			Chunk: chunkToProto(&r.Chunk),
			Score: float32(r.Similarity),
		})
	}

	return connect.NewResponse(resp), nil
}

func (s *SearchService) ListConversations(
	ctx context.Context,
	req *connect.Request[searchv1.ListConversationsRequest],
) (*connect.Response[searchv1.ListConversationsResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("storage not configured"))
	}

	msg := req.Msg
	limit := 20
	offset := 0

	if msg.Pagination != nil {
		if msg.Pagination.PageSize > 0 {
			limit = int(msg.Pagination.PageSize)
		}
	}

	summaries, err := s.storage.ListConversations(ctx, limit, offset, msg.ProjectFilter)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &searchv1.ListConversationsResponse{
		Conversations: make([]*searchv1.Conversation, 0, len(summaries)),
		Pagination:    &commonv1.PageResponse{TotalCount: int32(len(summaries))},
	}
	for _, sum := range summaries {
		resp.Conversations = append(resp.Conversations, &searchv1.Conversation{
			Id:          sum.ConversationID,
			Project:     sum.Project,
			CreatedAt:   timestamppb.New(sum.FirstMessage),
			UpdatedAt:   timestamppb.New(sum.LastMessage),
			Summary:     sum.Title,
			TotalChunks: int32(sum.MessageCount),
		})
	}

	return connect.NewResponse(resp), nil
}

func (s *SearchService) GetConversation(
	ctx context.Context,
	req *connect.Request[searchv1.GetConversationRequest],
) (*connect.Response[searchv1.GetConversationResponse], error) {
	if s.storage == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("storage not configured"))
	}

	msg := req.Msg
	if msg.ConversationId == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("conversation_id is required"))
	}

	exists, err := s.storage.ConversationExists(ctx, msg.ConversationId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if !exists {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("conversation not found"))
	}

	limit := 20
	offset := 0
	if msg.Pagination != nil {
		if msg.Pagination.PageSize > 0 {
			limit = int(msg.Pagination.PageSize)
		}
	}

	chunks, err := s.storage.GetConversationChunks(ctx, msg.ConversationId, limit, offset)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	resp := &searchv1.GetConversationResponse{
		Conversation: &searchv1.Conversation{
			Id:          msg.ConversationId,
			TotalChunks: int32(len(chunks)),
		},
		Chunks:     make([]*searchv1.ConversationChunk, 0, len(chunks)),
		Pagination: &commonv1.PageResponse{TotalCount: int32(len(chunks))},
	}
	for i, c := range chunks {
		resp.Chunks = append(resp.Chunks, chunkToProto(&c))
		resp.Chunks[i].Sequence = int32(i + 1)
	}

	if len(chunks) > 0 {
		resp.Conversation.CreatedAt = timestamppb.New(chunks[0].CreatedAt)
		resp.Conversation.UpdatedAt = timestamppb.New(chunks[len(chunks)-1].CreatedAt)
	}

	return connect.NewResponse(resp), nil
}

func chunkToProto(c *recall.Chunk) *searchv1.ConversationChunk {
	return &searchv1.ConversationChunk{
		Id:             c.ID,
		ConversationId: c.ConversationID,
		Content:        c.Content,
		CreatedAt:      timestamppb.New(c.CreatedAt),
	}
}
