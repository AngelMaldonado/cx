package cmd

import (
	"fmt"

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

func runDoctor(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx doctor must be run inside a git repo")
		return fmt.Errorf("not a git repository")
	}

	fmt.Println()
	ui.PrintHeader("cx doctor")
	ui.PrintDivider()

	groups := doctor.RunAllChecks(rootDir)
	errors, warnings := doctor.PrintReport(groups)

	fmt.Println()
	ui.PrintDivider()
	ui.PrintSummary(errors, warnings)

	if !fixFlag {
		fixable := doctor.CollectFixable(groups)
		if len(fixable) > 0 {
			fmt.Println()
			ui.PrintMuted(fmt.Sprintf("  %d fixable issue(s) — run cx doctor --fix to repair", len(fixable)))
		}
		if errors > 0 {
			return errExitCode1
		}
		return nil
	}

	// --fix mode
	fixable := doctor.CollectFixable(groups)
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

	fmt.Println()
	errs := doctor.ApplyFixes(fixable)
	for _, e := range errs {
		ui.PrintError(fmt.Sprintf("fix failed: %v", e))
	}

	// Re-check after fixes
	fmt.Println()
	ui.PrintHeader("re-checking")
	ui.PrintDivider()
	groups = doctor.RunAllChecks(rootDir)
	errors, warnings = doctor.PrintReport(groups)
	fmt.Println()
	ui.PrintDivider()
	ui.PrintSummary(errors, warnings)

	if errors > 0 {
		return errExitCode1
	}
	return nil
}
