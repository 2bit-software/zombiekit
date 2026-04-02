package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/2bit-software/zombiekit/internal/hook"
)

func newHookCommand() *cli.Command {
	return &cli.Command{
		Name:  "hook",
		Usage: "Handle AI agent hook events for rules injection",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "event",
				Usage:    "Hook event type: session-start, pre-tool-use, session-end",
				Required: true,
			},
		},
		Action: runHook,
	}
}

func runHook(c *cli.Context) error {
	eventType := c.String("event")

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

	output, err := handler.Handle(&event)
	if err != nil {
		return err
	}

	if output != "" {
		fmt.Print(output)
	}

	return nil
}
