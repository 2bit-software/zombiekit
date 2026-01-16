# Research: WebGUI Container Development Environment

**Feature**: 016-webgui-container-dev
**Date**: 2025-12-22

## Research Summary

This feature is straightforward development tooling with minimal unknowns. Research focuses on best practices for wgo configuration and Docker development patterns.

---

## R1: wgo File Watcher Configuration

**Question**: How to configure wgo for optimal Go file watching in a container?

**Decision**: Use `wgo run -file .go -file .html -file .css ./cmd/brains gui --port 9981`

**Rationale**:
- wgo is a simple, single-binary Go file watcher designed for Go development
- Supports multiple `-file` extensions for watching templates and static assets
- Automatically passes through arguments to the underlying command
- No complex configuration needed - works out of the box

**Alternatives Considered**:
- air: More feature-rich but requires config file, heavier dependency
- reflex: Similar to wgo but less Go-centric
- fswatch + script: More complex setup, not Go-aware

---

## R2: Dockerfile Base Image

**Question**: Which base image for Go development container?

**Decision**: Use `golang:1.24-alpine` as base image

**Rationale**:
- Matches project's Go 1.24.0 requirement
- Alpine provides smaller image size (~300MB vs ~800MB for debian-based)
- CGO disabled by default works well with modernc.org/sqlite (pure Go)
- Includes git for go mod operations if needed

**Alternatives Considered**:
- golang:1.24 (debian): Larger but more compatible - not needed here
- golang:1.24-bookworm: Even larger, no benefit for this use case
- Custom multi-stage: Overkill for development container

---

## R3: Source Code Volume Mount Strategy

**Question**: How to mount source code for live reloading?

**Decision**: Bind mount entire project root to `/app` in container

**Rationale**:
- Simple approach that works with wgo
- All source files accessible for watching
- go.mod/go.sum available for dependency resolution
- Matches typical Go development workflow

**Alternatives Considered**:
- Selective mounts (only cmd/, internal/): More complex, risk of missing files
- Docker volumes with sync: Unnecessary complexity for local development

---

## R4: SQLite Path in Container

**Question**: Where should SQLite store data inside container?

**Decision**: Mount `.data/` from host to `/app/.data/` in container; configure app via `BRAINS_DATA_DIR=/app/.data`

**Rationale**:
- Matches clarified spec requirement (`.data/` directory)
- Environment variable allows consistent path inside/outside container
- Simple bind mount ensures persistence
- Path is relative to working directory, aligns with existing code patterns

**Alternatives Considered**:
- Named Docker volume: Would hide data from host, less convenient for debugging
- /data mount: Different path requires more code changes

---

## R5: Container Networking

**Question**: How to expose port for development?

**Decision**: Standard port mapping `9981:9981` in docker-compose.yml

**Rationale**:
- Direct port mapping is simplest approach
- No need for host network mode
- Consistent with existing postgres service pattern (9432:5432)

**Alternatives Considered**:
- Host network mode: Not portable across platforms (doesn't work on macOS)
- Random port: Would break user expectation of fixed 9981

---

## No Further Research Required

All unknowns resolved. Proceeding to Phase 1 design.
