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
- Go 1.24.0 (per go.mod) + go-chi/chi/v5 (routing), html/template (rendering), mark3labs/mcp-go (MCP), marked.js (CDN - client-side markdown) (009-sticky-memory-plugin)
- Reuses existing `internal/memory` package (SQLite default, PostgreSQL optional) (009-sticky-memory-plugin)
- Go 1.24.0 (per go.mod) + None new required - interface-only feature (010-searchable-interface)
- N/A (interface contract only; implementations provide storage) (010-searchable-interface)
- Go 1.24.0 (per go.mod) + go-chi/chi/v5 (routing), html/template (rendering), HTMX 1.9.10 (client-side), Tailwind CSS (styling via CDN) (011-webgui-search)
- N/A (uses existing memory plugin storage; search is read-only) (011-webgui-search)
- Go 1.24.0 (per go.mod) + go-chi/chi/v5 (routing), mark3labs/mcp-go (MCP tools) (012-plugin-registration-api)
- N/A (no storage changes - this is an API/interface change) (012-plugin-registration-api)
- Go 1.24.0 + urfave/cli/v2 (CLI), modernc.org/sqlite (SQLite), jackc/pgx/v5 (PostgreSQL) (013-sqlite-postgres-import)
- SQLite (source, read-only), PostgreSQL (target, read-write with new import_metadata table) (013-sqlite-postgres-import)
- Go 1.24.0 (per go.mod) + go-chi/chi/v5 (routing), html/template (rendering), internal/version (build info), internal/config (storage config) (014-webgui-status)
- SQLite (default) or PostgreSQL - read-only status display (014-webgui-status)
- Go 1.24.0 (per go.mod) + BurntSushi/toml (config parsing), jackc/pgx/v5 (PostgreSQL), modernc.org/sqlite (SQLite), urfave/cli/v2 (CLI) (015-postgres-config)
- PostgreSQL (primary when configured) or SQLite (default/fallback) (015-postgres-config)
- Go 1.24.0 (per go.mod) + Docker Compose, wgo (github.com/bokwoon95/wgo) (016-webgui-container-dev)
- SQLite (modernc.org/sqlite) - persisted via volume mount to `.data/` (016-webgui-container-dev)
- Go 1.24.0 (per go.mod) + mark3labs/mcp-go (MCP server framework) (017-zombiekit-mcp)
- File system read-only (no database) (017-zombiekit-mcp)

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
- 017-zombiekit-mcp: Added Go 1.24.0 (per go.mod) + mark3labs/mcp-go (MCP server framework)
- 016-webgui-container-dev: Added Go 1.24.0 (per go.mod) + Docker Compose, wgo (github.com/bokwoon95/wgo)
- 015-postgres-config: Added Go 1.24.0 (per go.mod) + BurntSushi/toml (config parsing), jackc/pgx/v5 (PostgreSQL), modernc.org/sqlite (SQLite), urfave/cli/v2 (CLI)


<!-- MANUAL ADDITIONS START -->
<!-- MANUAL ADDITIONS END -->
