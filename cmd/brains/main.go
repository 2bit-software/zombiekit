package main

import (
	"log"
	"os"

	zombiekit "github.com/zombiekit/brains"
	internalcli "github.com/zombiekit/brains/internal/cli"
	"github.com/zombiekit/brains/internal/profile"
	"github.com/zombiekit/brains/internal/step"
	"github.com/zombiekit/brains/internal/version"
	"github.com/zombiekit/brains/internal/workflow"
)

func init() {
	// Register embedded filesystems (embed.go handles prefix stripping via fs.Sub)
	profile.SetEmbeddedFS(zombiekit.EmbeddedProfiles)
	workflow.SetEmbeddedFS(zombiekit.EmbeddedWorkflows)
	step.SetEmbeddedFS(zombiekit.EmbeddedSteps)
	step.SetTemplateFS(zombiekit.EmbeddedTemplates)
}

func main() {
	app := internalcli.NewApp(version.Get())

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
