package main

import (
	"io/fs"
	"log"
	"os"

	zombiekit "github.com/zombiekit/brains"
	internalcli "github.com/zombiekit/brains/internal/cli"
	"github.com/zombiekit/brains/internal/profile"
	"github.com/zombiekit/brains/internal/step"
	"github.com/zombiekit/brains/internal/version"
)

func init() {
	// Register embedded profiles so they're available as fallbacks
	profile.SetEmbeddedFS(zombiekit.EmbeddedProfiles)

	// Register embedded steps (strip "templates/" prefix so paths become "steps/*.md")
	// EmbeddedSteps embeds templates/steps/* -> files at templates/steps/feature.md
	// Step loader expects steps/feature.md
	if stepsSubFS, err := fs.Sub(zombiekit.EmbeddedSteps, "templates"); err == nil {
		step.SetEmbeddedFS(stepsSubFS)
	}

	// Register embedded templates (strip "templates/" prefix so paths become "templates/*.md")
	// EmbeddedTemplates embeds templates/templates/* -> files at templates/templates/spec-template.md
	// Initiative tool expects templates/spec-template.md
	if templatesSubFS, err := fs.Sub(zombiekit.EmbeddedTemplates, "templates"); err == nil {
		step.SetTemplateFS(templatesSubFS)
	}
}

func main() {
	app := internalcli.NewApp(version.Get())

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
