package main

import (
	"log"
	"os"

	internalcli "github.com/zombiekit/brains/internal/cli"
)

// Version info - set via ldflags at build time
var (
	version = "dev"
	commit  = "unknown"
)

func main() {
	app := internalcli.NewApp(version, commit)

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
