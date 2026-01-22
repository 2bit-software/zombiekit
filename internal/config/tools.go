package config

import (
	"strings"

	"github.com/zombiekit/brains/internal/logging"
)

// KnownTools is the list of known tool names for validation.
var KnownTools = []string{
	"stickymemory",
	"code-reasoning",
	"profile-compose",
	"profile-list",
	"profile-show",
	"profile-validate",
	"profile-save",
	"feature",
	"step",
	"initiative",
	"workflow-compose",
	"recall-list-conversations",
	"recall-read-conversation",
}

// ToolCategory derives the category name from a tool name.
// For hyphenated tool names, returns the prefix before the first hyphen.
// For non-hyphenated names, returns the full tool name.
//
// Examples:
//   - "profile-compose" -> "profile"
//   - "profile-list" -> "profile"
//   - "stickymemory" -> "stickymemory"
//   - "code-reasoning" -> "code"
func ToolCategory(toolName string) string {
	if idx := strings.Index(toolName, "-"); idx > 0 {
		return toolName[:idx]
	}
	return toolName
}

// IsToolEnabled checks if a tool is enabled in the configuration.
// It checks in order: tool-specific setting, category setting, then defaults to true.
func (c *Config) IsToolEnabled(toolName string) bool {
	// Check tool-specific setting first
	if tool, ok := c.Tools[toolName]; ok && tool.Enabled != nil {
		return *tool.Enabled
	}

	// Check category setting
	category := ToolCategory(toolName)
	if category != toolName {
		if cat, ok := c.Tools[category]; ok && cat.Enabled != nil {
			return *cat.Enabled
		}
	}

	// Default: enabled
	return true
}

// IsKnownTool returns true if the tool name is in the known tools list.
func IsKnownTool(toolName string) bool {
	for _, known := range KnownTools {
		if known == toolName {
			return true
		}
	}
	// Also accept category names
	for _, known := range KnownTools {
		if ToolCategory(known) == toolName {
			return true
		}
	}
	return false
}

// WarnUnknownTools logs warnings for any tool names that are not recognized.
func WarnUnknownTools(toolNames []string) {
	for _, name := range toolNames {
		if !IsKnownTool(name) {
			logging.Logger().Warn("unknown tool name in configuration",
				"tool", name,
				"hint", "check spelling or see available tools with 'brains serve --help'",
			)
		}
	}
}
