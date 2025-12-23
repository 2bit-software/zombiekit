# Feature Specification: Embedded Profile Fallback

**Feature Branch**: `019-embedded-profile-fallback`
**Created**: 2025-12-23
**Status**: Draft
**Input**: User description: "we need the profiles/compose composing endpoint, to use the local ./profiles folder as a fallback. This folder should get embedded in the brains CLI, and as a very last location, should try to load the profiles from that location."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Default Profiles Always Available (Priority: P1)

As a user running the brains CLI without any local or global profile configuration, I want default profiles to be automatically available so I can start using the CLI immediately without setup.

**Why this priority**: This is the core value proposition - users should have a working CLI experience out of the box. Without this, new users face a barrier to entry where they must first create profiles before they can use any profile-based features.

**Independent Test**: Can be fully tested by running `brains profile compose init` on a fresh machine with no `.brains/profiles/` directories, and the embedded `init.md` profile should be returned successfully.

**Acceptance Scenarios**:

1. **Given** a fresh installation with no `.brains/profiles/` directories anywhere, **When** a user runs `brains profile list`, **Then** they see the embedded default profiles listed with source marked as "embedded".

2. **Given** a fresh installation with no local profiles, **When** a user runs `brains profile compose research`, **Then** they receive the composed content from the embedded `research.md` profile.

3. **Given** a fresh installation, **When** a user runs `brains profile show init`, **Then** they see the content of the embedded `init.md` profile with its path shown as "[embedded]".

---

### User Story 2 - Local Profiles Override Embedded (Priority: P2)

As a user who has customized a profile locally, I want my local version to take precedence over the embedded default so I can tailor profiles to my specific needs without modifying the CLI binary.

**Why this priority**: This preserves the existing shadowing behavior and ensures users can customize their experience. The precedence order (local > parent > global > embedded) maintains backward compatibility.

**Independent Test**: Can be fully tested by creating a local `.brains/profiles/init.md` and running `brains profile compose init`, which should return the local version, not the embedded one.

**Acceptance Scenarios**:

1. **Given** a local profile `init.md` exists at `.brains/profiles/init.md`, **When** a user runs `brains profile compose init`, **Then** the local version is used and the embedded version is shadowed.

2. **Given** a global profile `research.md` exists at `~/.brains/profiles/research.md`, **When** a user runs `brains profile compose research` and no local version exists, **Then** the global version is used and the embedded version is shadowed.

3. **Given** both local and embedded versions of `plan.md` exist, **When** a user runs `brains profile list`, **Then** the local version is shown as primary and the embedded version is marked as shadowed.

---

### User Story 3 - MCP Tools Use Embedded Fallback (Priority: P2)

As a developer using brains as an MCP server, I want the profile-compose and profile-list tools to include embedded profiles so Claude Code users have default profiles available without additional setup.

**Why this priority**: MCP integration is a primary use case for brains, and the MCP tools should behave consistently with the CLI commands.

**Independent Test**: Can be fully tested by calling the `profile-compose` MCP tool with a default profile name on a machine with no profile directories, which should return the embedded profile content.

**Acceptance Scenarios**:

1. **Given** no profile directories exist, **When** the MCP `profile-compose` tool is called with `["init"]`, **Then** it returns the embedded `init.md` content successfully.

2. **Given** no profile directories exist, **When** the MCP `profile-list` tool is called, **Then** it returns the list of embedded profiles with source marked as "embedded".

---

### User Story 4 - Profiles Embedded at Build Time (Priority: P3)

As a developer building and distributing the brains CLI, I want the `./profiles/` directory to be embedded into the binary at compile time so the CLI is self-contained and the embedded profiles are versioned with the source code.

**Why this priority**: This ensures embedded profiles are always in sync with the CLI version and simplifies distribution - users don't need to install profiles separately.

**Independent Test**: Can be fully tested by building the brains binary, moving it to a new directory, and running `brains profile list`, which should show embedded profiles.

**Acceptance Scenarios**:

1. **Given** the `./profiles/` directory contains profile files, **When** the brains binary is built with `go build`, **Then** the profile files are embedded in the resulting binary.

2. **Given** an embedded profile `feature.md` exists, **When** a user inspects the binary (or runs `brains profile show feature`), **Then** the profile content matches the source `./profiles/feature.md` at build time.

---

### Edge Cases

- What happens when an embedded profile has a parse error? The system should skip that profile and log a warning, consistent with how filesystem profile parse errors are handled.
- What happens when the embedded profiles directory is empty? The system functions normally, just without any embedded fallbacks - existing local/global profiles work as before.
- What happens when a user's profile includes an embedded profile? The inclusion should work - embedded profiles should be resolvable by name just like any other profile.
- What happens when inherits:true is set on a local profile with an embedded base? The inheritance chain should include the embedded version as the base, enabling users to extend default profiles.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST embed all `.md` files from the `./profiles/` directory into the brains binary at build time using Go's embed directive
- **FR-002**: System MUST check embedded profiles as the last fallback source after local, parent, and global directories (precedence: local > parent > global > embedded)
- **FR-003**: System MUST allow local, parent, or global profiles to shadow (override) embedded profiles with the same name
- **FR-004**: System MUST display embedded profiles in `profile list` output with source identified as "embedded"
- **FR-005**: System MUST return embedded profile content via `profile compose` when no higher-precedence version exists
- **FR-006**: System MUST support profile inheritance (`inherits: true`) where embedded profiles serve as base versions
- **FR-007**: System MUST support profile inclusion (`includes: [name]`) referencing embedded profiles by name
- **FR-008**: System MUST handle embedded profile parse errors gracefully by skipping the invalid profile and continuing
- **FR-009**: MCP tools (`profile-compose`, `profile-list`) MUST have access to embedded profiles as fallback
- **FR-010**: System MUST show embedded profile paths as "[embedded]" or similar identifier in show/list output to distinguish them from filesystem paths

### Key Entities

- **Embedded Profile**: A profile file bundled into the binary at compile time from `./profiles/*.md`. Immutable at runtime, serves as default/fallback.
- **Profile Source**: The origin of a profile. Extended to include "embedded" in addition to existing "local", "parent", and "global" sources.
- **Resolution Order**: The priority sequence for profile lookup: local (highest) > parent directories > global > embedded (lowest).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can run profile commands immediately after installing brains without any prior configuration
- **SC-002**: All 15+ default profiles from `./profiles/` are accessible via CLI and MCP on fresh installations
- **SC-003**: Profile composition operations complete successfully when only embedded profiles are available
- **SC-004**: Users can override any embedded profile by creating a same-named profile locally or globally
- **SC-005**: Profile inheritance chains resolve correctly when base profiles are embedded
- **SC-006**: Profile validation passes when embedded profiles reference each other correctly

## Assumptions

- The `./profiles/` directory in the repository root contains the default profiles to be embedded
- Go's `//go:embed` directive will be used for embedding (standard Go approach)
- The embedded profiles are read-only at runtime (users cannot modify them)
- Embedded profile source will be represented as `SourceEmbedded` in the `ProfileSource` enum
- The resolver's `FindProfileDirs()` will be extended to include an embedded directory representation
- Build process does not need modification - `go build` handles embedding automatically
