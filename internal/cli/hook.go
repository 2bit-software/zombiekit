package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/2bit-software/zombiekit/internal/hook"
)

func newHookCommand() *cli.Command {
	return &cli.Command{
		Name:  "hook",
		Usage: "Handle AI agent hook events for rules injection",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "event",
				Usage: "Hook event type: session-start, session-inject, pre-tool-use, post-tool-use, session-end, compact",
			},
			&cli.StringFlag{
				Name:  "editor",
				Usage: "Target coding editor: claude, gemini, opencode (default: auto-detect via env, fallback claude)",
			},
		},
		Action: runHook,
		Subcommands: []*cli.Command{
			newHookLogCommand(),
		},
	}
}

func runHook(c *cli.Context) error {
	eventType := c.String("event")
	if eventType == "" {
		return fmt.Errorf("--event is required")
	}

	editor, editorSource, err := hook.ResolveEditor(c.String("editor"))
	if err != nil {
		return err
	}

	var event hook.HookEvent
	raw, _ := io.ReadAll(os.Stdin)
	f, _ := os.OpenFile("/tmp/zk-hook-debug.jsonl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	_, _ = f.Write(append(raw, '\n'))
	_ = f.Close()
	if err := json.Unmarshal(raw, &event); err != nil {
		return fmt.Errorf("reading hook event from stdin: %w", err)
	}

	// The --event flag is the canonical event identifier regardless of what
	// the editor wrote into hook_event_name. This lets editors with
	// different lifecycle vocabularies (Gemini's BeforeTool, Claude's
	// PreToolUse) share a single handler switch.
	switch eventType {
	case "session-start":
		event.HookEventName = "SessionStart"
	case "session-inject":
		event.HookEventName = "SessionStart"
		event.Source = "inject"
	case "pre-tool-use":
		event.HookEventName = "PreToolUse"
	case "post-tool-use":
		event.HookEventName = "PostToolUse"
	case "session-end":
		event.HookEventName = "SessionEnd"
	case "compact":
		event.HookEventName = "SessionStart"
		event.Source = "compact"
	default:
		return fmt.Errorf("unknown event type: %s", eventType)
	}

	homeDir, _ := os.UserHomeDir()
	handler := hook.NewHandler(event.CWD, homeDir, editor)

	sink := newHookAuditSink(homeDir)

	start := time.Now()
	result, handleErr := handler.Handle(&event)

	output := formatHookOutput(editor, event.HookEventName, result.Bodies)

	var command string
	if event.ToolInput != nil {
		command = event.ToolInput.Command
	}

	var filePaths []string
	if ed, ok := hook.LookupEditor(editor); ok {
		filePaths = ed.ExtractFilePaths(&event)
	}

	_ = sink.Write(hook.AuditRecord{
		Timestamp:      start.UTC(),
		Event:          event.HookEventName,
		SessionID:      event.SessionID,
		Agent:          string(editor),
		EditorSource:   string(editorSource),
		CWD:            event.CWD,
		Source:         event.Source,
		ToolName:       event.ToolName,
		Command:        command,
		FilePaths:      filePaths,
		MatchedRules:   result.MatchedRules,
		SkippedRules:   result.SkippedRules,
		OutputBytes:    len(output),
		DurationMicros: time.Since(start).Microseconds(),
		Err:            errString(handleErr),
	})

	if handleErr != nil {
		return handleErr
	}

	if output != "" {
		fmt.Print(output)
	}

	return nil
}

// formatHookOutput dispatches to the editor's formatter for the given event.
// Unregistered editors return empty output — ResolveEditor has already
// validated the editor ID when it came from the --editor flag, so this path
// is only reached for env/default editors which are always registered.
func formatHookOutput(editor hook.Agent, eventName string, bodies []string) string {
	formatter, ok := hook.LookupEditor(editor)
	if !ok {
		return ""
	}
	switch eventName {
	case "SessionStart":
		return formatter.FormatSessionStart(bodies)
	case "PreToolUse":
		return formatter.FormatPreToolUse(bodies)
	case "PostToolUse":
		return formatter.FormatPostToolUse(bodies)
	case "SessionEnd":
		return formatter.FormatSessionEnd(bodies)
	}
	return ""
}

// newHookAuditSink returns a FileSink unless ZK_HOOK_LOG=0 disables auditing.
func newHookAuditSink(homeDir string) hook.AuditSink {
	if os.Getenv("ZK_HOOK_LOG") == "0" {
		return hook.NopSink{}
	}
	return hook.NewFileSink(homeDir)
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
