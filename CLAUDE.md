# zombiekit Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-12-21

## Active Technologies
- Go 1.22+ + urfave/cli/v2 (CLI), mark3labs/mcp-go (MCP), pgx/v5 (database), slog (logging) (002-mcp-tools)
- PostgreSQL 16 with pgvector extension (002-mcp-tools)
- Go 1.22+ + urfave/cli/v2 (CLI), mark3labs/mcp-go (MCP), pgx/v5 (PostgreSQL), modernc.org/sqlite (SQLite), slog (logging) (002-mcp-tools)
- PostgreSQL 16 with pgvector (production) OR SQLite with WAL mode (development/single-user, default) (002-mcp-tools)
- Go 1.24+ (per go.mod) + urfave/cli/v2 (CLI), mark3labs/mcp-go (MCP), gopkg.in/yaml.v3 (YAML parsing) (003-profiles)
- File-based profiles (.md files), JSON registry (~/.brains/registry.json) with flock (003-profiles)
- Go 1.24.0 (per go.mod) + urfave/cli/v2 (CLI), adrg/frontmatter (YAML parsing), gopkg.in/yaml.v3 (YAML) (004-source-interface)
- File-based (.md files with YAML frontmatter) (004-source-interface)
- Go 1.24.0 + mark3labs/mcp-go v0.43.2 (MCP server) (006-remove-mcp-tools)
- N/A (no storage changes) (006-remove-mcp-tools)
- Go 1.24.0 (per go.mod) + urfave/cli/v2 (CLI), BurntSushi/toml (TOML parsing - already indirect dep), slog (logging) (007-cli-config)
- TOML files at `.brains/config.toml` (local), `~/.config/brains/config.toml` (global Unix), `%APPDATA%\brains\config.toml` (Windows) (007-cli-config)
- N/A (uses existing profile.Service from internal/profile) (008-plugin-web-gui)

- Go 1.22+ (per MASTER-DESIGN.md) (001-core-repo-setup)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.22+ (per MASTER-DESIGN.md)

## Code Style

Go 1.22+ (per MASTER-DESIGN.md): Follow standard conventions

## Recent Changes
- 008-plugin-web-gui: Added Go 1.24.0 (per go.mod)
- 007-cli-config: Added Go 1.24.0 (per go.mod) + urfave/cli/v2 (CLI), BurntSushi/toml (TOML parsing - already indirect dep), slog (logging)
- 006-remove-mcp-tools: Added Go 1.24.0 + mark3labs/mcp-go v0.43.2 (MCP server)


<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
