package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

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
	ui.Pause(300 * time.Millisecond)

	// Step 2: Scaffold docs/
	var scaffoldResult *project.ScaffoldResult
	scaffoldErr := ui.RunWithSpinner("scaffolding docs/", 600*time.Millisecond, func() error {
		var err error
		scaffoldResult, err = project.ScaffoldDocs(rootDir)
		return err
	})
	if scaffoldErr != nil {
		ui.PrintError(fmt.Sprintf("scaffolding docs: %v", scaffoldErr))
		return scaffoldErr
	}
	ui.PrintHeader("docs/ structure")
	for _, f := range scaffoldResult.Created {
		ui.PrintSuccess(fmt.Sprintf("created %s", f))
	}
	for _, f := range scaffoldResult.Skipped {
		ui.PrintMuted(fmt.Sprintf("skipped %s (exists)", f))
	}
	ui.Pause(200 * time.Millisecond)

	// Step 3: Scaffold .cx/
	var cxCreated bool
	cxErr := ui.RunWithSpinner("preparing .cx/", 400*time.Millisecond, func() error {
		var err error
		cxCreated, err = project.ScaffoldCXCache(rootDir)
		return err
	})
	if cxErr != nil {
		ui.PrintError(fmt.Sprintf("scaffolding .cx: %v", cxErr))
		return cxErr
	}
	if cxCreated {
		ui.PrintSuccess("created .cx/")
	} else {
		ui.PrintMuted("skipped .cx/ (exists)")
	}
	ui.Pause(300 * time.Millisecond)

	// Step 4: Agent selection (interactive form, then spinner for setup)
	fmt.Println()
	selectedSlugs, err := ui.NewAgentSelect()
	if err != nil {
		return err
	}
	ui.Pause(200 * time.Millisecond)

	type agentResult struct {
		name      string
		skills    int
		subagents int
		err       error
	}
	var agentResults []agentResult

	_ = ui.RunWithSpinner("installing agent configs + skills", 800*time.Millisecond, func() error {
		for _, slug := range selectedSlugs {
			agent, ok := agents.BySlug(slug)
			if !ok {
				continue
			}
			r := agentResult{name: agent.Name}
			if err := agents.EnsureAgentDir(rootDir, agent); err != nil {
				r.err = fmt.Errorf("creating dirs: %w", err)
				agentResults = append(agentResults, r)
				continue
			}
			if err := agents.WriteConfigFile(rootDir, agent); err != nil {
				r.err = fmt.Errorf("writing config: %w", err)
				agentResults = append(agentResults, r)
				continue
			}
			skillCount, err := agents.WriteSkills(rootDir, agent)
			if err != nil {
				r.err = fmt.Errorf("writing skills: %w", err)
				agentResults = append(agentResults, r)
				continue
			}
			r.skills = skillCount
			saCount, err := agents.WriteSubagents(rootDir, agent)
			r.subagents = saCount
			r.err = err
			agentResults = append(agentResults, r)
		}
		return nil
	})

	ui.PrintHeader("agent setup")
	for _, r := range agentResults {
		if r.err != nil {
			ui.PrintError(fmt.Sprintf("%s — %v", r.name, r.err))
		} else if r.subagents > 0 {
			ui.PrintSuccess(fmt.Sprintf("%s — config + %d skills + %d subagents installed", r.name, r.skills, r.subagents))
		} else {
			ui.PrintSuccess(fmt.Sprintf("%s — config + %d skills installed", r.name, r.skills))
		}
	}
	ui.Pause(300 * time.Millisecond)

	// Step 5: DIRECTION.md
	directionPath := filepath.Join(rootDir, "docs", "memory", "DIRECTION.md")
	if _, err := os.Stat(directionPath); os.IsNotExist(err) {
		fmt.Println()
		projectType, err := ui.NewProjectTypeSelect()
		if err != nil {
			return err
		}
		ui.Pause(200 * time.Millisecond)

		fmt.Println()
		priorities, err := ui.NewPrioritiesSelect()
		if err != nil {
			return err
		}
		ui.Pause(200 * time.Millisecond)

		dirErr := ui.RunWithSpinner("generating DIRECTION.md", 700*time.Millisecond, func() error {
			content := direction.GenerateDirection(projectType, priorities)
			return os.WriteFile(directionPath, []byte(content), 0o644)
		})
		if dirErr != nil {
			ui.PrintError(fmt.Sprintf("writing DIRECTION.md: %v", dirErr))
		} else {
			ui.PrintSuccess(fmt.Sprintf("created DIRECTION.md (%s)", direction.ProjectTypeLabel(projectType)))
		}
	} else {
		fmt.Println()
		ui.PrintMuted("skipped DIRECTION.md (exists)")
	}
	ui.Pause(300 * time.Millisecond)

	// Step 6: Git hooks (spinner for batch install, interactive for conflicts)
	type hookResult struct {
		hookType      string
		existsAlready bool
		installed     bool
		err           error
	}
	var hookResults []hookResult

	_ = ui.RunWithSpinner("installing git hooks", 500*time.Millisecond, func() error {
		for _, hookType := range []string{"post-merge", "post-checkout"} {
			existsAlready, err := project.InstallHook(rootDir, hookType, false)
			hookResults = append(hookResults, hookResult{
				hookType:      hookType,
				existsAlready: existsAlready,
				installed:     !existsAlready && err == nil,
				err:           err,
			})
		}
		return nil
	})

	ui.PrintHeader("git hooks")
	for _, hr := range hookResults {
		if hr.err != nil {
			ui.PrintError(fmt.Sprintf("installing %s hook: %v", hr.hookType, hr.err))
		} else if hr.installed {
			ui.PrintSuccess(fmt.Sprintf("installed %s hook", hr.hookType))
		} else if hr.existsAlready {
			confirmed, err := ui.NewConfirmPrompt(fmt.Sprintf("%s hook exists without CX marker. Overwrite?", hr.hookType))
			if err != nil {
				return err
			}
			if confirmed {
				overwriteErr := ui.RunWithSpinner(fmt.Sprintf("overwriting %s hook", hr.hookType), 300*time.Millisecond, func() error {
					_, err := project.InstallHook(rootDir, hr.hookType, true)
					return err
				})
				if overwriteErr != nil {
					ui.PrintError(fmt.Sprintf("installing %s hook: %v", hr.hookType, overwriteErr))
				} else {
					ui.PrintSuccess(fmt.Sprintf("installed %s hook (overwritten)", hr.hookType))
				}
			} else {
				ui.PrintMuted(fmt.Sprintf("skipped %s hook", hr.hookType))
			}
		}
	}
	ui.Pause(200 * time.Millisecond)

	// Step 7: Register project
	isFirstInit := project.IsFirstEverInit()

	var registered bool
	regErr := ui.RunWithSpinner("registering project", 400*time.Millisecond, func() error {
		var err error
		registered, err = project.RegisterProject(rootDir)
		return err
	})
	if regErr != nil {
		ui.PrintWarning(fmt.Sprintf("registering project: %v", regErr))
	} else if registered {
		ui.PrintSuccess("registered in ~/.cx/projects.json")
	}
	ui.Pause(200 * time.Millisecond)

	// Step 8: MCP check (no spinner — instant read)
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

	// Step 9: First-time preferences (interactive)
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
	ui.Pause(400 * time.Millisecond)
	fmt.Println()
	ui.PrintDivider()
	ui.PrintBanner("CX initialized")
	fmt.Println()
	ui.PrintItem("project", rootDir)
	ui.PrintItem("agents", fmt.Sprintf("%d configured", len(selectedSlugs)))
	if _, err := os.Stat(directionPath); err == nil {
		ui.PrintItem("direction", "docs/memory/DIRECTION.md")
	}
	fmt.Println()
	ui.PrintHeader("next steps")
	ui.PrintMuted("  1. Review docs/memory/DIRECTION.md and customize for your project")
	ui.Pause(100 * time.Millisecond)
	ui.PrintMuted("  2. Run cx doctor to verify setup")
	ui.Pause(100 * time.Millisecond)
	ui.PrintMuted("  3. Start a conversation with your AI agent")
	fmt.Println()

	return nil
}
