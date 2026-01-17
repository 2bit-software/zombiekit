# Quickstart: MCP Tools

This guide gets the MCP tools running locally for development and testing.

## Prerequisites

- Go 1.22+
- PostgreSQL 16+ (with pgvector extension)
- Docker (for integration tests)

## Setup Database

### Option 1: Docker (Recommended)

```bash
# Start PostgreSQL with docker-compose
docker compose up -d postgres

# Or use standalone container
docker run -d \
  --name brains-postgres \
  -e POSTGRES_USER=brains \
  -e POSTGRES_PASSWORD=brains \
  -e POSTGRES_DB=brains \
  -p 5432:5432 \
  postgres:16-alpine
```

### Option 2: Local PostgreSQL

```bash
# Create database
createdb brains

# Enable extensions (requires superuser)
psql brains -c "CREATE EXTENSION IF NOT EXISTS uuid-ossp;"
psql brains -c "CREATE EXTENSION IF NOT EXISTS pg_trgm;"
```

## Configure Environment

```bash
# Copy example config
cp .env.example .env

# Edit with your database URL
export DATABASE_URL="postgres://brains:brains@localhost:5432/brains?sslmode=disable"
```

## Build and Run

```bash
# Build the CLI
task build

# Run migrations
./bin/brains db migrate

# Check migration status
./bin/brains db status

# Start MCP server (stdio mode for Claude Desktop)
./bin/brains serve --mode stdio

# Start MCP server (HTTP mode for development)
./bin/brains serve --mode http --port 8080
```

## Verify Installation

```bash
# Check version
./bin/brains version

# Test memory operations via CLI
./bin/brains memory set test-key "Hello, World!"
./bin/brains memory get test-key
./bin/brains memory list
./bin/brains memory search "Hello"
./bin/brains memory delete test-key

# Health check (when running HTTP mode)
curl http://localhost:8080/health
```

## Claude Desktop Integration

Add to your Claude Desktop config (`~/Library/Application Support/Claude/claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "brains": {
      "command": "/path/to/brains",
      "args": ["serve", "--mode", "stdio"],
      "env": {
        "DATABASE_URL": "postgres://brains:brains@localhost:5432/brains?sslmode=disable"
      }
    }
  }
}
```

## Running Tests

```bash
# Unit tests (no database required)
task test

# Integration tests (requires Docker)
task test:integration

# All tests with coverage
task test:coverage
```

## CLI Commands Reference

```bash
# MCP Server
brains serve                         # Start with defaults (http, port 8080)
brains serve --mode stdio            # stdio mode for Claude Desktop
brains serve --mode sse              # SSE mode for legacy clients
brains serve --port 3000             # Custom port
brains serve --log-level debug       # Verbose logging

# Memory Management
brains memory list                   # List all memories
brains memory list --format json     # JSON output
brains memory get <name>             # Get a memory
brains memory set <name> "content"   # Set a memory
brains memory delete <name>          # Delete a memory
brains memory search "query"         # Search memories
brains memory clear                  # Clear all memories

# Database
brains db migrate                    # Run pending migrations
brains db status                     # Show migration status

# System
brains version                       # Show version info
brains --help                        # Show all commands
```

## Troubleshooting

### Database Connection Failed

```
Error: database connection failed: connection refused
```

**Fix**: Ensure PostgreSQL is running and DATABASE_URL is correct:
```bash
# Check PostgreSQL is running
pg_isready -h localhost -p 5432

# Test connection
psql $DATABASE_URL -c "SELECT 1;"
```

### Missing Extensions

```
Error: extension "uuid-ossp" does not exist
```

**Fix**: Create extensions (requires superuser):
```bash
psql $DATABASE_URL -c "CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";"
psql $DATABASE_URL -c "CREATE EXTENSION IF NOT EXISTS pg_trgm;"
```

### Port Already in Use

```
Error: listen tcp :8080: address already in use
```

**Fix**: Use a different port:
```bash
./bin/brains serve --port 3000
```

### Migrations Not Applied

```
Error: relation "memories" does not exist
```

**Fix**: Run migrations:
```bash
./bin/brains db migrate
```
