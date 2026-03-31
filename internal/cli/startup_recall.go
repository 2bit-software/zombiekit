package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/2bit-software/zombiekit/internal/config"
	"github.com/2bit-software/zombiekit/internal/recall"
	"github.com/2bit-software/zombiekit/internal/recall/claude"
	"github.com/2bit-software/zombiekit/internal/recall/postgres"
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

// recallDeps holds initialized dependencies for the recall import loop.
type recallDeps struct {
	storage   recall.Storage
	embedder  *recall.OllamaEmbedder
	claudePath string
}

// initRecallDeps creates storage, embedder, and resolves the Claude history
// path. The caller is responsible for closing storage.
func initRecallDeps(ctx context.Context) (*recallDeps, error) {
	storageConfig := config.LoadStorageConfigFromEnv()

	st, err := getServiceRecallStorage(ctx, storageConfig)
	if err != nil {
		return nil, fmt.Errorf("initialize storage: %w", err)
	}

	embedder, err := recall.NewOllamaEmbedder(storageConfig.OllamaURL, storageConfig.EmbeddingModel)
	if err != nil {
		st.Close()
		return nil, fmt.Errorf("create embedder: %w", err)
	}

	if err := embedder.CheckAvailable(ctx); err != nil {
		st.Close()
		return nil, fmt.Errorf("embedder not available: %w", err)
	}

	return &recallDeps{
		storage:    st,
		embedder:   embedder,
		claudePath: claude.DefaultClaudePath(),
	}, nil
}

// Run starts the recall watcher and blocks until the context is cancelled.
func (s *RecallService) Run(ctx context.Context) error {
	log := ServiceLogger(s.Name())

	deps, err := initRecallDeps(ctx)
	if err != nil {
		return err
	}
	defer deps.storage.Close()

	log.Info("watching for new conversations",
		"interval", s.config.Interval,
		"source", s.config.Source,
	)

	s.runInitialImport(ctx, deps, log)

	return s.watchLoop(ctx, deps, log)
}

func (s *RecallService) runInitialImport(ctx context.Context, deps *recallDeps, log interface{ Info(string, ...any); Warn(string, ...any) }) {
	result, err := s.importClaude(ctx, deps.storage, deps.embedder, deps.claudePath)
	if err != nil {
		log.Warn("initial import failed", "error", err)
		return
	}
	if result.newCount > 0 {
		log.Info("initial import complete",
			"new", result.newCount,
			"files", result.changedFiles,
		)
	}
}

func (s *RecallService) watchLoop(ctx context.Context, deps *recallDeps, log interface{ Info(string, ...any); Warn(string, ...any) }) error {
	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			result, err := s.importClaude(ctx, deps.storage, deps.embedder, deps.claudePath)
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

// svcImportResult holds statistics from an import operation.
type svcImportResult struct {
	newCount        int
	skipCount       int
	unchangedFiles  int
	changedFiles    int
	divergenceCount int
}

// accumulate folds a single file's result into the running totals.
func (r *svcImportResult) accumulate(fr *svcFileResult) {
	r.newCount += fr.newCount
	r.skipCount += fr.skipCount
	if fr.unchanged {
		r.unchangedFiles++
	} else {
		r.changedFiles++
	}
	if fr.divergence {
		r.divergenceCount++
	}
}

func (s *RecallService) importClaude(
	ctx context.Context,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	claudePath string,
) (*svcImportResult, error) {
	result := &svcImportResult{}

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
		svcFileResult, err := s.processFile(ctx, storage, embedder, filePath)
		if err != nil {
			if s.config.Verbose {
				ServiceLogger(s.Name()).Warn("failed to process file",
					"path", filePath,
					"error", err,
				)
			}
			continue
		}

		result.accumulate(svcFileResult)
	}

	return result, nil
}

type svcFileResult struct {
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
) (*svcFileResult, error) {
	return processClaudeFile(ctx, storage, embedder, filePath, s.config.Verbose)
}

func getServiceRecallStorage(ctx context.Context, cfg config.StorageConfig) (recall.Storage, error) {
	if cfg.Backend != config.BackendPostgres {
		return nil, fmt.Errorf("recall requires PostgreSQL backend (got %s)", cfg.Backend)
	}
	return postgres.New(ctx, cfg)
}

// parseWithDivergenceRetry parses a Claude JSONL file from the last known UUID.
// If the sync point is not found (divergence), it retries from the beginning.
func parseWithDivergenceRetry(filePath, lastKnownUUID string) (entries []claude.HistoryEntry, lastUUID string, diverged bool, err error) {
	entries, lastUUID, err = claude.ParseFileFromUUID(filePath, lastKnownUUID)
	if err == claude.ErrSyncPointNotFound {
		entries, lastUUID, err = claude.ParseFileFromUUID(filePath, "")
		if err != nil {
			return nil, "", false, fmt.Errorf("parse file: %w", err)
		}
		return entries, lastUUID, true, nil
	}
	if err != nil {
		return nil, "", false, fmt.Errorf("parse file: %w", err)
	}
	return entries, lastUUID, false, nil
}

// processClaudeFile handles the import of a single Claude JSONL file.
func processClaudeFile(
	ctx context.Context,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	filePath string,
	verbose bool,
) (*svcFileResult, error) {
	result := &svcFileResult{}

	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	fileMtime := stat.ModTime().UnixNano()

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

	entries, lastUUID, diverged, parseErr := parseWithDivergenceRetry(filePath, lastKnownUUID)
	if parseErr != nil {
		return nil, parseErr
	}
	result.divergence = diverged

	newCount, skipCount, importErr := importClaudeEntries(ctx, storage, embedder, entries, result.divergence)
	if importErr != nil {
		return nil, importErr
	}
	result.newCount = newCount
	result.skipCount = skipCount

	if lastUUID != "" {
		importState := &recall.ImportState{
			FilePath:      filePath,
			LastEntryUUID: lastUUID,
			FileMtime:     fileMtime,
		}
		if err := storage.SaveImportState(ctx, importState); err != nil {
			return nil, fmt.Errorf("save import state: %w", err)
		}
	}

	return result, nil
}

// importClaudeEntries iterates entries, extracts content, chunks, embeds, and
// saves each chunk with source tracking. It returns counts of new and skipped
// chunks, or the first error encountered.
func importClaudeEntries(
	ctx context.Context,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	entries []claude.HistoryEntry,
	divergence bool,
) (newCount, skipCount int, err error) {
	firstEntry := true
	for _, entry := range entries {
		content := claude.ExtractContent(entry)
		if content == "" {
			continue
		}

		chunks := claude.ChunkMessage(content)
		for i, chunkContent := range chunks {
			sourceID := claude.ChunkSourceID(entry.UUID, i, len(chunks))

			embedding, embedErr := embedder.Embed(ctx, chunkContent, recall.PurposeDocument)
			if embedErr != nil {
				return newCount, skipCount, fmt.Errorf("generate embedding: %w", embedErr)
			}

			metadata := buildEntryMetadata(entry)

			input := recall.ChunkInput{
				Content:        chunkContent,
				Source:         "claude",
				SourceID:       sourceID,
				ConversationID: entry.SessionID,
				Metadata:       metadata,
				HistoryGap:     divergence && firstEntry,
			}

			_, created, saveErr := storage.SaveWithSource(ctx, input, embedding)
			if saveErr != nil {
				return newCount, skipCount, fmt.Errorf("save chunk: %w", saveErr)
			}

			if created {
				newCount++
				firstEntry = false
			} else {
				skipCount++
			}
		}
	}
	return newCount, skipCount, nil
}

// buildEntryMetadata converts a HistoryEntry into recall Metadata.
func buildEntryMetadata(entry claude.HistoryEntry) *recall.Metadata {
	var parentID string
	if entry.ParentUUID != nil {
		parentID = *entry.ParentUUID
	}
	return &recall.Metadata{
		Role:      entry.Message.Role,
		Timestamp: entry.Timestamp,
		GitBranch: entry.GitBranch,
		CWD:       entry.CWD,
		ParentID:  parentID,
	}
}
