# Research: Embedded Profile Fallback

**Feature**: 019-embedded-profile-fallback
**Date**: 2025-12-23

## Research Summary

This document captures decisions and best practices for implementing embedded profiles using Go's `embed` package.

---

## Decision 1: Embedding Location

**Decision**: Embed the `profiles/` directory from the repository root into the `internal/profile` package.

**Rationale**:
- Go's `//go:embed` directive requires the embed variable to be declared in the same package or a parent package containing the files
- The `internal/profile` package is the natural location since it handles all profile operations
- Embedding at `internal/profile/embedded.go` keeps embed logic close to usage

**Alternatives Considered**:
1. **Embed at `cmd/brains/`**: Rejected - would require passing embed.FS through CLI layers
2. **Create separate `profiles/` package**: Rejected - unnecessary package for a single embed.FS variable
3. **Embed at repository root in `main.go`**: Rejected - couples main to profile internals

**Implementation**:
```go
// In internal/profile/embedded.go
//go:embed profiles/*
var embeddedProfiles embed.FS
```

Note: The embed directive uses a relative path from where the file is declared. Since `internal/profile/embedded.go` is 2 levels deep from the repo root, the path must be `../../profiles/*`. However, Go embed only allows embedding files in the same directory or subdirectories. We must place the embed declaration at the repo root level or copy profiles into the internal/profile directory.

**Revised Decision**: Create the embed.FS at the `cmd/brains/` level (where main.go is) and inject it into the profile package during initialization. This follows the Go embed constraint while keeping separation of concerns.

---

## Decision 2: Integration Pattern with Existing Sources

**Decision**: Extend `BrainsSource.FindProfileDirs()` to append an embedded directory representation as the last item in the returned slice.

**Rationale**:
- Maintains existing precedence order (local > parent > global)
- Embedded profiles become the lowest precedence, added after global
- No changes required to composer, service, or CLI code
- Uses existing `ResolvedDirectory` type with a new `SourceEmbedded` value

**Alternatives Considered**:
1. **Create separate EmbeddedSource implementing ProfileSourceInterface**: Rejected - would require changes to service initialization and CLI
2. **Modify Resolver directly**: Rejected - Resolver handles filesystem directories, embedded is different
3. **Chain at service level**: Rejected - more complex, service shouldn't know about source details

**Implementation Approach**:
- Add `SourceEmbedded ProfileSource = 4` constant to `types.go`
- Update `ProfileSource.String()` to return "embedded" for this value
- Modify `BrainsSource.FindProfileDirs()` to append embedded directory last
- Create helper `loadEmbeddedProfiles()` function to read from embed.FS

---

## Decision 3: embed.FS Access Pattern

**Decision**: Use `fs.ReadDir` and `fs.ReadFile` from the `io/fs` package to read embedded files.

**Rationale**:
- `embed.FS` implements `io/fs.FS` interface
- Works identically to filesystem operations, minimizing code changes
- Allows reusing existing `ParseProfile()` function with embedded content

**Best Practices from Go Documentation**:
1. Always embed with `//go:embed` at package level (not in functions)
2. Use `embed.FS` for multiple files (not `string` or `[]byte` for single file)
3. Access files using `io/fs` functions for consistency
4. Remember: embedded paths don't start with `./` and use forward slashes

---

## Decision 4: Path Representation for Embedded Profiles

**Decision**: Use `[embedded]/<profile-name>.md` as the virtual path for embedded profiles.

**Rationale**:
- Clearly distinguishes embedded from filesystem paths
- Works in JSON output and error messages
- Follows convention of using brackets for synthetic/virtual paths
- The `[embedded]` prefix is searchable and parseable

**Alternatives Considered**:
1. **`embedded://profile.md`**: Rejected - looks like a URL, may confuse users
2. **`<embedded>/profile.md`**: Rejected - angle brackets have shell escaping issues
3. **No path, just name**: Rejected - breaks existing API contract

---

## Decision 5: Embed Initialization

**Decision**: Initialize embedded profiles via a package-level `init()` function or explicit `RegisterEmbeddedProfiles()` call from main.

**Rationale**:
- Go's embed requires compile-time declaration at package level
- The embed variable must be passed from where it's declared to where it's used
- Using explicit registration allows flexibility in testing (can skip embedded for unit tests)

**Implementation**:
```go
// In cmd/brains/main.go
//go:embed profiles/*
var embeddedProfiles embed.FS

func main() {
    profile.SetEmbeddedFS(embeddedProfiles)
    // ... rest of main
}
```

```go
// In internal/profile/embedded.go
var globalEmbeddedFS embed.FS

func SetEmbeddedFS(fs embed.FS) {
    globalEmbeddedFS = fs
}
```

---

## Decision 6: Handling Missing Embedded Profiles Directory

**Decision**: If the embedded filesystem is not initialized or is empty, silently skip embedded profiles without error.

**Rationale**:
- During testing, embedded profiles may not be set
- Maintains backward compatibility
- Users who don't need embedded profiles shouldn't see errors
- Follows existing pattern where missing directories are skipped

---

## Dependencies & Technology Stack

| Component | Version | Usage |
|-----------|---------|-------|
| Go embed package | Go 1.16+ (we use 1.24) | File embedding |
| io/fs package | Go 1.16+ | Filesystem abstraction for reading embedded files |
| stretchr/testify | v1.11.1 | Unit testing |

---

## Open Questions (Resolved)

1. **Q: Where to declare the embed.FS variable?**
   A: At `cmd/brains/main.go` with explicit injection to profile package.

2. **Q: How to handle profiles with includes referencing embedded profiles?**
   A: Works naturally - composer resolves all profiles from the same combined map.

3. **Q: How to test embedded profiles in unit tests?**
   A: Create a test embed.FS or use `SetEmbeddedFS()` with a test filesystem.
