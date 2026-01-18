package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/zombiekit/brains/internal/config"
	"github.com/zombiekit/brains/internal/recall"
	"github.com/zombiekit/brains/internal/recall/claude"
	"github.com/zombiekit/brains/internal/recall/postgres"
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
								Usage:   "Show detailed import progress",
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

func recallSaveAction(c *cli.Context) error {
	ctx := context.Background()
	cfg := config.LoadStorageConfigFromEnv()

	// Read content from args or stdin
	var content string
	if c.NArg() >= 1 {
		arg := c.Args().Get(0)
		if arg == "-" {
			// Read from stdin
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			content = strings.TrimSpace(string(data))
		} else {
			content = arg
		}
	} else {
		// Try reading from stdin if piped
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			reader := bufio.NewReader(os.Stdin)
			data, err := io.ReadAll(reader)
			if err != nil {
				return fmt.Errorf("failed to read stdin: %w", err)
			}
			content = strings.TrimSpace(string(data))
		}
	}

	if content == "" {
		return fmt.Errorf("content is required\nUsage: brains recall save <text> or echo 'text' | brains recall save -")
	}

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

	// Generate embedding
	embedding, err := embedder.Embed(ctx, content, recall.PurposeDocument)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	// Save to database
	id, created, err := storage.Save(ctx, content, embedding)
	if err != nil {
		return fmt.Errorf("failed to save content: %w", err)
	}

	// Output only if new content was created (silent on duplicate per spec)
	if created {
		// Truncate content for display
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
	once := c.Bool("once")
	interval := c.Duration("interval")

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

	// Initial import
	newCount, skipCount, err := importClaudeHistory(ctx, c.App.Writer, storage, embedder, claudePath, projectPath, verbose)
	if err != nil {
		return err
	}

	fmt.Fprintf(c.App.Writer, "Done. Imported %d new messages (%d duplicates skipped).\n", newCount, skipCount)

	if once {
		return nil
	}

	// Watch mode - set up signal handling
	fmt.Fprintf(c.App.Writer, "Watching for new conversations (interval: %s)...\n", interval)

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			newCount, _, err := importClaudeHistory(ctx, c.App.Writer, storage, embedder, claudePath, projectPath, verbose)
			if err != nil {
				fmt.Fprintf(c.App.Writer, "Error during import: %v\n", err)
				continue
			}
			if newCount > 0 {
				fmt.Fprintf(c.App.Writer, "Imported %d new messages.\n", newCount)
			}
		case <-done:
			fmt.Fprintln(c.App.Writer, "\nShutting down...")
			return nil
		}
	}
}

// importClaudeHistory imports conversation history from Claude Code.
func importClaudeHistory(
	ctx context.Context,
	w io.Writer,
	storage recall.Storage,
	embedder *recall.OllamaEmbedder,
	claudePath, projectPath string,
	verbose bool,
) (newCount, skipCount int, err error) {
	// Discover history files
	var files []string
	if projectPath != "" {
		files, err = claude.DiscoverProjectFiles(claudePath, projectPath)
	} else {
		files, err = claude.DiscoverHistoryFiles(claudePath)
	}
	if err != nil {
		return 0, 0, fmt.Errorf("discover history files: %w", err)
	}

	if len(files) == 0 {
		if verbose {
			fmt.Fprintln(w, "No history files found.")
		}
		return 0, 0, nil
	}

	if verbose {
		fmt.Fprintf(w, "Found %d history files.\n", len(files))
	}

	// Process each file
	for _, filePath := range files {
		entries, err := claude.ParseFile(filePath)
		if err != nil {
			if verbose {
				fmt.Fprintf(w, "Warning: failed to parse %s: %v\n", filePath, err)
			}
			continue
		}

		importable := claude.FilterImportable(entries)
		if verbose && len(importable) > 0 {
			fmt.Fprintf(w, "  %s: %d messages\n", filePath, len(importable))
		}

		for _, entry := range importable {
			// Check if already exists (fast path, avoid embedding generation)
			exists, err := storage.ExistsBySourceID(ctx, "claude", entry.UUID)
			if err != nil {
				return newCount, skipCount, fmt.Errorf("check exists: %w", err)
			}
			if exists {
				skipCount++
				if verbose {
					fmt.Fprintf(w, "    [%s] %s... (skipped)\n", entry.Message.Role, truncate(claude.ExtractContent(entry), 40))
				}
				continue
			}

			// Extract content
			content := claude.ExtractContent(entry)
			if content == "" {
				continue
			}

			// Chunk message if needed
			chunks := claude.ChunkMessage(content)
			for i, chunkContent := range chunks {
				sourceID := claude.ChunkSourceID(entry.UUID, i, len(chunks))

				// Generate embedding
				embedding, err := embedder.Embed(ctx, chunkContent, recall.PurposeDocument)
				if err != nil {
					return newCount, skipCount, fmt.Errorf("generate embedding: %w", err)
				}

				// Build metadata
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

				// Save with source tracking
				input := recall.ChunkInput{
					Content:        chunkContent,
					Source:         "claude",
					SourceID:       sourceID,
					ConversationID: entry.SessionID,
					Metadata:       metadata,
				}

				_, created, err := storage.SaveWithSource(ctx, input, embedding)
				if err != nil {
					return newCount, skipCount, fmt.Errorf("save chunk: %w", err)
				}

				if created {
					newCount++
					if verbose {
						fmt.Fprintf(w, "    [%s] %s... (imported)\n", entry.Message.Role, truncate(chunkContent, 40))
					}
				} else {
					skipCount++
				}
			}
		}
	}

	return newCount, skipCount, nil
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
