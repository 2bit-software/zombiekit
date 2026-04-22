// OpenCode plugin shim bridging tool events to the zombiekit `brains`
// binary. Copy this file into your project at
// `.opencode/plugins/brains.ts`, then start OpenCode. The shim spawns
// `brains hook --editor opencode` for three events and merges any rule
// text `brains` returns into OpenCode's mutable hook output.
//
// Binary path: set `BRAINS_BIN` in OpenCode's environment to target a
// non-default binary (e.g. `BRAINS_BIN=brains-test` for development).
//
// Stability warning: `experimental.chat.system.transform` and
// `experimental.session.compacting` are OpenCode experimental hooks and
// may rename in future releases.

import type { Plugin } from "@opencode-ai/plugin"

const BRAINS_BIN = process.env.BRAINS_BIN ?? "brains"

let loggedStartup = false

type Envelope = {
  hookSpecificOutput?: { additionalContext?: string }
}

async function callBrains(
  event: string,
  payload: Record<string, unknown>,
): Promise<string> {
  if (!loggedStartup) {
    console.error(`[brains/opencode] plugin active, binary=${BRAINS_BIN}`)
    loggedStartup = true
  }
  try {
    const proc = Bun.spawn(
      [BRAINS_BIN, "hook", "--editor", "opencode", "--event", event],
      { stdin: "pipe", stdout: "pipe", stderr: "inherit" },
    )
    proc.stdin.write(JSON.stringify(payload))
    proc.stdin.end()
    const out = await new Response(proc.stdout).text()
    const exit = await proc.exited
    if (exit !== 0 || !out.trim()) return ""
    const env = JSON.parse(out) as Envelope
    return env.hookSpecificOutput?.additionalContext ?? ""
  } catch (err) {
    console.error(`[brains/opencode] ${event} failed:`, err)
    return ""
  }
}

// TODO(opencode): multi-edit passes an `edits` array with per-entry file
// paths. This helper only extracts the top-level filePath, so multi-edit
// events only match the first file. Extend to iterate edits[].filePath and
// call brains once per unique path (or batch them in a single payload).
function extractFilePath(tool: string, args: unknown): string | undefined {
  if (tool !== "write" && tool !== "edit" && tool !== "multi-edit") {
    return undefined
  }
  const a = args as { filePath?: string; file_path?: string } | undefined
  return a?.filePath ?? a?.file_path
}

export const server: Plugin = async ({ directory }) => ({
  // Fires on every assistant turn. `brains` dedups per session so only
  // the first call of a given sessionID returns rules; subsequent calls
  // are a no-op. Append-only on output.system — never touch index 0,
  // which OpenCode preserves byte-identical for upstream prompt caching.
  "experimental.chat.system.transform": async (input, output) => {
    const ctx = await callBrains("session-inject", {
      session_id: input.sessionID,
      hook_event_name: "SessionStart",
      cwd: directory,
    })
    if (ctx) output.system.push(ctx)
  },

  // Fires during conversation compaction. `brains` resets the session
  // dedup state AND returns the unconditional rule bodies so the
  // compacted conversation is born with rules already present.
  "experimental.session.compacting": async (input, output) => {
    const ctx = await callBrains("compact", {
      session_id: input.sessionID,
      hook_event_name: "SessionStart",
      cwd: directory,
    })
    if (ctx) output.context.push(ctx)
  },

  // TODO(opencode): tool.execute.before (PreToolUse) is not wired in this
  // iteration. Bash-command rules and pre-edit rules only fire on the
  // "after" edge. Wire tool.execute.before once OpenCode stabilizes the
  // hook signature and we confirm the output shape supports injection.

  // Fires after a tool call completes. For file-editing tools, forward
  // the file path to `brains`, which matches it against rule globs and
  // returns any rule bodies the model should see as part of the tool
  // result on its next turn.
  "tool.execute.after": async (input, output) => {
    const filePath = extractFilePath(input.tool, (output as { args?: unknown }).args)
    if (!filePath) return
    const ctx = await callBrains("post-tool-use", {
      session_id: input.sessionID,
      hook_event_name: "PostToolUse",
      cwd: directory,
      tool_name: input.tool,
      tool_input: { file_path: filePath },
    })
    if (ctx) {
      const existing = (output as { output?: string }).output ?? ""
      ;(output as { output: string }).output = existing ? `${existing}\n\n${ctx}` : ctx
    }
  },
})
