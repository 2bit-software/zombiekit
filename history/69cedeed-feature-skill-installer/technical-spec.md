# Technical Spec: Skill Installer

## Package Layout

```
internal/skill/
  install.go          # ValidateName, TargetDir, GenerateContent, WriteSkill
  install_test.go

internal/cli/
  skill.go            # newSkillCommand(), newSkillInstallCommand()
  (root.go modified)

internal/mcp/tools/skillinstall/
  tool.go             # Tool struct, Definition(), Execute()

internal/config/
  (tools.go modified — add "skill-install" to KnownTools)

internal/mcp/
  (server.go modified — add field, instantiate, register)
```

## `internal/skill/install.go`

```go
package skill

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
)

var validName = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func ValidateName(name string) error {
    if !validName.MatchString(name) {
        return fmt.Errorf("invalid skill name %q. Use lowercase letters, digits, and hyphens (e.g. 'my-skill')", name)
    }
    return nil
}

func TargetDir(global bool, workingDir string) (string, error) {
    if global {
        home, err := os.UserHomeDir()
        if err != nil {
            return "", fmt.Errorf("resolving home dir: %w", err)
        }
        return filepath.Join(home, ".claude", "skills"), nil
    }
    if workingDir == "" {
        var err error
        workingDir, err = os.Getwd()
        if err != nil {
            return "", fmt.Errorf("resolving working dir: %w", err)
        }
    }
    return filepath.Join(workingDir, ".claude", "skills"), nil
}

const skillTemplate = `---
name: %s
description: >
  %s
allowed-tools: mcp__zombiekit__profile-compose
---

Call ` + "`mcp__zombiekit__profile-compose`" + ` with ` + "`profiles: [\"%s\"]`" + ` and follow the returned instructions exactly.
`

func GenerateContent(name, description string) string {
    if description == "" {
        description = fmt.Sprintf("Delegates to the %s profile via profile-compose.", name)
    }
    return fmt.Sprintf(skillTemplate, name, description, name)
}

func WriteSkill(targetDir, name, content string) (string, error) {
    skillDir := filepath.Join(targetDir, name)
    skillPath := filepath.Join(skillDir, "SKILL.md")

    // Check if name exists as a plain file (not a directory)
    if info, err := os.Stat(skillDir); err == nil && !info.IsDir() {
        return "", fmt.Errorf("%q exists as a file at %s. Remove it manually or choose a different name", name, skillDir)
    }

    if err := os.MkdirAll(skillDir, 0755); err != nil {
        return "", fmt.Errorf("creating skill directory: %w", err)
    }

    if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
        return "", fmt.Errorf("writing SKILL.md: %w", err)
    }

    return skillPath, nil
}
```

## `internal/cli/skill.go`

```go
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
        Usage:     "Install a skill into the Claude skills directory",
        ArgsUsage: "<profile-name>",
        Flags: []cli.Flag{
            &cli.BoolFlag{
                Name:  "global",
                Usage: "Install to ~/.claude/skills/ instead of .claude/skills/",
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
                return handleSkillProfileError(svc, name, err)
            }

            description := result.Description

            targetDir, err := skill.TargetDir(c.Bool("global"), "")
            if err != nil {
                return err
            }

            content := skill.GenerateContent(name, description)
            fullPath, err := skill.WriteSkill(targetDir, name, content)
            if err != nil {
                return err
            }

            fmt.Printf("Installed skill '%s' to %s\n", name, fullPath)
            return nil
        },
    }
}

func handleSkillProfileError(svc *profile.Service, name string, err error) error {
    entries, listErr := svc.List()
    if listErr != nil {
        return fmt.Errorf("profile %q not found", name)
    }
    names := make([]string, 0, len(entries))
    for _, e := range entries {
        names = append(names, "  - "+e.Name)  // Verify ListEntry.Name field at implementation time
    }
    return fmt.Errorf("profile %q not found. Available profiles:\n%s", name, strings.Join(names, "\n"))
}
```

## `internal/mcp/tools/skillinstall/tool.go`

```go
package skillinstall

import (
    "context"
    "fmt"
    "strings"

    "github.com/2bit-software/zombiekit/internal/profile"
    "github.com/2bit-software/zombiekit/internal/skill"
)

type Tool struct{}

func NewTool() *Tool {
    return &Tool{}
}

func (t *Tool) Execute(ctx context.Context, args map[string]any) (string, error) {
    name, _ := args["name"].(string)
    scope, _ := args["scope"].(string)
    workingDir, _ := args["working_directory"].(string)

    if name == "" {
        return "", fmt.Errorf("name is required")
    }
    if scope == "" {
        return "", fmt.Errorf("scope is required (local or global)")
    }

    if err := skill.ValidateName(name); err != nil {
        return "", err
    }

    svc, err := profile.NewService(workingDir)
    if err != nil {
        return "", fmt.Errorf("initializing profile service: %w", err)
    }

    result, err := svc.Show(name, false)
    if err != nil {
        return "", buildProfileNotFoundError(svc, name)
    }

    description := result.Description

    targetDir, err := skill.TargetDir(scope == "global", workingDir)
    if err != nil {
        return "", err
    }

    content := skill.GenerateContent(name, description)
    fullPath, err := skill.WriteSkill(targetDir, name, content)
    if err != nil {
        return "", err
    }

    return fmt.Sprintf("Installed skill '%s' to %s", name, fullPath), nil
}

func buildProfileNotFoundError(svc *profile.Service, name string) error {
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
```

## `internal/mcp/server.go` changes

```go
// Add to Server struct:
skillInstallTool *skillinstalltool.Tool

// Add to NewServer():
skillInstallTool: skillinstalltool.NewTool(),

// Add to registerTools():
if s.config.IsToolEnabled("skill-install") {
    t := mcp.NewTool("skill-install",
        mcp.WithDescription("Install a zombiekit skill into the local or global Claude skills directory"),
        mcp.WithString("name",
            mcp.Required(),
            mcp.Description("Profile name to install as a skill"),
        ),
        mcp.WithString("scope",
            mcp.Required(),
            mcp.Description("Installation scope: 'local' (.claude/skills/) or 'global' (~/.claude/skills/)"),
            mcp.Enum("local", "global"),
        ),
        mcp.WithString("working_directory",
            mcp.Description("Working directory for local install (defaults to process CWD)"),
        ),
    )
    s.mcpServer.AddTool(t, s.handleSkillInstall)
}

// New handler:
func (s *Server) handleSkillInstall(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
    args, ok := req.Params.Arguments.(map[string]any)
    if !ok {
        return mcp.NewToolResultError("invalid arguments format"), nil
    }
    result, err := s.skillInstallTool.Execute(ctx, args)
    if err != nil {
        return mcp.NewToolResultError(err.Error()), nil
    }
    return mcp.NewToolResultText(result), nil
}
```

## Verification Items

Before implementing, read these to confirm field names:
- `internal/profile/service.go` lines 124-165: what does `ShowResult` look like? Confirm `result.Description` path.
- `internal/profile/types.go`: confirm `ListEntry` struct has `Name string` field.

## SKILL.md Template (exact output)

For `name=create-pr` with description `"Creates a pull request..."`:
```
---
name: create-pr
description: >
  Creates a pull request...
allowed-tools: mcp__zombiekit__profile-compose
---

Call `mcp__zombiekit__profile-compose` with `profiles: ["create-pr"]` and follow the returned instructions exactly.
```

Note: trailing newline after the body line (standard for text files).
