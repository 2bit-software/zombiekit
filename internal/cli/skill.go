package cli

import (
	"fmt"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/2bit-software/zombiekit/internal/profile"
	"github.com/2bit-software/zombiekit/internal/skill"
)

func newSkillCommand() *cli.Command {
	return &cli.Command{
		Name:  "skill",
		Usage: "Manage Claude Code skills",
		Subcommands: []*cli.Command{
			newSkillInstallCommand(),
		},
	}
}

func newSkillInstallCommand() *cli.Command {
	return &cli.Command{
		Name:      "install",
		Usage:     "Install a profile as a Claude Code skill",
		ArgsUsage: "<profile-name>",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "global",
				Usage: "Install to ~/.claude/skills/ instead of .claude/skills/ in the current directory",
			},
		},
		Action: func(c *cli.Context) error {
			name := c.Args().First()
			if name == "" {
				return fmt.Errorf("profile name is required")
			}
			if err := skill.ValidateName(name); err != nil {
				return err
			}

			svc, err := profile.NewServiceWithSource(profile.SourceTypeBrains, "")
			if err != nil {
				return fmt.Errorf("initializing profile service: %w", err)
			}

			result, err := svc.Show(name, false)
			if err != nil {
				return skillProfileNotFoundError(svc, name)
			}

			targetDir, err := skill.TargetDir(c.Bool("global"), "")
			if err != nil {
				return err
			}

			content := skill.GenerateContent(name, result.Description)
			fullPath, err := skill.WriteSkill(targetDir, name, content)
			if err != nil {
				return err
			}

			fmt.Printf("Installed skill '%s' to %s\n", name, fullPath)
			return nil
		},
	}
}

func skillProfileNotFoundError(svc *profile.Service, name string) error {
	entries, err := svc.List()
	if err != nil {
		return fmt.Errorf("profile %q not found", name)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, "  - "+e.Name)
	}
	return fmt.Errorf("profile %q not found. Available profiles:\n%s", name, strings.Join(names, "\n"))
}
