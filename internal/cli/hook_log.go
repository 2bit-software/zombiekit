package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	"github.com/2bit-software/zombiekit/internal/hook"
)

func newHookLogCommand() *cli.Command {
	return &cli.Command{
		Name:  "log",
		Usage: "Display the hook audit log",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "follow",
				Aliases: []string{"f"},
				Usage:   "Follow the log file as new records are appended",
			},
			&cli.StringFlag{
				Name:  "session",
				Usage: "Filter records to the given session ID",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Emit raw JSON instead of pretty-printed output",
			},
		},
		Action: runHookLog,
	}
}

func runHookLog(c *cli.Context) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving home directory: %w", err)
	}
	path := filepath.Join(homeDir, ".zombiekit", "logs", "hooks.jsonl")

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("no hook log at %s (run a hook event first)", path)
		}
		return fmt.Errorf("opening hook log: %w", err)
	}
	defer func() { _ = f.Close() }()

	session := c.String("session")
	raw := c.Bool("json")
	follow := c.Bool("follow")

	if err := streamRecords(f, os.Stdout, session, raw); err != nil {
		return err
	}
	if !follow {
		return nil
	}
	return tailRecords(f, os.Stdout, session, raw)
}

// streamRecords reads existing records from r and writes filtered output to w.
func streamRecords(r io.Reader, w io.Writer, session string, raw bool) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		if err := emitLine(w, scanner.Bytes(), session, raw); err != nil {
			return err
		}
	}
	return scanner.Err()
}

// tailRecords polls the file for new content after an initial read.
func tailRecords(f *os.File, w io.Writer, session string, raw bool) error {
	reader := bufio.NewReader(f)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			if emitErr := emitLine(w, line, session, raw); emitErr != nil {
				return emitErr
			}
		}
		if err == io.EOF {
			time.Sleep(200 * time.Millisecond)
			continue
		}
		if err != nil {
			return err
		}
	}
}

func emitLine(w io.Writer, line []byte, session string, raw bool) error {
	if len(line) == 0 {
		return nil
	}
	var rec hook.AuditRecord
	if err := json.Unmarshal(line, &rec); err != nil {
		return nil // skip malformed lines silently
	}
	if session != "" && rec.SessionID != session {
		return nil
	}
	if raw {
		_, err := fmt.Fprintln(w, string(line))
		return err
	}
	return printPretty(w, rec)
}

func printPretty(w io.Writer, rec hook.AuditRecord) error {
	ts := rec.Timestamp.Local().Format("15:04:05.000")
	_, err := fmt.Fprintf(w,
		"%s  %-12s  %-15s  session=%s  output=%dB  dur=%dµs  matched=%s  skipped=%s%s\n",
		ts,
		rec.Event,
		formatEditor(rec.Agent, rec.EditorSource),
		shortSession(rec.SessionID),
		rec.OutputBytes,
		rec.DurationMicros,
		formatMatchedRules(rec.MatchedRules),
		formatMatchedRules(rec.SkippedRules),
		errSuffix(rec.Err),
	)
	return err
}

// formatEditor renders the editor ID with its selection source annotation
// (e.g. "gemini(flag)"). Records written before the EditorSource field
// existed just show the agent name.
func formatEditor(agent, source string) string {
	if agent == "" {
		agent = "-"
	}
	if source == "" {
		return agent
	}
	return fmt.Sprintf("%s(%s)", agent, source)
}

// formatMatchedRules renders a slice of MatchedRule entries for the pretty
// log output. Rules with a non-empty trigger render as "id(trigger)" so
// command warnings are distinguishable from file-glob ones.
func formatMatchedRules(entries []hook.MatchedRule) string {
	if len(entries) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Trigger != "" {
			parts = append(parts, fmt.Sprintf("%s(%s)", e.ID, e.Trigger))
			continue
		}
		parts = append(parts, e.ID)
	}
	return "[" + strings.Join(parts, " ") + "]"
}

func shortSession(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func errSuffix(err string) string {
	if err == "" {
		return ""
	}
	return "  err=" + err
}
