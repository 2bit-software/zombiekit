# Quickstart: ZombieKit Development

## Prerequisites

- Go 1.22 or later
- [Task](https://taskfile.dev/) (go-task)
- Docker (for database services, optional for Tier 1)

## Getting Started

### 1. Clone and Initialize

```bash
git clone <repo-url> zombiekit
cd zombiekit
task init
```

This downloads Go dependencies and installs development tools.

### 2. Build the CLI

```bash
task build
```

Produces `./bin/brains` binary.

### 3. Verify Installation

```bash
./bin/brains --help
./bin/brains version
```

## Development Tasks

### Core Commands

| Command | Description |
|---------|-------------|
| `task` | List available tasks |
| `task init` | Download deps, install tools |
| `task build` | Build the brains binary |
| `task test` | Run tests with coverage |
| `task lint` | Run golangci-lint |
| `task fmt` | Format Go code |
| `task vet` | Run go vet |
| `task ci` | Run all quality checks |

### Database Commands (Tier 2)

| Command | Description |
|---------|-------------|
| `task db:up` | Start PostgreSQL container |
| `task db:down` | Stop PostgreSQL container |
| `task db:migrate` | Run database migrations |

## Project Structure

```
zombiekit/
├── cmd/brains/          # CLI entry point
├── internal/            # Internal packages
│   ├── cli/             # Command implementations
│   ├── config/          # Configuration
│   ├── profile/         # Profile composition
│   ├── spec/            # Spec management
│   ├── conversation/    # Conversation import
│   ├── mcp/             # MCP server
│   └── web/             # Web UI
├── migrations/          # Database schemas
├── profiles/            # Default profiles
├── configs/             # Config files
├── Taskfile.yml         # Build automation
└── docker-compose.yml   # Dev services
```

## Testing

Run all tests:
```bash
task test
```

Run specific package tests:
```bash
go test -v ./internal/profile/...
```

## Adding a New Feature

1. Create test harness in the appropriate package
2. Write failing tests
3. Implement the feature
4. Run `task ci` to verify

## Troubleshooting

### Go version mismatch
```bash
go version  # Check current version
# Install Go 1.22+ if needed
```

### Missing development tools
```bash
task init  # Installs golangci-lint, gotestsum, etc.
```

### Port conflict on database
The database uses port 9432 to avoid conflicts. If still blocked:
```bash
docker ps  # Check running containers
lsof -i :9432  # Check what's using the port
```
