package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/2bit-software/zombiekit/internal/config"
	"github.com/2bit-software/zombiekit/internal/recall"
	"github.com/2bit-software/zombiekit/internal/recall/claude"
	"github.com/2bit-software/zombiekit/internal/recall/postgres"
)

// newRecallCommand creates the recall subcommand for semantic memory operations.
func newRecallCommand() *cli.Command {
	return &cli.Command{
		Name:  "recall",
		Usage: "Semantic memory storage and retrieval",
		Subcommands: []*cli.Command{
			{
				Name:      "save",
				Usage:     "Store text content for semantic search",
				ArgsUsage: "<text> or - for stdin",
				Action:    recallSaveAction,
			},
			{
				Name:   "list",
				Usage:  "List all stored content",
				Action: recallListAction,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "limit",
						Usage: "Maximum number of items to return",
						Value: 20,
					},
				},
			},
			{
				Name:      "search",
				Usage:     "Search stored content by meaning",
				ArgsUsage: "<query>",
				Action:    recallSearchAction,
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "limit",
						Usage: "Maximum number of results",
						Value: 5,
					},
				},
			},
			{
				Name:  "watch",
				Usage: "Watch and import content from external sources",
				Subcommands: []*cli.Command{
					{
						Name:   "claude",
						Usage:  "Import Claude Code conversation history",
						Action: recallWatchClaudeAction,
						Flags: []cli.Flag{
							&cli.BoolFlag{
								Name:  "once",
								Usage: "Import once and exit (no continuous watch)",
							},
							&cli.StringFlag{
								Name:  "path",
								Usage: "Path to Claude config directory",
								Value: claude.DefaultClaudePath(),
							},
							&cli.StringFlag{
								Name:  "project",
								Usage: "Filter to specific project path",
							},
							&cli.BoolFlag{
								Name:    "verbose",
								Aliases: []string{"v"},
								Usage:   "Show detailed import progress (per-file output)",
							},
							&cli.BoolFlag{
								Name:  "force",
								Usage: "Bypass import state tracking and re-import all entries",
							},
							&cli.DurationFlag{
								Name:  "interval",
								Usage: "Poll interval for watch mode",
								Value: 30 * time.Second,
							},
						},
					},
				},
			},
			{
				Name:      "conversation",
				Usage:     "View all messages in a conversation",
				ArgsUsage: "<conversation-id>",
				Action:    recallConversationAction,
			},
		},
	}
}

func getRecallStorage(ctx context.Context, cfg config.StorageConfig) (recall.Storage, error) {
	if cfg.Backend != config.BackendPostgres {
		return nil, fmt.Errorf("recall requires PostgreSQL backend (got %s)\nSet BRAINS_BACKEND=postgres and BRAINS_POSTGRES_URL", cfg.Backend)
	}
	return postgres.New(ctx, cfg)
}

func getEmbedder(cfg config.StorageConfig) (*recall.OllamaEmbedder, error) {
	return recall.NewOllamaEmbedder(cfg.OllamaURL, cfg.EmbeddingModel)
}

// readContentFromArgs reads content from CLI arguments or stdin.
// Returns empty string if no content is available.
func readContentFromArgs(c *cli.Context) (string, error) {
	if c.NArg() >= 1 {
		arg := c.Args().Get(0)
		if arg != "-" {
			return arg, nil
		}
		return readStdin(os.Stdin)
	}

	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) != 0 {
		return "", nil
	}
	return readStdin(os.Stdin)
}

func readStdin(r io.Reader) (string, error) {
	data, err := io.ReadAll(bufio.NewReader(r))
	if err != nil {
		return "", fmt.Errorf("failed to read stdin: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

func recallSaveAction(c *cli.Context) error {
	ctx := context.Background()
	cfg := config.LoadStorageConfigFromEnv()

	content, err := readContentFromArgs(c)
	if err != nil {
		return err
	}
	if content == "" {
		return fmt.Errorf("content is required\nUsage: brains recall save <text> or echo 'text' | brains recall save -")
	}

	embedder, err := getEmbedder(cfg)
	if err != nil {
		return err
	}

	if err := embedder.CheckAvailable(ctx); err != nil {
		return fmt.Errorf("cannot connect to Ollama at %s\nMake sure Ollama is running: ollama serve", cfg.OllamaURL)
	}

	storage, err := getRecallStorage(ctx, cfg)
	if err != nil {
		return err
	}
	defer storage.Close()

	embedding, err := embedder.Embed(ctx, content, recall.PurposeDocument)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	id, created, err := storage.Save(ctx, content, embedding)
	if err != nil {
		return fmt.Errorf("failed to save content: %w", err)
	}

	if created {
		displayContent := content
		if len(displayContent) > 60 {
			displayContent = displayContent[:57] + "..."
		}
		fmt.Fprintf(c.App.Writer, "Stored: %s (%s)\n", displayContent, id[:8])
	}

	return nil
}

func recallListAction(c *cli.Context) error {
	ctx := context.Background()
	cfg := config.LoadStorageConfigFromEnv()

	// Initialize storage
	storage, err := getRecallStorage(ctx, cfg)
	if err != nil {
		return err
	}
	defer storage.Close()

	limit := c.Int("limit")
	chunks, err := storage.List(ctx, limit)
	if err != nil {
		return fmt.Errorf("failed to list content: %w", err)
	}

	if len(chunks) == 0 {
		fmt.Fprintln(c.App.Writer, "No content stored yet.")
		return nil
	}

	w := tabwriter.NewWriter(c.App.Writer, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tCREATED\tCONTENT")
	for _, chunk := range chunks {
		// Truncate content for display
		displayContent := chunk.Content
		if len(displayContent) > 50 {
			displayContent = displayContent[:47] + "..."
		}
		// Replace newlines with spaces for single-line display
		displayContent = strings.ReplaceAll(displayContent, "\n", " ")

		fmt.Fprintf(w, "%s\t%s\t%s\n",
			chunk.ID[:8],
			chunk.CreatedAt.Format(time.DateTime),
			displayContent,
		)
	}
	w.Flush()

	return nil
}

func recallSearchAction(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("query is required\nUsage: brains recall search <query>")
	}
	query := c.Args().Get(0)

	ctx := context.Background()
	cfg := config.LoadStorageConfigFromEnv()

	// Initialize embedder and check availability
	embedder, err := getEmbedder(cfg)
	if err != nil {
		return err
	}

	if err := embedder.CheckAvailable(ctx); err != nil {
		return fmt.Errorf("cannot connect to Ollama at %s\nMake sure Ollama is running: ollama serve", cfg.OllamaURL)
	}

	// Initialize storage
	storage, err := getRecallStorage(ctx, cfg)
	if err != nil {
		return err
	}
	defer storage.Close()

	// Generate embedding for query
	embedding, err := embedder.Embed(ctx, query, recall.PurposeQuery)
	if err != nil {
		return fmt.Errorf("failed to generate query embedding: %w", err)
	}

	limit := c.Int("limit")
	results, err := storage.Search(ctx, embedding, limit)
	if err != nil {
		return fmt.Errorf("failed to search: %w", err)
	}

	if len(results) == 0 {
		fmt.Fprintln(c.App.Writer, "No matching content found.")
		return nil
	}

	w := tabwriter.NewWriter(c.App.Writer, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SIMILARITY\tCREATED\tID\tCONTENT")
	for _, result := range results {
		// Truncate content for display
		displayContent := result.Chunk.Content
		if len(displayContent) > 50 {
			displayContent = displayContent[:47] + "..."
		}
		// Replace newlines with spaces for single-line display
		displayContent = strings.ReplaceAll(displayContent, "\n", " ")

		fmt.Fprintf(w, "%.4f\t%s\t%s\t%s\n",
			result.Similarity,
			result.Chunk.CreatedAt.Format(time.DateTime),
			result.Chunk.ID[:8],
			displayContent,
		)
	}
	w.Flush()

	return nil
}

// recallWatchClaudeAction imports Claude Code conversation history.
func recallWatchClaudeAction(c *cli.Context) error {
	ctx := context.Background()
	cfg := config.LoadStorageConfigFromEnv()

	claudePath := c.String("path")
	projectPath := c.String("project")
	verbose := c.Bool("verbose")
	force := c.Bool("force")
	interval := c.Duration("interval")

	embedder, err := getEmbedder(cfg)
	if err != nil {
		return err
	}

	if err := embedder.CheckAvailable(ctx); err != nil {
		return fmt.Errorf("cannot connect to Ollama at %s\nMake sure Ollama is running: ollama serve", cfg.OllamaURL)
	}

	storage, err := getRecallStorage(ctx, cfg)
	if err != nil {
		return err
	}
	defer storage.Close()

	result, err := importClaudeHistory(ctx, c.App.Writer, storage, embedder, claudePath, projectPath, verbose, force)
	if err != nil {
		return err
	}
	result.printSummary(c.App.Writer, verbose)

	if c.Bool("once") {
		return nil
	}

	return watchClaudeHistory(ctx, c.App.Writer, storage, embedder, claudePath, projectPath, verbose, force, interval)
}

// watchClaudeHistory polls for new Claude conversations on an interval until
// interrupted by SIGINT/SIGTERM.
func watchClaudeHistory(
	ctx context.Context,
	w io.Writer,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	claudePath, projectPath string,
	verbose, force bool,
	interval time.Duration,
) error {
	fmt.Fprintf(w, "Watching for new conversations (interval: %s)...\n", interval)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			result, err := importClaudeHistory(ctx, w, storage, embedder, claudePath, projectPath, verbose, force)
			if err != nil {
				fmt.Fprintf(w, "Error during import: %v\n", err)
				continue
			}
			if result.newCount > 0 || result.divergenceCount > 0 {
				result.printSummary(w, verbose)
			}
		case <-done:
			fmt.Fprintln(w, "\nShutting down...")
			return nil
		}
	}
}

// claudeImportResult contains statistics from a Claude import operation.
type claudeImportResult struct {
	newCount        int // Number of new messages imported
	skipCount       int // Number of duplicates skipped (within changed files)
	unchangedFiles  int // Number of files skipped via mtime check
	changedFiles    int // Number of files processed
	divergenceCount int // Number of files with sync divergence
}

// printSummary outputs the import result summary.
func (r *claudeImportResult) printSummary(w io.Writer, verbose bool) {
	// Always show divergence warnings
	if r.divergenceCount > 0 {
		fmt.Fprintf(w, "Warning: %d file(s) had sync divergence (history_gap markers added)\n", r.divergenceCount)
	}

	// Summary line
	if r.newCount > 0 {
		fmt.Fprintf(w, "Imported %d new messages from %d files", r.newCount, r.changedFiles)
		if r.unchangedFiles > 0 {
			fmt.Fprintf(w, " (%d unchanged files skipped)", r.unchangedFiles)
		}
		fmt.Fprintln(w, ".")
	} else if r.unchangedFiles > 0 {
		fmt.Fprintf(w, "No new messages. %d files unchanged.\n", r.unchangedFiles)
	} else {
		fmt.Fprintln(w, "No new messages.")
	}
}

// importClaudeHistory imports conversation history from Claude Code.
func importClaudeHistory(
	ctx context.Context,
	w io.Writer,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	claudePath, projectPath string,
	verbose, force bool,
) (*claudeImportResult, error) {
	result := &claudeImportResult{}

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
	var files []string
	if projectPath != "" {
		files, err = claude.DiscoverProjectFiles(claudePath, projectPath)
	} else {
		files, err = claude.DiscoverHistoryFiles(claudePath)
	}
	if err != nil {
		return nil, fmt.Errorf("discover history files: %w", err)
	}

	if len(files) == 0 {
		if verbose {
			fmt.Fprintln(w, "No history files found.")
		}
		return result, nil
	}

	// Cleanup stale import states for deleted files
	if !force {
		if err := storage.CleanupStaleImportStates(ctx, files); err != nil {
			return nil, fmt.Errorf("cleanup stale import states: %w", err)
		}
	}

	if verbose {
		fmt.Fprintf(w, "Found %d history files.\n", len(files))
	}

	// Process each file
	processFiles(ctx, w, storage, embedder, files, verbose, force, result)

	return result, nil
}

// processFiles iterates over history files, processes each, and accumulates
// results into the provided claudeImportResult.
func processFiles(
	ctx context.Context,
	w io.Writer,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	files []string,
	verbose, force bool,
	result *claudeImportResult,
) {
	for _, filePath := range files {
		fr, err := processFile(ctx, w, storage, embedder, filePath, verbose, force)
		if err != nil {
			if verbose {
				fmt.Fprintf(w, "Warning: failed to process %s: %v\n", filePath, err)
			}
			continue
		}
		accumulateFileResult(result, fr)
	}
}

// accumulateFileResult merges a single file's import stats into the aggregate result.
func accumulateFileResult(result *claudeImportResult, fr *fileResult) {
	result.newCount += fr.newCount
	result.skipCount += fr.skipCount
	if fr.unchanged {
		result.unchangedFiles++
	} else {
		result.changedFiles++
	}
	if fr.divergence {
		result.divergenceCount++
	}
}

// fileResult contains statistics from processing a single file.
type fileResult struct {
	newCount   int
	skipCount  int
	unchanged  bool // File was skipped via mtime check
	divergence bool // Sync point not found, history_gap was set
}

// checkImportState determines whether a file needs re-importing by comparing
// its mtime against previously stored state. Returns the last known UUID for
// incremental parsing and whether the file is unchanged.
func checkImportState(
	ctx context.Context,
	storage recall.Storage,
	filePath string,
	fileMtime int64,
	force bool,
) (lastKnownUUID string, unchanged bool, err error) {
	if force {
		return "", false, nil
	}

	state, err := storage.GetImportState(ctx, filePath)
	if err != nil {
		return "", false, fmt.Errorf("get import state: %w", err)
	}
	if state == nil {
		return "", false, nil
	}

	if state.FileMtime == fileMtime {
		return "", true, nil
	}
	return state.LastEntryUUID, false, nil
}

// importEntries embeds and saves parsed history entries as chunks, returning
// counts of new and skipped records.
func importEntries(
	ctx context.Context,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	entries []claude.HistoryEntry,
	divergence, verbose bool,
	w io.Writer,
) (newCount, skipCount int, err error) {
	firstEntry := true
	for _, entry := range entries {
		content := claude.ExtractContent(entry)
		if content == "" {
			continue
		}

		n, s, err := importChunks(ctx, storage, embedder, entry, content, divergence && firstEntry, verbose, w)
		if err != nil {
			return newCount, skipCount, err
		}
		newCount += n
		skipCount += s
		if n > 0 {
			firstEntry = false
		}
	}
	return newCount, skipCount, nil
}

// importChunks splits a single entry's content into chunks, embeds each, and
// saves them. Returns new/skip counts for the entry.
func importChunks(
	ctx context.Context,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	entry claude.HistoryEntry,
	content string,
	historyGap, verbose bool,
	w io.Writer,
) (newCount, skipCount int, err error) {
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

	chunks := claude.ChunkMessage(content)
	gapApplied := false
	for i, chunkContent := range chunks {
		embedding, err := embedder.Embed(ctx, chunkContent, recall.PurposeDocument)
		if err != nil {
			return newCount, skipCount, fmt.Errorf("generate embedding: %w", err)
		}

		input := recall.ChunkInput{
			Content:        chunkContent,
			Source:         "claude",
			SourceID:       claude.ChunkSourceID(entry.UUID, i, len(chunks)),
			ConversationID: entry.SessionID,
			Metadata:       metadata,
			HistoryGap:     historyGap && !gapApplied,
		}

		_, created, err := storage.SaveWithSource(ctx, input, embedding)
		if err != nil {
			return newCount, skipCount, fmt.Errorf("save chunk: %w", err)
		}

		if created {
			newCount++
			gapApplied = true
			if verbose {
				fmt.Fprintf(w, "    [%s] %s... (imported)\n", entry.Message.Role, truncate(chunkContent, 40))
			}
		} else {
			skipCount++
		}
	}
	return newCount, skipCount, nil
}

// processFile handles the import of a single JSONL file.
func processFile(
	ctx context.Context,
	w io.Writer,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	filePath string,
	verbose, force bool,
) (*fileResult, error) {
	result := &fileResult{}

	stat, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}
	fileMtime := stat.ModTime().UnixNano()

	lastKnownUUID, unchanged, err := checkImportState(ctx, storage, filePath, fileMtime, force)
	if err != nil {
		return nil, err
	}
	if unchanged {
		result.unchanged = true
		if verbose {
			fmt.Fprintf(w, "  %s: unchanged (skipped)\n", filePath)
		}
		return result, nil
	}

	entries, lastUUID, divergence, err := parseWithDivergenceFallback(filePath, lastKnownUUID, w)
	if err != nil {
		return nil, err
	}
	result.divergence = divergence

	if verbose && len(entries) > 0 {
		fmt.Fprintf(w, "  %s: %d new entries\n", filePath, len(entries))
	}

	newCount, skipCount, err := importEntries(ctx, storage, embedder, entries, result.divergence, verbose, w)
	if err != nil {
		return nil, err
	}
	result.newCount = newCount
	result.skipCount = skipCount

	if lastUUID == "" || force {
		return result, nil
	}

	state := &recall.ImportState{
		FilePath:      filePath,
		LastEntryUUID: lastUUID,
		FileMtime:     fileMtime,
	}
	if err := storage.SaveImportState(ctx, state); err != nil {
		return nil, fmt.Errorf("save import state: %w", err)
	}

	return result, nil
}

// parseWithDivergenceFallback attempts incremental parsing from lastKnownUUID.
// On sync-point-not-found, it falls back to a full re-parse and flags a
// divergence so the caller can apply history_gap markers.
func parseWithDivergenceFallback(filePath, lastKnownUUID string, w io.Writer) (entries []claude.HistoryEntry, lastUUID string, divergence bool, err error) {
	entries, lastUUID, err = claude.ParseFileFromUUID(filePath, lastKnownUUID)
	if !errors.Is(err, claude.ErrSyncPointNotFound) {
		if err != nil {
			return nil, "", false, fmt.Errorf("parse file: %w", err)
		}
		return entries, lastUUID, false, nil
	}

	fmt.Fprintf(w, "Warning: sync point not found in %s - importing with history_gap marker\n", filePath)

	entries, lastUUID, err = claude.ParseFileFromUUID(filePath, "")
	if err != nil {
		return nil, "", false, fmt.Errorf("parse file: %w", err)
	}
	return entries, lastUUID, true, nil
}

// recallConversationAction displays all messages in a conversation.
func recallConversationAction(c *cli.Context) error {
	if c.NArg() < 1 {
		return fmt.Errorf("conversation ID is required\nUsage: brains recall conversation <conversation-id>")
	}
	conversationID := c.Args().Get(0)

	ctx := context.Background()
	cfg := config.LoadStorageConfigFromEnv()

	// Initialize storage
	storage, err := getRecallStorage(ctx, cfg)
	if err != nil {
		return err
	}
	defer storage.Close()

	chunks, err := storage.GetByConversation(ctx, conversationID)
	if err != nil {
		return fmt.Errorf("failed to get conversation: %w", err)
	}

	if len(chunks) == 0 {
		fmt.Fprintln(c.App.Writer, "No messages found for this conversation.")
		return nil
	}

	fmt.Fprintf(c.App.Writer, "Conversation: %s (%d messages)\n\n", conversationID, len(chunks))

	for _, chunk := range chunks {
		role := "unknown"
		timestamp := chunk.CreatedAt.Format(time.DateTime)
		if chunk.Metadata != nil {
			if chunk.Metadata.Role != "" {
				role = chunk.Metadata.Role
			}
			if !chunk.Metadata.Timestamp.IsZero() {
				timestamp = chunk.Metadata.Timestamp.Format(time.DateTime)
			}
		}

		// Format role label
		roleLabel := fmt.Sprintf("[%s]", role)

		// Truncate content for preview
		displayContent := chunk.Content
		if len(displayContent) > 200 {
			displayContent = displayContent[:197] + "..."
		}
		displayContent = strings.ReplaceAll(displayContent, "\n", " ")

		fmt.Fprintf(c.App.Writer, "%s %s\n  %s\n\n", roleLabel, timestamp, displayContent)
	}

	return nil
}

// truncate shortens a string to the specified length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
