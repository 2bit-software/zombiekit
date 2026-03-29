# Spike Results: cmux CLI Behavior (v0.63.0)

## Environment

- cmux 0.63.0 (78) [e4aeed8dc]
- macOS, Homebrew install
- Socket: `~/Library/Application Support/cmux/cmux.sock`

## Verified Behaviors

### new-workspace

```
$ cmux new-workspace --name "SPIKE-TEST: spike verification" --cwd /tmp
OK workspace:9
```

- Returns `OK workspace:N` on success (plain text, not JSON)
- `--name` flag accepted but name NOT reflected in `list-workspaces` (shows cwd path instead)
- `--cwd` sets working directory
- `--command` sends text after 500ms shell init delay
- Workspace ref format: `workspace:N` where N is an integer

### rename-workspace

```
$ cmux rename-workspace --workspace workspace:9 "SPIKE-TEST: spike verification"
OK workspace:9
```

- Reliably sets the display name shown in `list-workspaces`
- Required workaround: create workspace, then rename immediately after

### list-workspaces

```
$ cmux list-workspaces
* workspace:5  zombiekit  [selected]
  workspace:4  clawbeam
  workspace:9  SPIKE-TEST: spike verification
  workspace:3  general ai
```

- Plain text only -- no `--json` flag available
- Format: `[*] workspace:N  <name>  [selected]`
- `*` prefix and `[selected]` suffix mark focused workspace
- Leading whitespace (2 spaces) for non-selected, `* ` for selected
- No `--id-format` effect observed

### close-workspace

```
$ cmux close-workspace --workspace workspace:9
OK workspace:9

$ cmux close-workspace --workspace workspace:9
Error: not_found: Workspace not found
(exit code 1)
```

- Success: `OK workspace:N`, exit 0
- Not found: `Error: not_found: Workspace not found`, exit 1

### ping

```
$ cmux ping
PONG
```

- Returns `PONG` on success, exit 0

### identify

```json
{
  "socket_path": "~/Library/Application Support/cmux/cmux.sock",
  "caller": { "workspace_ref": "workspace:5", ... },
  "focused": { "workspace_ref": "workspace:5", ... }
}
```

- Returns JSON with refs, not UUIDs
- `--id-format uuids` has no observable effect

## Key Design Implications

1. **No UUIDs available** -- workspace refs (`workspace:N`) are the only identifiers
2. **No JSON output from list** -- must parse plain text
3. **`--name` unreliable on create** -- use `rename-workspace` as a second step
4. **Refs are ephemeral** -- `workspace:N` is an integer index that may be reused after close
5. **Name matching is the reliable approach** -- list workspaces, grep by name pattern
6. **Error format is parseable** -- `Error: <category>: <message>` on stderr with exit 1
