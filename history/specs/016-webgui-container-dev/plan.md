# Implementation Plan: WebGUI Container Development Environment

**Branch**: `016-webgui-container-dev` | **Date**: 2025-12-22 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/016-webgui-container-dev/spec.md`

## Summary

Add a Taskfile entry that starts the WebGUI in a Docker container with wgo for hot-reloading during development. The container exposes port 9981, mounts source code for live changes, and persists SQLite data in `.data/` directory. Uses Docker Compose for orchestration, extending the existing docker-compose.yml.

## Technical Context

**Language/Version**: Go 1.24.0 (per go.mod)
**Primary Dependencies**: Docker Compose, wgo (github.com/bokwoon95/wgo)
**Storage**: SQLite (modernc.org/sqlite) - persisted via volume mount to `.data/`
**Testing**: Manual verification (start/stop/hot-reload/persistence tests)
**Target Platform**: Linux container (development), host OS for Docker
**Project Type**: CLI application with web server
**Performance Goals**: Hot-reload within 5 seconds of file change
**Constraints**: Port 9981 required, SQLite must persist between restarts
**Scale/Scope**: Single developer workflow

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

The constitution file contains template placeholders - no specific gates defined. Proceeding with standard best practices:

- [x] No new libraries unless necessary (wgo is required for file watching)
- [x] Extend existing infrastructure (uses existing docker-compose.yml)
- [x] Simple implementation (Dockerfile + compose service + task entry)

## Project Structure

### Documentation (this feature)

```text
specs/016-webgui-container-dev/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # N/A - no data model changes
├── quickstart.md        # Phase 1 output
├── contracts/           # N/A - no API contracts
└── tasks.md             # Phase 2 output
```

### Source Code (repository root)

```text
# Files to create/modify
Taskfile.yml                    # Add webgui:dev task
docker-compose.yml              # Add webgui-dev service
docker/webgui-dev/Dockerfile    # New Dockerfile for development container
.gitignore                      # Add .data/ if not present
```

**Structure Decision**: Extend existing project infrastructure. New Dockerfile goes in `docker/webgui-dev/` to keep Docker-related files organized.

## Complexity Tracking

No constitution violations - this is a straightforward addition of development tooling.

## Design Details

### Dockerfile (docker/webgui-dev/Dockerfile)

```dockerfile
FROM golang:1.24-alpine

WORKDIR /app

# Install wgo for file watching
RUN go install github.com/bokwoon95/wgo@latest

# Create data directory
RUN mkdir -p /app/.data

# Expose the webgui port
EXPOSE 9981

# Run wgo with file watching for Go, HTML, and CSS files
CMD ["wgo", "run", "-file", ".go", "-file", ".html", "-file", ".css", "./cmd/brains", "gui", "--port", "9981"]
```

### Docker Compose Service

```yaml
webgui-dev:
  build:
    context: .
    dockerfile: docker/webgui-dev/Dockerfile
  container_name: brains-webgui-dev
  ports:
    - "9981:9981"
  volumes:
    - .:/app
    - .data:/app/.data
  environment:
    - BRAINS_DATA_DIR=/app/.data
  working_dir: /app
```

### Taskfile Entry

```yaml
webgui:dev:
  desc: Start WebGUI in development mode with hot-reloading
  cmds:
    - docker compose up --build webgui-dev
```

### Post-Phase 1 Constitution Re-Check

- [x] Minimal new dependencies (only wgo added, required for core functionality)
- [x] Extends existing patterns (same docker-compose.yml, same Taskfile structure)
- [x] No architectural changes to application code
- [x] Simple, focused implementation
