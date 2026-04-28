# ZombieKit

Prompt composition, structured workflows, and artifact management for AI coding agents.

## Quick Start

### Prerequisites

- Go 1.25+ (`go version`)
- [Task](https://taskfile.dev) (`task --version`)
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code), [Gemini CLI](https://github.com/google-gemini/gemini-cli), or [OpenCode](https://opencode.ai)

### Installation

```bash
git clone https://github.com/2bit-software/zombiekit.git
cd zombiekit
task install
```

This builds the `brains` CLI and installs it to your `$GOBIN`.

### MCP Registration

```bash
brains init --claude    # registers the MCP server + slash commands
```

Or add manually to your Claude Code MCP settings:

```json
{
  "mcpServers": {
    "zombiekit": {
      "command": "brains",
      "args": ["serve"]
    }
  }
}
```

### Verify

In Claude Code:

```
/brains.help
```

## How It Works

Claude Code is the brain; ZombieKit is the memory. ZombieKit does not orchestrate AI -- Claude Code does that natively. ZombieKit provides structured workflows, persistent artifact storage, composable prompts (profiles), and contextual rule injection (hooks) that the agent invokes through MCP tools and slash commands.

All artifacts are stored in a `history/` folder in your project, versioned alongside your code.

## Commands

Four slash commands drive all workflows:

| Command | Purpose |
|---------|---------|
| `/brains.new [desc]` | Start new work -- auto-detects feature, bug, or refactor |
| `/brains.next [step]` | Advance to next step, or jump to a named step |
| `/brains.complete` | Finish the current initiative |
| `/brains.help` | Show commands, current state, and valid next actions |

### Workflow Types

`/brains.new` classifies your input and loads the appropriate workflow:

| Type | Trigger | Phases |
|------|---------|--------|
| **feature** | "add X", "implement Y", "create Z" | spec &rarr; plan &rarr; tasks &rarr; implement |
| **feature-light** | "quick feature", "fl: X" | plan &rarr; implement |
| **bug** | "fix X", "broken", "error" | report &rarr; investigate &rarr; fix-plan &rarr; implement |
| **refactor** | "refactor X", "cleanup", "reorganize" | goal &rarr; analysis &rarr; plan &rarr; tasks &rarr; implement |
| **unmanaged** | "unmanaged", "manual" | branch scaffold only -- you handle implementation |

Each phase runs a composable profile. `/brains.next` advances through them. At any point you can detour (`/brains.next audit`, `/brains.next clarify`) without disrupting the main sequence.

## Example

```
brains init --claude                         # one-time setup (terminal)
/brains.new add user authentication          # starts a feature workflow
# ... review and approve the spec ...
/brains.next                                 # advance: spec -> plan
# ... review the plan ...
/brains.next                                 # advance: plan -> tasks
/brains.next                                 # advance: tasks -> implement
/brains.complete                             # wrap up, offer commit/PR
```

## Hooks

ZombieKit injects contextual rules into the agent conversation at the right moments. Rules live in `.brains/rules/` (project-local) and `~/.brains/rules/` (global).

Supports Claude Code, Gemini CLI, and OpenCode. See [INFRASTRUCTURE.md](INFRASTRUCTURE.md#hooks) for the full hook table, editor setup, rule resolution order, and matching semantics.

## Development

For contributors or those wanting the full stack (recall, semantic search, web GUI):

```bash
task dev -- setup           # create .env, check deps, install tools
task dev -- build           # build the binary
task dev -- test            # run tests
task dev -- ci              # run all CI checks (fmt, vet, lint, test, build)
task up                     # start full stack (PostgreSQL, Ollama, GUI)
```

See `task dev -- --list` for all available targets.

## Learn More

- [Architecture and Design](docs/DESIGN.md) -- full architecture, profiles, MCP tools, configuration
- [Infrastructure](INFRASTRUCTURE.md) -- executables, hooks, rule resolution, editor integration

## License

[AGPL-3.0](LICENSE)
