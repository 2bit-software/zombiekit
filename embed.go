// Package zombiekit provides embedded assets for the brains CLI.
package zombiekit

import "embed"

// EmbeddedProfiles contains the default profiles embedded at build time.
// These profiles serve as fallbacks when no local/global profiles exist.
//
//go:embed profiles/*
var EmbeddedProfiles embed.FS

// EmbeddedCommands contains Claude Code command files (skills) embedded at build time.
// These are copied to .claude/commands/ during `brains init`.
//
//go:embed integrations/claude/commands/*
var EmbeddedCommands embed.FS

// EmbeddedTemplates contains specification templates embedded at build time.
// These are copied to .brains/templates/ during `brains init`.
//
//go:embed templates/templates/*
var EmbeddedTemplates embed.FS

// EmbeddedSteps contains the default step definitions embedded at build time.
// These steps serve as fallbacks when no local/global step definitions exist.
//
//go:embed templates/steps/*
var EmbeddedSteps embed.FS
