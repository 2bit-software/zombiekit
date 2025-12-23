// Package zombiekit provides embedded assets for the brains CLI.
package zombiekit

import "embed"

// EmbeddedProfiles contains the default profiles embedded at build time.
// These profiles serve as fallbacks when no local/global profiles exist.
//
//go:embed profiles/*
var EmbeddedProfiles embed.FS
