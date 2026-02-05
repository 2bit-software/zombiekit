// Package zombiekit provides embedded assets for the brains CLI.
package zombiekit

import (
	"embed"
	"io/fs"
)

//go:embed embed/profiles/*
var embeddedProfiles embed.FS

//go:embed embed/integrations/claude/commands/*
var embeddedCommands embed.FS

//go:embed embed/templates/*
var embeddedTemplates embed.FS

//go:embed embed/workflows/*
var embeddedWorkflows embed.FS

// Exported filesystems with embed/ prefix stripped via fs.Sub.
// Consumers see the same paths they expect (e.g., "profiles/*", "workflows/*").
var (
	EmbeddedProfiles  fs.FS
	EmbeddedCommands  fs.FS
	EmbeddedTemplates fs.FS
	EmbeddedWorkflows fs.FS
)

func init() {
	var err error
	EmbeddedProfiles, err = fs.Sub(embeddedProfiles, "embed")
	if err != nil {
		panic("embed: profiles: " + err.Error())
	}
	EmbeddedCommands, err = fs.Sub(embeddedCommands, "embed")
	if err != nil {
		panic("embed: commands: " + err.Error())
	}
	EmbeddedTemplates, err = fs.Sub(embeddedTemplates, "embed")
	if err != nil {
		panic("embed: templates: " + err.Error())
	}
	EmbeddedWorkflows, err = fs.Sub(embeddedWorkflows, "embed")
	if err != nil {
		panic("embed: workflows: " + err.Error())
	}
}
