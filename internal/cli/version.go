package cli

import (
	"fmt"

	"github.com/urfave/cli/v2"
	"github.com/zombiekit/brains/internal/version"
)

// newVersionCommand creates the version subcommand that displays version and commit info.
func newVersionCommand(info *version.BuildInfo) *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print version and build information",
		Action: func(c *cli.Context) error {
			fmt.Println(info.PrettyPrint())
			return nil
		},
	}
}
