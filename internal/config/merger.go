package config

// Merge applies settings from src onto the receiver (dst).
// Values from src override values in dst when set (non-nil).
// This allows layered configuration: global -> local -> CLI.
func (dst *Config) Merge(src *Config) {
	if src == nil {
		return
	}

	for name, srcTool := range src.Tools {
		if srcTool.Enabled != nil {
			dstTool := dst.Tools[name]
			dstTool.Enabled = srcTool.Enabled
			dst.Tools[name] = dstTool
		}
	}
}

// ApplyCLIOverrides applies command-line flag overrides to the configuration.
// enabledTools and disabledTools are lists of tool names from CLI flags.
// CLI flags have the highest precedence and always override config file settings.
func (c *Config) ApplyCLIOverrides(enabledTools, disabledTools []string) {
	// Apply disabled tools first
	for _, name := range disabledTools {
		enabled := false
		c.Tools[name] = ToolConfig{Enabled: &enabled}
	}

	// Apply enabled tools (these override disabled if both specified)
	for _, name := range enabledTools {
		enabled := true
		c.Tools[name] = ToolConfig{Enabled: &enabled}
	}
}
