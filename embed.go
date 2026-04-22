// Package zombiekit provides embedded assets for the brains CLI.
package zombiekit

import (
	"embed"
	"io/fs"
)

//go:embed embed/profiles/*
var embeddedProfiles embed.FS

//go:embed embed/integrations/claude/commands/*
var embeddedClaudeCommands embed.FS

//go:embed embed/commands/*
var embeddedBrainsCommands embed.FS

//go:embed embed/templates embed/templates/init-spec-creator/*
var embeddedTemplates embed.FS

//go:embed embed/scripts embed/scripts/commit-message/* embed/scripts/permissions-audit/* embed/scripts/repo-auditor/*
var embeddedScripts embed.FS

//go:embed embed/workflows/*
var embeddedWorkflows embed.FS

//go:embed embed/integrations/opencode/brains.ts
var embeddedOpencodeShim embed.FS

// Exported filesystems with embed/ prefix stripped via fs.Sub.
// Consumers see the same paths they expect (e.g., "profiles/*", "workflows/*").
var (
	EmbeddedProfiles       fs.FS
	EmbeddedClaudeCommands fs.FS
	EmbeddedBrainsCommands fs.FS
	EmbeddedTemplates      fs.FS
	EmbeddedScripts        fs.FS
	EmbeddedWorkflows      fs.FS
	// TODO(opencode): EmbeddedOpencodeShim is exported but not yet consumed
	// by any CLI subcommand. Add a `brains init opencode` or similar that
	// extracts brains.ts into .opencode/plugins/ for the user.
	EmbeddedOpencodeShim fs.FS
)

func init() {
	var err error
	EmbeddedProfiles, err = fs.Sub(embeddedProfiles, "embed")
	if err != nil {
		panic("embed: profiles: " + err.Error())
	}
	EmbeddedClaudeCommands, err = fs.Sub(embeddedClaudeCommands, "embed")
	if err != nil {
		panic("embed: claude commands: " + err.Error())
	}
	EmbeddedBrainsCommands, err = fs.Sub(embeddedBrainsCommands, "embed")
	if err != nil {
		panic("embed: brains commands: " + err.Error())
	}
	EmbeddedTemplates, err = fs.Sub(embeddedTemplates, "embed")
	if err != nil {
		panic("embed: templates: " + err.Error())
	}
	EmbeddedScripts, err = fs.Sub(embeddedScripts, "embed")
	if err != nil {
		panic("embed: scripts: " + err.Error())
	}
	EmbeddedWorkflows, err = fs.Sub(embeddedWorkflows, "embed")
	if err != nil {
		panic("embed: workflows: " + err.Error())
	}
	EmbeddedOpencodeShim, err = fs.Sub(embeddedOpencodeShim, "embed")
	if err != nil {
		panic("embed: opencode shim: " + err.Error())
	}
}
