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
	case "PostToolUse":
		return h.handlePostToolUse(event)
	case "SessionEnd":
		return h.handleSessionEnd(event)
	default:
		return HandleResult{}, fmt.Errorf("hook: unrecognized event: %s", event.HookEventName)
	}
}

// handleSessionStart delivers unconditional rules (and graphite status)
// at the start of a session or the equivalent reset boundary. Two distinct
// entry paths route here:
//
//   - Lifecycle events ("startup", "resume", "compact") reset per-session
//     dedup state and re-emit all unconditional rules. Claude Code and
//     Gemini CLI fire these exactly once per reset point.
//   - The OpenCode "inject" source is a stateless per-turn query driven by
//     experimental.chat.system.transform. It does not touch session state
//     and always emits the full unconditional rule set, because the
//     OpenCode system prompt is a fresh array per LLM call — any "inject
//     once, subsequent calls empty" behavior would drop the rules from the
//     prompt after the first turn.
func (h *Handler) handleSessionStart(event *HookEvent) (HandleResult, error) {
	if event.Source == "inject" {
		return h.handleSessionInject(event)
	}

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

// handleSessionInject returns unconditional rules for a per-turn system
// prompt refresh without touching session state. OpenCode's
// experimental.chat.system.transform fires on every assistant turn and
// populates a fresh output.system array each time; emitting the same
// rules on every call is the only way to keep them in the system prompt
// across turns. Stateless-by-design: no dedup, no marking, no save.
func (h *Handler) handleSessionInject(event *HookEvent) (HandleResult, error) {
	unconditional, err := h.rules.ResolveUnconditional()
	if err != nil {
		return HandleResult{}, err
	}

	var bodies []string
	var matched []MatchedRule
	for _, rule := range unconditional {
		matched = append(matched, MatchedRule{ID: rule.ID()})
		bodies = append(bodies, rule.Body)
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
	editor, ok := LookupEditor(h.agent)
	if !ok {
		return HandleResult{}, fmt.Errorf("hook: editor not registered: %s", h.agent)
	}

	if editor.IsShellTool(event.ToolName) {
		return h.handlePreBash(event)
	}

	filePaths := editor.ExtractFilePaths(event)
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

func (h *Handler) handlePostToolUse(event *HookEvent) (HandleResult, error) {
	editor, ok := LookupEditor(h.agent)
	if !ok {
		return HandleResult{}, fmt.Errorf("hook: editor not registered: %s", h.agent)
	}

	// We don't inject rules for failed tool calls
	if event.ToolResponse != nil && event.ToolResponse.Success != nil && !*event.ToolResponse.Success {
		return HandleResult{}, nil
	}

	filePaths := editor.ExtractFilePaths(event)
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
