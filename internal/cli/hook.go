package cli

import (
	"encoding/json"
	"fmt"
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
				Usage: "Hook event type: session-start, pre-tool-use, session-end",
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

	var event hook.HookEvent
	if err := json.NewDecoder(os.Stdin).Decode(&event); err != nil {
		return fmt.Errorf("reading hook event from stdin: %w", err)
	}

	// Map CLI event flag to protocol event name if needed
	if event.HookEventName == "" {
		switch eventType {
		case "session-start":
			event.HookEventName = "SessionStart"
		case "pre-tool-use":
			event.HookEventName = "PreToolUse"
		case "session-end":
			event.HookEventName = "SessionEnd"
		default:
			return fmt.Errorf("unknown event type: %s", eventType)
		}
	}

	agent := hook.DetectAgent()
	homeDir, _ := os.UserHomeDir()
	handler := hook.NewHandler(event.CWD, homeDir, agent)

	sink := newHookAuditSink(homeDir)

	start := time.Now()
	result, handleErr := handler.Handle(&event)

	var command string
	if event.ToolInput != nil {
		command = event.ToolInput.Command
	}

	_ = sink.Write(hook.AuditRecord{
		Timestamp:      start.UTC(),
		Event:          event.HookEventName,
		SessionID:      event.SessionID,
		Agent:          string(agent),
		CWD:            event.CWD,
		Source:         event.Source,
		ToolName:       event.ToolName,
		Command:        command,
		FilePaths:      event.ExtractFilePaths(),
		MatchedRules:   result.MatchedRules,
		SkippedRules:   result.SkippedRules,
		OutputBytes:    len(result.Output),
		DurationMicros: time.Since(start).Microseconds(),
		Err:            errString(handleErr),
	})

	if handleErr != nil {
		return handleErr
	}

	if result.Output != "" {
		fmt.Print(result.Output)
	}

	return nil
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
