package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// newVersionCommand creates the version subcommand that displays version and commit info.
func newVersionCommand(version, commit string) *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print version and build information",
		Action: func(c *cli.Context) error {
			fmt.Printf("brains version %s\n", version)
			fmt.Printf("commit: %s\n", commit)
			return nil
		},
	}
}
