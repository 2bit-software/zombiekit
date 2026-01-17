package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/zombiekit/brains/internal/config"
	"github.com/zombiekit/brains/internal/recall"
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
