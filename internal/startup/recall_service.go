package startup

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/zombiekit/brains/internal/config"
	"github.com/zombiekit/brains/internal/recall"
	"github.com/zombiekit/brains/internal/recall/claude"
	"github.com/zombiekit/brains/internal/recall/postgres"
)

// RecallService wraps the recall watcher as a Service.
type RecallService struct {
	config config.RecallConfig
}

// NewRecallService creates a new recall service with the given configuration.
func NewRecallService(cfg config.RecallConfig) *RecallService {
	return &RecallService{config: cfg}
}

// Name returns the service identifier.
func (s *RecallService) Name() string {
	return "recall"
}

// Run starts the recall watcher and blocks until the context is cancelled.
func (s *RecallService) Run(ctx context.Context) error {
	log := ServiceLogger(s.Name())

	// Load storage config from environment
	storageConfig := config.LoadStorageConfigFromEnv()

	// Initialize storage
	storage, err := getRecallStorage(ctx, storageConfig)
	if err != nil {
		return fmt.Errorf("initialize storage: %w", err)
	}
	defer storage.Close()

	// Initialize embedder
	embedder, err := recall.NewOllamaEmbedder(storageConfig.OllamaURL, storageConfig.EmbeddingModel)
	if err != nil {
		return fmt.Errorf("create embedder: %w", err)
	}

	if err := embedder.CheckAvailable(ctx); err != nil {
		return fmt.Errorf("embedder not available: %w", err)
	}

	// Get Claude path
	claudePath := claude.DefaultClaudePath()

	log.Info("watching for new conversations",
		"interval", s.config.Interval,
		"source", s.config.Source,
	)

	// Initial import
	result, err := s.importClaude(ctx, storage, embedder, claudePath)
	if err != nil {
		log.Warn("initial import failed", "error", err)
	} else if result.newCount > 0 {
		log.Info("initial import complete",
			"new", result.newCount,
			"files", result.changedFiles,
		)
	}

	// Watch loop
	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			result, err := s.importClaude(ctx, storage, embedder, claudePath)
			if err != nil {
				log.Warn("import failed", "error", err)
				continue
			}
			if result.newCount > 0 || result.divergenceCount > 0 {
				log.Info("import complete",
					"new", result.newCount,
					"files", result.changedFiles,
					"divergence", result.divergenceCount,
				)
			}
		case <-ctx.Done():
			log.Info("shutting down")
			return nil
		}
	}
}

// importResult holds statistics from an import operation.
type importResult struct {
	newCount        int
	skipCount       int
	unchangedFiles  int
	changedFiles    int
	divergenceCount int
}

func (s *RecallService) importClaude(
	ctx context.Context,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	claudePath string,
) (*importResult, error) {
	result := &importResult{}

	// Acquire import lock
	lockPath, err := claude.DefaultLockPath()
	if err != nil {
		return nil, fmt.Errorf("get lock path: %w", err)
	}

	lock, err := claude.AcquireLock(lockPath)
	if err != nil {
		return nil, err
	}
	defer lock.Release()

	// Discover history files
	files, err := claude.DiscoverHistoryFiles(claudePath)
	if err != nil {
		return nil, fmt.Errorf("discover history files: %w", err)
	}

	if len(files) == 0 {
		return result, nil
	}

	// Cleanup stale import states
	if err := storage.CleanupStaleImportStates(ctx, files); err != nil {
		return nil, fmt.Errorf("cleanup stale import states: %w", err)
	}

	// Process each file
	for _, filePath := range files {
		fileResult, err := s.processFile(ctx, storage, embedder, filePath)
		if err != nil {
			if s.config.Verbose {
				ServiceLogger(s.Name()).Warn("failed to process file",
					"path", filePath,
					"error", err,
				)
			}
			continue
		}

		result.newCount += fileResult.newCount
		result.skipCount += fileResult.skipCount
		if fileResult.unchanged {
			result.unchangedFiles++
		} else {
			result.changedFiles++
		}
		if fileResult.divergence {
			result.divergenceCount++
		}
	}

	return result, nil
}

type fileResult struct {
	newCount   int
	skipCount  int
	unchanged  bool
	divergence bool
}

func (s *RecallService) processFile(
	ctx context.Context,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	filePath string,
) (*fileResult, error) {
	return processClaudeFile(ctx, storage, embedder, filePath, s.config.Verbose)
}

func getRecallStorage(ctx context.Context, cfg config.StorageConfig) (recall.Storage, error) {
	if cfg.Backend != config.BackendPostgres {
		return nil, fmt.Errorf("recall requires PostgreSQL backend (got %s)", cfg.Backend)
	}
	return postgres.New(ctx, cfg)
}

// processClaudeFile handles the import of a single Claude JSONL file.
// This is extracted from the CLI code for reuse.
func processClaudeFile(
	ctx context.Context,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	filePath string,
	verbose bool,
) (*fileResult, error) {
	result := &fileResult{}

	// Get file info for mtime check
	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	fileMtime := stat.ModTime().UnixNano()

	// Check import state (skip if unchanged)
	state, err := storage.GetImportState(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("get import state: %w", err)
	}

	var lastKnownUUID string
	if state != nil {
		if state.FileMtime == fileMtime {
			result.unchanged = true
			return result, nil
		}
		lastKnownUUID = state.LastEntryUUID
	}

	// Parse file from sync point
	entries, lastUUID, err := claude.ParseFileFromUUID(filePath, lastKnownUUID)
	if err == claude.ErrSyncPointNotFound {
		result.divergence = true
		entries, lastUUID, err = claude.ParseFileFromUUID(filePath, "")
		if err != nil {
			return nil, fmt.Errorf("parse file: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("parse file: %w", err)
	}

	// Import entries
	firstEntry := true
	for _, entry := range entries {
		content := claude.ExtractContent(entry)
		if content == "" {
			continue
		}

		chunks := claude.ChunkMessage(content)
		for i, chunkContent := range chunks {
			sourceID := claude.ChunkSourceID(entry.UUID, i, len(chunks))

			embedding, err := embedder.Embed(ctx, chunkContent, recall.PurposeDocument)
			if err != nil {
				return nil, fmt.Errorf("generate embedding: %w", err)
			}

			var parentID string
			if entry.ParentUUID != nil {
				parentID = *entry.ParentUUID
			}
			metadata := &recall.Metadata{
				Role:      entry.Message.Role,
				Timestamp: entry.Timestamp,
				GitBranch: entry.GitBranch,
				CWD:       entry.CWD,
				ParentID:  parentID,
			}

			input := recall.ChunkInput{
				Content:        chunkContent,
				Source:         "claude",
				SourceID:       sourceID,
				ConversationID: entry.SessionID,
				Metadata:       metadata,
				HistoryGap:     result.divergence && firstEntry,
			}

			_, created, err := storage.SaveWithSource(ctx, input, embedding)
			if err != nil {
				return nil, fmt.Errorf("save chunk: %w", err)
			}

			if created {
				result.newCount++
				firstEntry = false
			} else {
				result.skipCount++
			}
		}
	}

	// Update import state
	if lastUUID != "" {
		state := &recall.ImportState{
			FilePath:      filePath,
			LastEntryUUID: lastUUID,
			FileMtime:     fileMtime,
		}
		if err := storage.SaveImportState(ctx, state); err != nil {
			return nil, fmt.Errorf("save import state: %w", err)
		}
	}

	return result, nil
}
