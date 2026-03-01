package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/amald/cx/internal/agents"
	"github.com/amald/cx/internal/project"
	"github.com/amald/cx/internal/ui"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Regenerate agent configs, skills, and MCP settings",
	RunE:  runSync,
}

func runSync(cmd *cobra.Command, args []string) error {
	// Step 1: Verify git repo
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx sync must be run inside a git repo")
		return fmt.Errorf("not a git repository")
	}

	fmt.Println()
	ui.PrintHeader("cx sync")
	ui.PrintDivider()
	ui.Pause(300 * time.Millisecond)

	// Step 2: Scaffold docs/ + .cx/ (fill gaps)
	var scaffoldResult *project.ScaffoldResult
	var cxCreated bool
	scaffoldErr := ui.RunWithSpinner("checking project structure", 500*time.Millisecond, func() error {
		var err error
		scaffoldResult, err = project.ScaffoldDocs(rootDir)
		if err != nil {
			return err
		}
		cxCreated, err = project.ScaffoldCXCache(rootDir)
		return err
	})
	if scaffoldErr != nil {
		ui.PrintError(fmt.Sprintf("scaffolding: %v", scaffoldErr))
		return scaffoldErr
	}
	for _, f := range scaffoldResult.Created {
		ui.PrintSuccess(fmt.Sprintf("created %s", f))
	}
	if cxCreated {
		ui.PrintSuccess("created .cx/")
	}
	if len(scaffoldResult.Created) == 0 && !cxCreated {
		ui.PrintMuted("project structure up to date")
	}
	ui.Pause(200 * time.Millisecond)

	// Step 3: Detect installed agents
	installed := agents.DetectInstalled(rootDir)
	if len(installed) == 0 {
		fmt.Println()
		ui.PrintWarning("no agent directories found — run cx init first")
		return nil
	}

	// Step 4: Check what needs syncing
	var statuses []agents.SyncStatus
	_ = ui.RunWithSpinner("checking agent configs + skills", 600*time.Millisecond, func() error {
		statuses = agents.CheckSyncStatus(rootDir, installed)
		return nil
	})

	// Report what's out of date
	fmt.Println()
	ui.PrintHeader("changes detected")
	anyChanges := false
	for _, s := range statuses {
		if s.UpToDate() {
			ui.PrintMuted(fmt.Sprintf("%s — up to date", s.Agent.Name))
			continue
		}
		anyChanges = true
		ui.PrintWarning(fmt.Sprintf("%s — %s", s.Agent.Name, s.Summary()))

		// Detail what specifically changed
		if s.ConfigChanged {
			ui.PrintMuted(fmt.Sprintf("  → %s", s.Agent.ConfigFile))
		}
		for _, slug := range s.SkillsMissing {
			ui.PrintMuted(fmt.Sprintf("  → %s/%s/SKILL.md (missing)", s.Agent.SkillsDir, slug))
		}
		for _, slug := range s.SkillsChanged {
			ui.PrintMuted(fmt.Sprintf("  → %s/%s/SKILL.md (changed)", s.Agent.SkillsDir, slug))
		}
		for _, slug := range s.SubagentsMissing {
			ext := ".md"
			if s.Agent.Slug == "codex" {
				ext = ".toml"
			}
			ui.PrintMuted(fmt.Sprintf("  → %s/%s%s (missing)", s.Agent.AgentsDir, slug, ext))
		}
		for _, slug := range s.SubagentsChanged {
			ext := ".md"
			if s.Agent.Slug == "codex" {
				ext = ".toml"
			}
			ui.PrintMuted(fmt.Sprintf("  → %s/%s%s (changed)", s.Agent.AgentsDir, slug, ext))
		}
	}

	// Check hooks + MCP
	hookChanges := checkHookStatus(rootDir)
	if len(hookChanges) > 0 {
		anyChanges = true
		for _, msg := range hookChanges {
			ui.PrintWarning(msg)
		}
	}
	ui.Pause(200 * time.Millisecond)

	if !anyChanges {
		fmt.Println()
		ui.PrintDivider()
		ui.PrintBanner("everything up to date")
		fmt.Println()
		return nil
	}

	// Step 5: Apply sync
	fmt.Println()
	type agentResult struct {
		name      string
		skills    int
		subagents int
		err       error
	}
	var agentResults []agentResult

	_ = ui.RunWithSpinner("syncing agent configs + skills", 600*time.Millisecond, func() error {
		for _, agent := range installed {
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

	ui.PrintHeader("synced")
	for _, r := range agentResults {
		if r.err != nil {
			ui.PrintError(fmt.Sprintf("%s — %v", r.name, r.err))
		} else {
			var parts []string
			parts = append(parts, "config")
			parts = append(parts, fmt.Sprintf("%d skills", r.skills))
			if r.subagents > 0 {
				parts = append(parts, fmt.Sprintf("%d subagents", r.subagents))
			}
			ui.PrintSuccess(fmt.Sprintf("%s — %s", r.name, strings.Join(parts, " + ")))
		}
	}
	ui.Pause(200 * time.Millisecond)

	// Step 6: MCP configs
	slugs := make([]string, len(installed))
	for i, a := range installed {
		slugs[i] = a.Slug
	}

	mcpErr := ui.RunWithSpinner("configuring MCP servers", 400*time.Millisecond, func() error {
		return project.WriteMCPConfigs(rootDir, slugs)
	})
	if mcpErr != nil {
		ui.PrintError(fmt.Sprintf("writing MCP configs: %v", mcpErr))
	} else {
		ui.PrintSuccess("MCP servers synced (context7 + linear)")
	}
	ui.Pause(200 * time.Millisecond)

	// Step 7: Git hooks
	_ = ui.RunWithSpinner("syncing git hooks", 400*time.Millisecond, func() error {
		for _, hookType := range []string{"post-merge", "post-checkout"} {
			project.InstallHook(rootDir, hookType, true)
		}
		return nil
	})
	ui.PrintSuccess("git hooks synced (post-merge + post-checkout)")
	ui.Pause(200 * time.Millisecond)

	// Step 8: Summary
	fmt.Println()
	ui.PrintDivider()
	ui.PrintBanner("sync complete")
	fmt.Println()

	return nil
}

func checkHookStatus(rootDir string) []string {
	var changes []string
	for _, hookType := range []string{"post-merge", "post-checkout"} {
		hasCX, _ := project.HookContainsCX(rootDir, hookType)
		if !hasCX {
			changes = append(changes, fmt.Sprintf("%s hook — needs sync", hookType))
		}
	}
	return changes
}
