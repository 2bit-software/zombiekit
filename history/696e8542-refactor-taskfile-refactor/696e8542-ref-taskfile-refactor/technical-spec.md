# Technical Specification: Taskfile Two-File Refactor

## File Structure

```
zombiekit/
├── Taskfile.yml      # User-facing (9 tasks)
└── Taskfile.dev.yml  # Development (12 tasks)
```

## Taskfile.yml (User-Facing)

```yaml
version: '3'

vars:
  VERSION:
    sh: git describe --tags --always --dirty 2>/dev/null || echo "dev"
  COMMIT:
    sh: git rev-parse --short HEAD 2>/dev/null || echo "unknown"
  BUILD_DATE:
    sh: date -u +%Y-%m-%dT%H:%M:%SZ
  VERSION_PKG: github.com/zombiekit/brains/internal/version
  LDFLAGS: -s -w -X {{.VERSION_PKG}}.version={{.VERSION}} -X {{.VERSION_PKG}}.commit={{.COMMIT}} -X {{.VERSION_PKG}}.buildDate={{.BUILD_DATE}}

tasks:
  default:
    desc: List available tasks
    silent: true
    cmds:
      - task --list

  dev:
    desc: Run development tasks. Use like `task dev -- <args>`
    cmds:
      - task --taskfile Taskfile.dev.yml {{.CLI_ARGS}}

  init:
    desc: Download dependencies and install development tools
    cmds:
      - go mod download
      - go mod tidy
      - task: init:golangci-lint

  init:golangci-lint:
    internal: true
    desc: Install golangci-lint if not present
    status:
      - command -v golangci-lint
    cmds:
      - echo "Installing golangci-lint..."
      - go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

  build:
    desc: Build the brains CLI binary
    cmds:
      - mkdir -p bin
      - go build -ldflags "{{.LDFLAGS}}" -o bin/brains ./cmd/brains
    sources:
      - "**/*.go"
      - go.mod
      - go.sum
    generates:
      - bin/brains

  install:
    desc: Build and install brains to GOBIN with bs symlink
    deps: [build]
    vars:
      INSTALL_DIR:
        sh: go env GOBIN || echo "$HOME/go/bin"
    cmds:
      - |
        echo "Installing brains to {{.INSTALL_DIR}}..."
        mkdir -p {{.INSTALL_DIR}}
        if ! cp bin/brains {{.INSTALL_DIR}}/brains; then
          echo "ERROR: Failed to copy brains to {{.INSTALL_DIR}}"
          exit 1
        fi
        cd {{.INSTALL_DIR}} && ln -sf brains bs
        echo ""
        echo "Installed successfully!"
        echo "  brains: {{.INSTALL_DIR}}/brains"
        echo "  bs:     {{.INSTALL_DIR}}/bs -> brains"
        echo ""
        {{.INSTALL_DIR}}/brains version

  test:
    desc: Run tests with coverage
    cmds:
      - task --taskfile Taskfile.dev.yml test

  up:
    desc: Start PostgreSQL database container
    cmds:
      - docker compose up -d postgres
      - echo "Waiting for PostgreSQL to be ready..."
      - |
        for i in $(seq 1 30); do
          if docker compose exec -T postgres pg_isready -U brains -d brains > /dev/null 2>&1; then
            echo "PostgreSQL is ready on port 9432"
            exit 0
          fi
          sleep 1
        done
        echo "Timeout waiting for PostgreSQL"
        exit 1

  down:
    desc: Stop and remove PostgreSQL container
    cmds:
      - docker compose down

  ci:
    desc: Run all CI checks (fmt, vet, lint, test, build)
    cmds:
      - task --taskfile Taskfile.dev.yml ci
```

## Taskfile.dev.yml (Development)

```yaml
version: '3'

tasks:
  default:
    desc: List development tasks
    silent: true
    cmds:
      - task --taskfile Taskfile.dev.yml --list

  test:
    desc: Run tests with coverage
    cmds:
      - go test -v -race -coverprofile=coverage.out ./...
    sources:
      - "**/*.go"
    generates:
      - coverage.out

  ci:
    desc: Run all CI checks (fmt, vet, lint, test, build)
    cmds:
      - task: fmt
      - task: vet
      - task: lint
      - task: test
      - task --taskfile Taskfile.yml build

  fmt:
    desc: Format Go code
    cmds:
      - go fmt ./...

  vet:
    desc: Run go vet
    cmds:
      - go vet ./...

  lint:
    desc: Run golangci-lint
    cmds:
      - golangci-lint run --config configs/.golangci.yml ./...

  db:migrate:
    desc: Run all database migrations
    cmds:
      - task: db:migrate:memory
      - task: db:migrate:recall

  db:migrate:memory:
    desc: Run memory table migration
    cmds:
      - docker compose exec -T postgres psql -U brains -d brains -f /dev/stdin < internal/database/migrations/postgres/001_memory_items.sql

  db:migrate:recall:
    desc: Run recall chunks migration (pgvector)
    cmds:
      - docker compose exec -T postgres psql -U brains -d brains -f /dev/stdin < internal/database/migrations/postgres/002_recall_chunks.sql

  ollama:pull:
    desc: Pull required embedding model for recall
    cmds:
      - ollama pull nomic-embed-text
      - echo "Model nomic-embed-text ready for recall embeddings"

  recall:demo:
    desc: Demo the recall feature (requires db:up and ollama)
    env:
      BRAINS_BACKEND: postgres
      BRAINS_POSTGRES_URL: postgres://brains:brains_dev@localhost:9432/brains
    cmds:
      - ./bin/brains recall save "The deployment failed due to memory limits"
      - ./bin/brains recall save "CSS styling updated for login page"
      - ./bin/brains recall save "Database connection pool exhausted"
      - echo ""
      - echo "=== List all chunks ==="
      - ./bin/brains recall list
      - echo ""
      - echo "=== Search for memory-related issues ==="
      - ./bin/brains recall search "memory problems"

  webgui:dev:
    desc: Start WebGUI in development mode with hot-reloading
    cmds:
      - docker compose up --build webgui-dev
```

## Traceability Matrix

| Requirement | Implementation |
|-------------|----------------|
| FR-1.1 | `Taskfile.yml` created with 9 user-facing tasks |
| FR-1.2 | `Taskfile.dev.yml` created with 12 dev tasks |
| FR-1.3 | `dev` task uses `--taskfile` flag |
| FR-2 | Task distribution matches spec tables |
| FR-3.1 | `init:golangci-lint` uses `status:` pattern |
| FR-3.2 | `command -v golangci-lint` check |
| FR-4.1 | `default` has `silent: true` |
| NFR-1 | `db:up`/`db:down` removed, `up`/`down` added |
| NFR-2 | All tasks have `desc` field |
| NFR-3 | Variables in main file; dev `ci` calls main `build` |

## Notes

1. **Variable scope**: `LDFLAGS` stays in main file. Dev file's `ci` task calls `task --taskfile Taskfile.yml build` to access it.

2. **Init subtask visibility**: `init:golangci-lint` appears in task list but is an internal detail. This is acceptable per Taskfile conventions.

3. **Dev file self-reference**: Dev file's `default` task uses `task --taskfile Taskfile.dev.yml --list` for accurate output when invoked via main file's `dev` task.
