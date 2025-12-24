# Quickstart: CLI Init Enhancement

**Feature**: 020-cli-init-here
**Date**: 2025-12-23

## Prerequisites

- Go 1.24.0 or later
- ZombieKit repository cloned

## Building

```bash
# Build the CLI with embedded assets
go build -o brains ./cmd/brains
```

## Usage

### Initialize ZombieKit in Current Directory (Default)

```bash
cd /path/to/your/project
brains init
```

**Output**:
```
Created .claude/
Created .claude/commands/
  Copied brains.feature.md
  Copied brains.plan.md
  Copied brains.tasks.md
  ... (all 15 command files)
Created .brains/
Created .brains/templates/
  Copied spec-template.md
  Copied plan-template.md
  ... (all 5 template files)

Initialized ZombieKit: 20 files copied, 0 skipped
```

### Update Existing Installation

```bash
brains init --force
```

**Output**:
```
.claude/ exists
.claude/commands/ exists
  Overwrote brains.feature.md
  Overwrote brains.plan.md
  ... (all files overwritten)

Initialized ZombieKit: 0 files copied, 20 overwritten
```

### Initialize Global Directory Only

```bash
brains init --global
```

**Output**:
```
Initialized ~/.brains/profiles
```

## Verification

After running `brains init`, verify the setup:

```bash
# Check Claude Code commands are available
ls .claude/commands/

# Check templates are available
ls .brains/templates/

# Test a ZombieKit command in Claude Code
# (Open your project in an IDE with Claude Code extension)
# Type: /brains.feature
```

## Common Issues

### Permission Denied

```
Error: creating directory: permission denied
```

**Solution**: Ensure you have write permissions to the current directory.

### Files Already Exist

```
Skipped brains.feature.md (exists)
```

**Solution**: Use `--force` to overwrite existing files:
```bash
brains init --force
```

### Empty Embedded Assets

```
Error: embedded commands filesystem is empty - binary may be corrupted
```

**Solution**: Reinstall or rebuild the brains binary.

## Development Testing

```bash
# Run unit tests
go test ./internal/cli/... -v

# Test in a temporary directory
mkdir /tmp/test-init && cd /tmp/test-init
/path/to/brains init
ls -la .claude/commands/
ls -la .brains/templates/

# Clean up
rm -rf /tmp/test-init
```
