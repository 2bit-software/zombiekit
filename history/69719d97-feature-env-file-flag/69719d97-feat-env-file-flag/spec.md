# Feature Specification: --env-file Flag for MCP Serve Command

**Feature Branch**: `69719d97-feature-env-file-flag`
**Created**: 2026-01-21
**Status**: Draft
**Input**: User description: "I want the MCP tool to support an --env-file argument, which points to an environment file used for loading. This way, we can use the same .env for the postgres/etc, and *also* use it for the various mcp binaries running in stdio mode inside their respective shells."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Load Environment from File for MCP Stdio Mode (Priority: P1)

As a developer using Claude Code with the brains MCP server, I want to specify `--env-file .env` in my MCP configuration so that the server automatically loads PostgreSQL credentials and other settings without requiring manual environment setup in the shell.

**Why this priority**: This is the primary use case that motivated the feature. MCP clients like Claude Code spawn processes in isolated shells where environment variables aren't inherited from the user's shell or direnv setup.

**Independent Test**: Can be fully tested by running `brains serve --env-file .env --mode stdio` and verifying PostgreSQL connection succeeds using credentials from the file.

**Acceptance Scenarios**:

1. **Given** a valid `.env` file with `BRAINS_POSTGRES_URL` and `BRAINS_BACKEND=postgres`, **When** user runs `brains serve --env-file .env --mode stdio`, **Then** the server connects to PostgreSQL using the URL from the file
2. **Given** no existing `BRAINS_BACKEND` in environment, **When** user runs `brains serve --env-file .env`, **Then** the value from `.env` is used
3. **Given** `BRAINS_LOG_LEVEL=debug` already in environment and `BRAINS_LOG_LEVEL=info` in .env file, **When** user runs `brains serve --env-file .env`, **Then** the existing environment value `debug` takes precedence

---

### User Story 2 - Error Handling for Missing File (Priority: P2)

As a developer, I want clear error messages when the specified env file doesn't exist so that I can fix configuration issues quickly.

**Why this priority**: Essential for usability - without clear errors, users will be confused when configuration fails silently.

**Independent Test**: Can be tested by running `brains serve --env-file nonexistent.env` and verifying an error message is shown.

**Acceptance Scenarios**:

1. **Given** a path to a file that doesn't exist, **When** user runs `brains serve --env-file /path/to/missing.env`, **Then** the command exits with error "env file not found: /path/to/missing.env"
2. **Given** a path to a directory (not a file), **When** user runs `brains serve --env-file /some/directory`, **Then** the command exits with error indicating it's not a valid file

---

### User Story 3 - Relative and Absolute Path Support (Priority: P2)

As a developer, I want to use both relative and absolute paths for the env file so that configuration works regardless of how I reference the file.

**Why this priority**: Relative paths are more portable for MCP configurations shared across machines.

**Independent Test**: Can be tested by running with both `--env-file .env` and `--env-file /full/path/to/.env`.

**Acceptance Scenarios**:

1. **Given** an env file at `./.env`, **When** user runs `brains serve --env-file .env`, **Then** the file is loaded relative to current working directory
2. **Given** an env file at `/Users/dev/project/.env`, **When** user runs `brains serve --env-file /Users/dev/project/.env`, **Then** the file is loaded from the absolute path

---

### Edge Cases

- What happens when the env file is empty? → Server starts normally with no additional variables
- What happens when the env file has invalid syntax? → Server exits with parsing error indicating line number
- What happens when the env file has a BOM? → File should be parsed correctly (UTF-8 BOM stripped)
- What happens with comments and blank lines? → Lines starting with `#` are ignored, blank lines are skipped

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept `--env-file <path>` flag on the `serve` command
- **FR-002**: System MUST load and parse the specified file using standard dotenv format before other configuration processing
- **FR-003**: System MUST apply precedence: file values < existing environment < CLI flags
- **FR-004**: System MUST exit with clear error if specified file doesn't exist or is unreadable
- **FR-005**: System MUST support both relative and absolute paths
- **FR-006**: System MUST ignore comments (lines starting with `#`) and blank lines
- **FR-007**: System MUST support quoted values (both single and double quotes)

### Key Entities

- **Environment File**: A text file in dotenv format containing KEY=value pairs, one per line

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: MCP server successfully connects to PostgreSQL when credentials are only in the env file
- **SC-002**: Existing environment variables are not overwritten by file values
- **SC-003**: CLI flags override both file and environment values
- **SC-004**: Clear, actionable error message for missing or invalid files

## Testing Requirements *(mandatory)*

### Test Strategy

- Integration tests for the flag parsing and file loading
- Unit tests for the dotenv parsing logic (if implementing custom parser)
- Manual smoke test with actual MCP client (Claude Code)

### FR to Test Mapping

| FR | Test Type | Description |
|----|-----------|-------------|
| FR-001 | Integration | Verify flag is accepted and parsed |
| FR-002 | Integration | Verify file is loaded before config.LoadStorageConfig() |
| FR-003 | Integration | Verify precedence: file < env < CLI |
| FR-004 | Integration | Verify error on missing file |
| FR-005 | Integration | Test with relative and absolute paths |
| FR-006 | Unit | Verify comments and blanks ignored |
| FR-007 | Unit | Verify quoted values parsed correctly |

### Edge Case Coverage

- Empty file → Integration test verifying no error
- Invalid syntax → Integration test verifying error with line number
- Permission denied → Integration test verifying clear error message

## Technical Notes

- Consider using `github.com/joho/godotenv` for parsing (widely used, MIT licensed, handles edge cases)
- Load env file early in `runServe()` before `config.LoadStorageConfig()` is called
- Use `godotenv.Load()` which sets env vars without overwriting existing ones (correct precedence by default)
- Consider adding to `gui` and `start` commands for consistency (out of scope for P1)
