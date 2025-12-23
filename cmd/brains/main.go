package main

import (
	"log"
	"os"

	internalcli "github.com/zombiekit/brains/internal/cli"
	"github.com/zombiekit/brains/internal/version"
)

func main() {
	app := internalcli.NewApp(version.Get())

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
