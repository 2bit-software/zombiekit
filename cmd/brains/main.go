package main

import (
	"log"
	"os"

	zombiekit "github.com/2bit-software/zombiekit"
	internalcli "github.com/2bit-software/zombiekit/internal/cli"
	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/2bit-software/zombiekit/internal/step"
	"github.com/2bit-software/zombiekit/internal/version"
	"github.com/2bit-software/zombiekit/internal/workflow"
)

func init() {
	// Register embedded filesystems (embed.go handles prefix stripping via fs.Sub)
	profile.SetEmbeddedFS(zombiekit.EmbeddedProfiles)
	workflow.SetEmbeddedFS(zombiekit.EmbeddedWorkflows)
	step.SetTemplateFS(zombiekit.EmbeddedTemplates)
}

func main() {
	app := internalcli.NewApp(version.Get())

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
