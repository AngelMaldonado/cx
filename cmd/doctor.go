package cmd

import (
	"fmt"
	"time"

	"github.com/amald/cx/internal/doctor"
	"github.com/amald/cx/internal/project"
	"github.com/amald/cx/internal/ui"
	"github.com/spf13/cobra"
)

var fixFlag bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check project health and fix common issues",
	RunE:  runDoctor,
}

func init() {
	doctorCmd.Flags().BoolVar(&fixFlag, "fix", false, "Auto-fix fixable issues")
}

type checkDef struct {
	label string
	fn    func(string) doctor.CheckGroup
}

var checks = []checkDef{
	{"checking docs/ structure", doctor.CheckDocsStructure},
	{"checking memory health", doctor.CheckMemoryHealth},
	{"checking index health", doctor.CheckIndexHealth},
	{"checking git hooks", doctor.CheckGitHooks},
	{"checking MCP config", doctor.CheckMCPConfig},
	{"checking skill files", doctor.CheckSkillFiles},
}

func runDoctor(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx doctor must be run inside a git repo")
		return fmt.Errorf("not a git repository")
	}

	fmt.Println()
	ui.PrintHeader("cx doctor")
	ui.PrintDivider()
	ui.Pause(300 * time.Millisecond)

	// Cascading check reveal
	allGroups, totalErrors, totalWarnings := runChecks(rootDir, 400*time.Millisecond, 120*time.Millisecond)

	fmt.Println()
	ui.PrintDivider()
	ui.PrintSummary(totalErrors, totalWarnings)

	if !fixFlag {
		fixable := doctor.CollectFixable(allGroups)
		if len(fixable) > 0 {
			fmt.Println()
			ui.PrintMuted(fmt.Sprintf("  %d fixable issue(s) — run cx doctor --fix to repair", len(fixable)))
		}
		if totalErrors > 0 {
			return errExitCode1
		}
		return nil
	}

	// --fix mode
	fixable := doctor.CollectFixable(allGroups)
	if len(fixable) == 0 {
		fmt.Println()
		ui.PrintSuccess("nothing to fix")
		return nil
	}

	fmt.Println()
	doctor.PrintFixableList(fixable)
	fmt.Println()

	confirmed, err := ui.NewConfirmPrompt("Fix these issues?")
	if err != nil {
		return err
	}
	if !confirmed {
		ui.PrintMuted("  skipped")
		return nil
	}

	// Apply fixes with per-item spinner
	fmt.Println()
	ui.PrintHeader("applying fixes")
	for i, item := range fixable {
		fixErr := ui.RunWithSpinner(fmt.Sprintf("fixing: %s", item.Label), 400*time.Millisecond, func() error {
			return item.Fix()
		})
		if fixErr != nil {
			ui.PrintError(fmt.Sprintf("fix failed: %s — %v", item.Label, fixErr))
		} else {
			ui.PrintSuccess(fmt.Sprintf("fixed: %s", item.Label))
		}
		if i < len(fixable)-1 {
			ui.Pause(100 * time.Millisecond)
		}
	}

	// Re-check after fixes
	ui.Pause(300 * time.Millisecond)
	fmt.Println()
	ui.PrintHeader("re-checking")
	ui.PrintDivider()

	_, totalErrors, totalWarnings = runChecks(rootDir, 300*time.Millisecond, 100*time.Millisecond)

	fmt.Println()
	ui.PrintDivider()
	ui.PrintSummary(totalErrors, totalWarnings)

	if totalErrors > 0 {
		return errExitCode1
	}
	return nil
}

func runChecks(rootDir string, spinnerDur, pauseDur time.Duration) ([]doctor.CheckGroup, int, int) {
	var allGroups []doctor.CheckGroup
	var totalErrors, totalWarnings int

	for _, c := range checks {
		var group doctor.CheckGroup
		_ = ui.RunWithSpinner(c.label, spinnerDur, func() error {
			group = c.fn(rootDir)
			return nil
		})
		e, w := doctor.PrintGroupReport(group)
		allGroups = append(allGroups, group)
		totalErrors += e
		totalWarnings += w
		ui.Pause(pauseDur)
	}

	return allGroups, totalErrors, totalWarnings
}
