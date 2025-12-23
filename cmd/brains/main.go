package main

import (
	"log"
	"os"

	zombiekit "github.com/zombiekit/brains"
	internalcli "github.com/zombiekit/brains/internal/cli"
	"github.com/zombiekit/brains/internal/profile"
	"github.com/zombiekit/brains/internal/version"
)

func init() {
	// Register embedded profiles so they're available as fallbacks
	profile.SetEmbeddedFS(zombiekit.EmbeddedProfiles)
}

func main() {
	app := internalcli.NewApp(version.Get())

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
