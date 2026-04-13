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

// HandleResult is returned from Handle with the raw rule bodies to inject
// and audit data describing which rules fired (or were deduped) for this
// invocation. Output formatting is the CLI layer's responsibility; the
// handler is editor-agnostic. Each rule entry carries the trigger that
// caused the match — empty for file-glob rules, a command prefix for Bash
// rules.
type HandleResult struct {
	Bodies       []string
	MatchedRules []MatchedRule
	SkippedRules []MatchedRule
}

// NewHandler creates a new Handler.
func NewHandler(workingDir, homeDir string, agent Agent) *Handler {
	return &Handler{
		rules: rules.NewService(workingDir, homeDir),
		agent: agent,
	}
}

// Handle dispatches the hook event and returns the text to write to stdout
// along with the rule IDs that were injected or skipped due to deduplication.
func (h *Handler) Handle(event *HookEvent) (HandleResult, error) {
	switch event.HookEventName {
	case "SessionStart":
		return h.handleSessionStart(event)
	case "PreToolUse":
		return h.handlePreToolUse(event)
	case "SessionEnd":
		return h.handleSessionEnd(event)
	default:
		return HandleResult{}, fmt.Errorf("hook: unrecognized event: %s", event.HookEventName)
	}
}

func (h *Handler) handleSessionStart(event *HookEvent) (HandleResult, error) {
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
		return HandleResult{}, err
	}

	var bodies []string
	var matched []MatchedRule
	for _, rule := range unconditional {
		MarkRuleInjected(state, rule.ID())
		matched = append(matched, MatchedRule{ID: rule.ID()})
		bodies = append(bodies, rule.Body)
	}

	if err := SaveState(state); err != nil {
		return HandleResult{}, err
	}

	if graphiteStatus := DetectGraphiteStatus(event.CWD); graphiteStatus != "" {
		bodies = append(bodies, graphiteStatus)
	}

	return HandleResult{
		Bodies:       bodies,
		MatchedRules: matched,
	}, nil
}

func (h *Handler) handlePreToolUse(event *HookEvent) (HandleResult, error) {
	if event.ToolName == "Bash" {
		return h.handlePreBash(event)
	}

	filePaths := event.ExtractFilePaths()
	if len(filePaths) == 0 {
		return HandleResult{}, nil
	}

	state := LoadState(event.SessionID, h.agent)

	matchedRules, err := h.rules.ResolveForFiles(filePaths)
	if err != nil {
		return HandleResult{}, err
	}

	var bodies []string
	var matched, skipped []MatchedRule
	for _, rule := range matchedRules {
		if IsRuleInjected(state, rule.ID()) {
			skipped = append(skipped, MatchedRule{ID: rule.ID()})
			continue
		}
		MarkRuleInjected(state, rule.ID())
		matched = append(matched, MatchedRule{ID: rule.ID()})
		bodies = append(bodies, rule.Body)
	}

	if err := SaveState(state); err != nil {
		return HandleResult{}, err
	}

	return HandleResult{
		Bodies:       bodies,
		MatchedRules: matched,
		SkippedRules: skipped,
	}, nil
}

// handlePreBash inspects a Bash tool invocation and injects non-blocking
// warnings for every command rule whose trigger matches the invocation
// and whose file-existence gates are satisfied. Each (rule, trigger)
// combination fires at most once per session.
func (h *Handler) handlePreBash(event *HookEvent) (HandleResult, error) {
	if event.ToolInput == nil || event.ToolInput.Command == "" {
		return HandleResult{}, nil
	}

	state := LoadState(event.SessionID, h.agent)

	cwd := event.CWD
	if cwd == "" {
		cwd = "."
	}
	ruleMatches, err := h.rules.ResolveForCommand(event.ToolInput.Command, cwd)
	if err != nil {
		return HandleResult{}, err
	}

	var bodies []string
	var matched, skipped []MatchedRule
	for _, m := range ruleMatches {
		entry := MatchedRule{ID: m.Rule.ID(), Trigger: m.Trigger}
		if IsRuleInjectedFor(state, m.Rule.ID(), m.Trigger) {
			skipped = append(skipped, entry)
			continue
		}
		MarkRuleInjectedFor(state, m.Rule.ID(), m.Trigger)
		matched = append(matched, entry)
		bodies = append(bodies, m.Rule.Body)
	}

	if err := SaveState(state); err != nil {
		return HandleResult{}, err
	}

	return HandleResult{
		Bodies:       bodies,
		MatchedRules: matched,
		SkippedRules: skipped,
	}, nil
}

func (h *Handler) handleSessionEnd(event *HookEvent) (HandleResult, error) {
	return HandleResult{}, DeleteState(event.SessionID)
}
