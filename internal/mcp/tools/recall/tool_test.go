package recall

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/zombiekit/brains/internal/recall"
)

// mockStorage implements recall.Storage for testing.
type mockStorage struct {
	conversations map[string][]recall.Chunk
	summaries     []recall.ConversationSummary
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		conversations: make(map[string][]recall.Chunk),
		summaries:     []recall.ConversationSummary{},
	}
}

func (m *mockStorage) Save(_ context.Context, _ string, _ []float32) (string, bool, error) {
	return "", false, nil
}

func (m *mockStorage) SaveWithSource(_ context.Context, _ recall.ChunkInput, _ []float32) (string, bool, error) {
	return "", false, nil
}

func (m *mockStorage) ExistsBySourceID(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

func (m *mockStorage) GetByConversation(_ context.Context, conversationID string) ([]recall.Chunk, error) {
	return m.conversations[conversationID], nil
}

func (m *mockStorage) List(_ context.Context, _ int) ([]recall.Chunk, error) {
	return nil, nil
}

func (m *mockStorage) Search(_ context.Context, _ []float32, _ int) ([]recall.SearchResult, error) {
	return nil, nil
}

func (m *mockStorage) ListConversations(_ context.Context, limit, offset int, project string) ([]recall.ConversationSummary, error) {
	result := m.summaries

	// Apply project filter
	if project != "" {
		filtered := []recall.ConversationSummary{}
		for _, s := range result {
			if len(s.Project) >= len(project) && s.Project[:len(project)] == project {
				filtered = append(filtered, s)
			}
		}
		result = filtered
	}

	// Apply offset
	if offset >= len(result) {
		return []recall.ConversationSummary{}, nil
	}
	result = result[offset:]

	// Apply limit
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

func (m *mockStorage) ListDistinctProjects(_ context.Context) ([]string, error) {
	return nil, nil
}

func (m *mockStorage) GetConversationChunks(_ context.Context, conversationID string, limit, offset int) ([]recall.Chunk, error) {
	chunks := m.conversations[conversationID]
	if offset >= len(chunks) {
		return []recall.Chunk{}, nil
	}
	end := offset + limit
	if end > len(chunks) {
		end = len(chunks)
	}
	return chunks[offset:end], nil
}

func (m *mockStorage) ConversationExists(_ context.Context, conversationID string) (bool, error) {
	_, exists := m.conversations[conversationID]
	return exists, nil
}

func (m *mockStorage) GetImportState(_ context.Context, _ string) (*recall.ImportState, error) {
	return nil, nil
}

func (m *mockStorage) SaveImportState(_ context.Context, _ *recall.ImportState) error {
	return nil
}

func (m *mockStorage) DeleteImportState(_ context.Context, _ string) error {
	return nil
}

func (m *mockStorage) CleanupStaleImportStates(_ context.Context, _ []string) error {
	return nil
}

func (m *mockStorage) Close() error {
	return nil
}

// addConversation adds a test conversation to the mock storage.
func (m *mockStorage) addConversation(convID string, chunks []recall.Chunk) {
	m.conversations[convID] = chunks
}

// addSummary adds a test summary to the mock storage.
func (m *mockStorage) addSummary(summary recall.ConversationSummary) {
	m.summaries = append(m.summaries, summary)
}

// === ListConversations Tests (T008) ===

func TestListConversations_DefaultPagination(t *testing.T) {
	storage := newMockStorage()
	for i := range 25 {
		storage.addSummary(recall.ConversationSummary{
			ConversationID: uuid.New().String(),
			Title:          "Conversation " + string(rune('A'+i)),
			MessageCount:   i + 1,
			FirstMessage:   time.Now().Add(-time.Hour),
			LastMessage:    time.Now(),
			Source:         "claude",
			Project:        "/project",
		})
	}
	tool := NewTool(storage)

	result, err := tool.ListConversations(context.Background(), map[string]any{})
	require.NoError(t, err)

	var response ListResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.Equal(t, 1, response.Page)
	assert.Equal(t, DefaultPageLimit, response.Limit)
	assert.True(t, response.HasMore)
	assert.Len(t, response.Items, 20)
}

func TestListConversations_CustomPage(t *testing.T) {
	storage := newMockStorage()
	for i := range 50 {
		storage.addSummary(recall.ConversationSummary{
			ConversationID: uuid.New().String(),
			Title:          "Conversation " + string(rune('A'+i%26)),
			MessageCount:   i + 1,
			FirstMessage:   time.Now().Add(-time.Hour),
			LastMessage:    time.Now(),
			Source:         "claude",
			Project:        "/project",
		})
	}
	tool := NewTool(storage)

	result, err := tool.ListConversations(context.Background(), map[string]any{
		"page":  float64(2),
		"limit": float64(10),
	})
	require.NoError(t, err)

	var response ListResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.Equal(t, 2, response.Page)
	assert.Equal(t, 10, response.Limit)
	assert.True(t, response.HasMore)
	assert.Len(t, response.Items, 10)
}

func TestListConversations_ProjectFilter(t *testing.T) {
	storage := newMockStorage()
	storage.addSummary(recall.ConversationSummary{
		ConversationID: uuid.New().String(),
		Title:          "Project A",
		Project:        "/project/a",
	})
	storage.addSummary(recall.ConversationSummary{
		ConversationID: uuid.New().String(),
		Title:          "Project B",
		Project:        "/project/b",
	})
	storage.addSummary(recall.ConversationSummary{
		ConversationID: uuid.New().String(),
		Title:          "Other",
		Project:        "/other",
	})
	tool := NewTool(storage)

	result, err := tool.ListConversations(context.Background(), map[string]any{
		"project": "/project",
	})
	require.NoError(t, err)

	var response ListResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.Len(t, response.Items, 2)
}

func TestListConversations_LimitCapped(t *testing.T) {
	storage := newMockStorage()
	tool := NewTool(storage)

	result, err := tool.ListConversations(context.Background(), map[string]any{
		"limit": float64(500),
	})
	require.NoError(t, err)

	var response ListResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.Equal(t, MaxPageLimit, response.Limit)
}

func TestListConversations_EmptyResult(t *testing.T) {
	storage := newMockStorage()
	tool := NewTool(storage)

	result, err := tool.ListConversations(context.Background(), map[string]any{})
	require.NoError(t, err)

	var response ListResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.False(t, response.HasMore)
	assert.Empty(t, response.Items)
}

// === ReadConversation Tests (T009) ===

func TestReadConversation_DefaultPagination(t *testing.T) {
	convID := uuid.New().String()
	storage := newMockStorage()
	chunks := make([]recall.Chunk, 25)
	for i := range chunks {
		chunks[i] = recall.Chunk{
			ID:             uuid.New().String(),
			Content:        "Message " + string(rune('A'+i)),
			ConversationID: convID,
			Metadata: &recall.Metadata{
				Role:      "user",
				Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
				CWD:       "/project",
				GitBranch: "main",
			},
		}
	}
	storage.addConversation(convID, chunks)
	tool := NewTool(storage)

	result, err := tool.ReadConversation(context.Background(), map[string]any{
		"conversation_id": convID,
	})
	require.NoError(t, err)

	var response ReadResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.Equal(t, convID, response.ConversationID)
	assert.Equal(t, 1, response.Page)
	assert.Equal(t, DefaultPageLimit, response.Limit)
	assert.True(t, response.HasMore)
	assert.Len(t, response.Items, 20)
}

func TestReadConversation_CustomPage(t *testing.T) {
	convID := uuid.New().String()
	storage := newMockStorage()
	chunks := make([]recall.Chunk, 50)
	for i := range chunks {
		chunks[i] = recall.Chunk{
			ID:             uuid.New().String(),
			Content:        "Message " + string(rune('A'+i%26)),
			ConversationID: convID,
			Metadata:       &recall.Metadata{Role: "user"},
		}
	}
	storage.addConversation(convID, chunks)
	tool := NewTool(storage)

	result, err := tool.ReadConversation(context.Background(), map[string]any{
		"conversation_id": convID,
		"page":            float64(5),
		"limit":           float64(10),
	})
	require.NoError(t, err)

	var response ReadResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.Equal(t, 5, response.Page)
	assert.Equal(t, 10, response.Limit)
	assert.False(t, response.HasMore)
	assert.Len(t, response.Items, 10)
}

func TestReadConversation_InvalidUUID(t *testing.T) {
	storage := newMockStorage()
	tool := NewTool(storage)

	result, err := tool.ReadConversation(context.Background(), map[string]any{
		"conversation_id": "not-a-uuid",
	})
	require.NoError(t, err)

	var response ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.Equal(t, "invalid conversation_id format", response.Error)
}

func TestReadConversation_NotFound(t *testing.T) {
	storage := newMockStorage()
	tool := NewTool(storage)

	result, err := tool.ReadConversation(context.Background(), map[string]any{
		"conversation_id": uuid.New().String(),
	})
	require.NoError(t, err)

	var response ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.Equal(t, "conversation not found", response.Error)
}

func TestReadConversation_MissingID(t *testing.T) {
	storage := newMockStorage()
	tool := NewTool(storage)

	result, err := tool.ReadConversation(context.Background(), map[string]any{})
	require.NoError(t, err)

	var response ErrorResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.Equal(t, "conversation_id is required", response.Error)
}

func TestReadConversation_ChronologicalOrder(t *testing.T) {
	convID := uuid.New().String()
	storage := newMockStorage()

	// Add chunks with specific timestamps in random order
	t1 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 1, 1, 11, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 1, 1, 9, 0, 0, 0, time.UTC) // Earliest

	// Storage stores them in order they'll be returned (simulating DB ordering)
	storage.addConversation(convID, []recall.Chunk{
		{ID: "3", Content: "First", Metadata: &recall.Metadata{Timestamp: t3}},
		{ID: "1", Content: "Second", Metadata: &recall.Metadata{Timestamp: t1}},
		{ID: "2", Content: "Third", Metadata: &recall.Metadata{Timestamp: t2}},
	})
	tool := NewTool(storage)

	result, err := tool.ReadConversation(context.Background(), map[string]any{
		"conversation_id": convID,
	})
	require.NoError(t, err)

	var response ReadResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	require.Len(t, response.Items, 3)
	assert.Equal(t, "First", response.Items[0].Content)
	assert.Equal(t, "Second", response.Items[1].Content)
	assert.Equal(t, "Third", response.Items[2].Content)
}

// === Edge Case Tests (T010) ===

func TestListConversations_PageBeyondData(t *testing.T) {
	storage := newMockStorage()
	for i := range 5 {
		storage.addSummary(recall.ConversationSummary{
			ConversationID: uuid.New().String(),
			Title:          "Conv " + string(rune('A'+i)),
		})
	}
	tool := NewTool(storage)

	result, err := tool.ListConversations(context.Background(), map[string]any{
		"page":  float64(100),
		"limit": float64(10),
	})
	require.NoError(t, err)

	var response ListResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.False(t, response.HasMore)
	assert.Empty(t, response.Items)
}

func TestListConversations_LimitZero(t *testing.T) {
	storage := newMockStorage()
	tool := NewTool(storage)

	result, err := tool.ListConversations(context.Background(), map[string]any{
		"limit": float64(0),
	})
	require.NoError(t, err)

	var response ListResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.Equal(t, DefaultPageLimit, response.Limit)
}

func TestListConversations_NegativePage(t *testing.T) {
	storage := newMockStorage()
	storage.addSummary(recall.ConversationSummary{
		ConversationID: uuid.New().String(),
		Title:          "Test",
	})
	tool := NewTool(storage)

	result, err := tool.ListConversations(context.Background(), map[string]any{
		"page": float64(-5),
	})
	require.NoError(t, err)

	var response ListResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.Equal(t, 1, response.Page)
	assert.Len(t, response.Items, 1)
}

func TestReadConversation_PageBeyondData(t *testing.T) {
	convID := uuid.New().String()
	storage := newMockStorage()
	storage.addConversation(convID, []recall.Chunk{
		{ID: "1", Content: "Test", Metadata: &recall.Metadata{}},
	})
	tool := NewTool(storage)

	result, err := tool.ReadConversation(context.Background(), map[string]any{
		"conversation_id": convID,
		"page":            float64(100),
	})
	require.NoError(t, err)

	var response ReadResponse
	require.NoError(t, json.Unmarshal([]byte(result), &response))

	assert.False(t, response.HasMore)
	assert.Empty(t, response.Items)
}

func TestNormalizePageParams(t *testing.T) {
	tests := []struct {
		name      string
		args      map[string]any
		wantPage  int
		wantLimit int
	}{
		{
			name:      "empty args",
			args:      map[string]any{},
			wantPage:  1,
			wantLimit: DefaultPageLimit,
		},
		{
			name:      "page only",
			args:      map[string]any{"page": float64(5)},
			wantPage:  5,
			wantLimit: DefaultPageLimit,
		},
		{
			name:      "limit only",
			args:      map[string]any{"limit": float64(50)},
			wantPage:  1,
			wantLimit: 50,
		},
		{
			name:      "both specified",
			args:      map[string]any{"page": float64(3), "limit": float64(25)},
			wantPage:  3,
			wantLimit: 25,
		},
		{
			name:      "limit exceeds max",
			args:      map[string]any{"limit": float64(200)},
			wantPage:  1,
			wantLimit: MaxPageLimit,
		},
		{
			name:      "page zero treated as 1",
			args:      map[string]any{"page": float64(0)},
			wantPage:  1,
			wantLimit: DefaultPageLimit,
		},
		{
			name:      "negative page treated as 1",
			args:      map[string]any{"page": float64(-3)},
			wantPage:  1,
			wantLimit: DefaultPageLimit,
		},
		{
			name:      "negative limit uses default",
			args:      map[string]any{"limit": float64(-10)},
			wantPage:  1,
			wantLimit: DefaultPageLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, limit := normalizePageParams(tt.args)
			assert.Equal(t, tt.wantPage, page)
			assert.Equal(t, tt.wantLimit, limit)
		})
	}
}

func TestCalculateOffset(t *testing.T) {
	tests := []struct {
		page, limit, want int
	}{
		{1, 20, 0},
		{2, 20, 20},
		{3, 10, 20},
		{5, 25, 100},
	}

	for _, tt := range tests {
		got := calculateOffset(tt.page, tt.limit)
		assert.Equal(t, tt.want, got)
	}
}
