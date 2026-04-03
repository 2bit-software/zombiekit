package hook

import (
	"fmt"

	"github.com/2bit-software/zombiekit/internal/rules"
)

// Handler processes hook events and returns rules content for injection.
type Handler struct {
	rules *rules.Service
	agent Agent
}

// NewHandler creates a new Handler.
func NewHandler(workingDir, homeDir string, agent Agent) *Handler {
	return &Handler{
		rules: rules.NewService(workingDir, homeDir),
		agent: agent,
	}
}

// Handle dispatches the hook event and returns text to write to stdout.
// Returns empty string when no rules need injection.
func (h *Handler) Handle(event *HookEvent) (string, error) {
	switch event.HookEventName {
	case "SessionStart":
		return h.handleSessionStart(event)
	case "PreToolUse":
		return h.handlePreToolUse(event)
	case "SessionEnd":
		return h.handleSessionEnd(event)
	default:
		return "", fmt.Errorf("unknown hook event: %s", event.HookEventName)
	}
}

func (h *Handler) handleSessionStart(event *HookEvent) (string, error) {
	state := LoadState(event.SessionID, h.agent)

	// Reset tracking for all sources (startup, resume, compact)
	ResetInjectedRules(state)

	// For compact events, the compaction count was already incremented by Reset.
	// For startup/resume, decrement it back since this isn't a compaction.
	if event.Source != "compact" {
		state.CompactionCount--
		if state.CompactionCount < 0 {
			state.CompactionCount = 0
		}
	}

	unconditional, err := h.rules.ResolveUnconditional()
	if err != nil {
		return "", err
	}

	var bodies []string
	for _, rule := range unconditional {
		MarkRuleInjected(state, rule.ID())
		bodies = append(bodies, rule.Body)
	}

	if err := SaveState(state); err != nil {
		return "", err
	}

	if graphiteStatus := DetectGraphiteStatus(event.CWD); graphiteStatus != "" {
		bodies = append(bodies, graphiteStatus)
	}

	return FormatOutput(h.agent, bodies), nil
}

func (h *Handler) handlePreToolUse(event *HookEvent) (string, error) {
	filePaths := event.ExtractFilePaths()
	if len(filePaths) == 0 {
		return "", nil
	}

	state := LoadState(event.SessionID, h.agent)

	matched, err := h.rules.ResolveForFiles(filePaths)
	if err != nil {
		return "", err
	}

	var bodies []string
	for _, rule := range matched {
		if IsRuleInjected(state, rule.ID()) {
			continue
		}
		MarkRuleInjected(state, rule.ID())
		bodies = append(bodies, rule.Body)
	}

	if err := SaveState(state); err != nil {
		return "", err
	}

	return FormatPreToolOutput(h.agent, bodies), nil
}

func (h *Handler) handleSessionEnd(event *HookEvent) (string, error) {
	return "", DeleteState(event.SessionID)
}
