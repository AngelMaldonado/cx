package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/amald/cx/internal/agents"
	"github.com/amald/cx/internal/direction"
	"github.com/amald/cx/internal/project"
	"github.com/amald/cx/internal/ui"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize CX in the current project",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	// Step 1: Verify git repo
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx init must be run inside a git repo")
		return fmt.Errorf("not a git repository")
	}

	userName, _ := project.GitUserName()

	fmt.Println()
	if userName != "" {
		ui.PrintHeader(fmt.Sprintf("cx init — %s", userName))
	} else {
		ui.PrintHeader("cx init")
	}
	ui.PrintDivider()

	// Step 2: Scaffold docs/
	ui.PrintHeader("docs/ structure")
	result, err := project.ScaffoldDocs(rootDir)
	if err != nil {
		ui.PrintError(fmt.Sprintf("scaffolding docs: %v", err))
		return err
	}
	for _, f := range result.Created {
		ui.PrintSuccess(fmt.Sprintf("created %s", f))
	}
	for _, f := range result.Skipped {
		ui.PrintMuted(fmt.Sprintf("skipped %s (exists)", f))
	}

	// Step 3: Scaffold .cx/
	created, err := project.ScaffoldCXCache(rootDir)
	if err != nil {
		ui.PrintError(fmt.Sprintf("scaffolding .cx: %v", err))
		return err
	}
	if created {
		ui.PrintSuccess("created .cx/")
	} else {
		ui.PrintMuted("skipped .cx/ (exists)")
	}

	// Step 4: Agent selection
	fmt.Println()
	selectedSlugs, err := ui.NewAgentSelect()
	if err != nil {
		return err
	}

	ui.PrintHeader("agent setup")
	for _, slug := range selectedSlugs {
		agent, ok := agents.BySlug(slug)
		if !ok {
			continue
		}

		if err := agents.EnsureAgentDir(rootDir, agent); err != nil {
			ui.PrintError(fmt.Sprintf("creating %s dirs: %v", agent.Name, err))
			continue
		}

		if err := agents.WriteConfigFile(rootDir, agent); err != nil {
			ui.PrintError(fmt.Sprintf("writing %s config: %v", agent.Name, err))
			continue
		}

		count, err := agents.WriteSkills(rootDir, agent)
		if err != nil {
			ui.PrintError(fmt.Sprintf("writing %s skills: %v", agent.Name, err))
			continue
		}

		ui.PrintSuccess(fmt.Sprintf("%s — config + %d skills installed", agent.Name, count))
	}

	// Step 5: DIRECTION.md
	directionPath := filepath.Join(rootDir, "DIRECTION.md")
	if _, err := os.Stat(directionPath); os.IsNotExist(err) {
		fmt.Println()
		projectType, err := ui.NewProjectTypeSelect()
		if err != nil {
			return err
		}

		fmt.Println()
		priorities, err := ui.NewPrioritiesSelect()
		if err != nil {
			return err
		}

		content := direction.GenerateDirection(projectType, priorities)
		if err := os.WriteFile(directionPath, []byte(content), 0o644); err != nil {
			ui.PrintError(fmt.Sprintf("writing DIRECTION.md: %v", err))
		} else {
			fmt.Println()
			ui.PrintSuccess(fmt.Sprintf("created DIRECTION.md (%s)", direction.ProjectTypeLabel(projectType)))
		}
	} else {
		fmt.Println()
		ui.PrintMuted("skipped DIRECTION.md (exists)")
	}

	// Step 6: Git hooks
	ui.PrintHeader("git hooks")
	hooks := []string{"post-merge", "post-checkout"}
	for _, hookType := range hooks {
		existsAlready, err := project.InstallHook(rootDir, hookType, false)
		if err != nil {
			ui.PrintError(fmt.Sprintf("installing %s hook: %v", hookType, err))
			continue
		}
		if existsAlready {
			confirmed, err := ui.NewConfirmPrompt(fmt.Sprintf("%s hook exists without CX marker. Overwrite?", hookType))
			if err != nil {
				return err
			}
			if confirmed {
				if _, err := project.InstallHook(rootDir, hookType, true); err != nil {
					ui.PrintError(fmt.Sprintf("installing %s hook: %v", hookType, err))
				} else {
					ui.PrintSuccess(fmt.Sprintf("installed %s hook (overwritten)", hookType))
				}
			} else {
				ui.PrintMuted(fmt.Sprintf("skipped %s hook", hookType))
			}
		} else {
			ui.PrintSuccess(fmt.Sprintf("installed %s hook", hookType))
		}
	}

	// Step 7: Register project
	isFirstInit := project.IsFirstEverInit()

	registered, err := project.RegisterProject(rootDir)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("registering project: %v", err))
	} else if registered {
		ui.PrintSuccess("registered in ~/.cx/projects.json")
	}

	// Step 8: MCP check
	hasMCP, missing, err := project.CheckMCP(rootDir)
	if err != nil {
		ui.PrintWarning(fmt.Sprintf("checking MCP config: %v", err))
	} else {
		if !hasMCP {
			ui.PrintWarning("no .mcp.json found")
		}
		for _, server := range missing {
			fmt.Println()
			ui.PrintWarning(fmt.Sprintf("%s MCP server not configured", server))
			ui.PrintMuted("add this to your .mcp.json:")
			ui.PrintMuted(project.LinearMCPSnippet())
		}
	}

	// Step 9: First-time preferences
	if isFirstInit {
		fmt.Println()
		autoUpdate, err := ui.NewConfirmPrompt("Enable automatic update checks?")
		if err != nil {
			return err
		}
		prefs := &project.Preferences{AutoUpdateCheck: autoUpdate}
		if err := project.SavePreferences(prefs); err != nil {
			ui.PrintWarning(fmt.Sprintf("saving preferences: %v", err))
		}
	}

	// Step 10: Summary
	fmt.Println()
	ui.PrintDivider()
	ui.PrintBanner("CX initialized")
	fmt.Println()
	ui.PrintItem("project", rootDir)
	ui.PrintItem("agents", fmt.Sprintf("%d configured", len(selectedSlugs)))
	if _, err := os.Stat(directionPath); err == nil {
		ui.PrintItem("direction", "DIRECTION.md")
	}
	fmt.Println()
	ui.PrintHeader("next steps")
	ui.PrintMuted("  1. Review DIRECTION.md and customize for your project")
	ui.PrintMuted("  2. Run cx doctor to verify setup")
	ui.PrintMuted("  3. Start a conversation with your AI agent")
	fmt.Println()

	return nil
}
