package cmd

import (
	"fmt"
	"strings"

	"github.com/amald/cx/internal/change"
	"github.com/amald/cx/internal/project"
	"github.com/amald/cx/internal/ui"
	"github.com/amald/cx/internal/verify"
	"github.com/spf13/cobra"
)

var changeCmd = &cobra.Command{
	Use:   "change",
	Short: "Manage structured changes",
}

var changeNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new change with template files",
	Args:  cobra.ExactArgs(1),
	RunE:  runChangeNew,
}

var changeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "List active changes with completion state",
	Args:  cobra.NoArgs,
	RunE:  runChangeStatus,
}

var changeArchiveCmd = &cobra.Command{
	Use:   "archive <name>",
	Short: "Archive a completed change",
	Args:  cobra.ExactArgs(1),
	RunE:  runChangeArchive,
}

var changeVerifyCmd = &cobra.Command{
	Use:   "verify <name>",
	Short: "Scaffold verification prompt for a change",
	Args:  cobra.ExactArgs(1),
	RunE:  runChangeVerify,
}

var changeSpecSyncCmd = &cobra.Command{
	Use:   "spec-sync <name>",
	Short: "Merge delta specs into canonical specs without archiving",
	Args:  cobra.ExactArgs(1),
	RunE:  runChangeSpecSync,
}

var skipSpecsFlag bool

func init() {
	changeCmd.AddCommand(changeNewCmd)
	changeCmd.AddCommand(changeStatusCmd)
	changeCmd.AddCommand(changeArchiveCmd)
	changeCmd.AddCommand(changeVerifyCmd)
	changeCmd.AddCommand(changeSpecSyncCmd)
	changeArchiveCmd.Flags().BoolVar(&skipSpecsFlag, "skip-specs", false, "Skip spec verification (for non-behavioral changes)")
}

func runChangeNew(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx change must be run inside a git repo")
		return errExitCode1
	}

	name := args[0]
	if err := change.Create(rootDir, name); err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}

	ui.PrintSuccess(fmt.Sprintf("created docs/changes/%s/", name))
	ui.PrintMuted("  proposal.md")
	ui.PrintMuted("  design.md")
	ui.PrintMuted("  tasks.md")
	ui.PrintMuted("  specs/")
	return nil
}

func runChangeStatus(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx change must be run inside a git repo")
		return errExitCode1
	}

	changes, err := change.ListChanges(rootDir)
	if err != nil {
		ui.PrintError(fmt.Sprintf("listing changes: %v", err))
		return errExitCode1
	}

	if len(changes) == 0 {
		ui.PrintMuted("no active changes")
		return nil
	}

	for _, c := range changes {
		fmt.Println()
		ui.PrintHeader(c.Name)

		proposal := fileSymbol(c.HasProposal)
		design := fileSymbol(c.HasDesign)
		tasks := fileSymbol(c.HasTasks)
		fmt.Printf("    %s proposal  %s design  %s tasks\n", proposal, design, tasks)
		fmt.Printf("    verify: %s\n", c.VerifyStatus)

		if len(c.DeltaSpecs) > 0 {
			var specLabels []string
			syncedSet := make(map[string]bool)
			for _, s := range c.SyncedDeltas {
				syncedSet[s] = true
			}
			for _, d := range c.DeltaSpecs {
				if syncedSet[d] {
					specLabels = append(specLabels, d+" [synced]")
				} else {
					specLabels = append(specLabels, d)
				}
			}
			ui.PrintMuted(fmt.Sprintf("Delta specs: %s", strings.Join(specLabels, ", ")))
			// Count unsynced deltas
			unsyncedCount := len(c.DeltaSpecs) - len(c.SyncedDeltas)
			if unsyncedCount > 0 {
				ui.PrintWarning(fmt.Sprintf("  %d delta spec(s) pending merge", unsyncedCount))
			}
		} else {
			ui.PrintMuted("Delta specs: (none)")
		}

		var missing []string
		if !c.HasProposal {
			missing = append(missing, "proposal.md")
		}
		if !c.HasDesign {
			missing = append(missing, "design.md")
		}
		if !c.HasTasks {
			missing = append(missing, "tasks.md")
		}

		if len(missing) == 0 {
			ui.PrintSuccess("Ready to archive")
		} else {
			ui.PrintWarning(fmt.Sprintf("Missing: %s", strings.Join(missing, ", ")))
		}
	}
	fmt.Println()

	return nil
}

func runChangeArchive(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx change must be run inside a git repo")
		return errExitCode1
	}

	name := args[0]
	opts := change.ArchiveOptions{SkipSpecs: skipSpecsFlag}
	result, err := change.Archive(rootDir, name, opts)
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}

	ui.PrintSuccess(fmt.Sprintf("archived %s → %s/", name, result.ArchivePath))
	if len(result.BootstrappedSpecs) > 0 {
		ui.PrintMuted(fmt.Sprintf("  Bootstrapped new spec areas: %s", strings.Join(result.BootstrappedSpecs, ", ")))
	}
	if skipSpecsFlag {
		ui.PrintMuted("  skipped spec verification (--skip-specs)")
	}
	if len(result.DeltaSpecs) > 0 && !skipSpecsFlag {
		fmt.Println()
		ui.PrintWarning(fmt.Sprintf("NEXT: Delta specs detected in: %s", strings.Join(result.DeltaSpecs, ", ")))
		ui.PrintMuted("  → Dispatch Planner in archive mode to merge into canonical specs")
		ui.PrintMuted(fmt.Sprintf("  → Archived change at: %s/", result.ArchivePath))
	}
	return nil
}

func runChangeVerify(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx change must be run inside a git repo")
		return errExitCode1
	}

	name := args[0]
	prompt, err := verify.BuildPrompt(rootDir, name)
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}

	fmt.Print(prompt)

	if err := verify.Record(rootDir, name); err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}

	ui.PrintMuted(fmt.Sprintf("verify.md created at docs/changes/%s/verify.md", name))
	return nil
}

func runChangeSpecSync(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx change must be run inside a git repo")
		return errExitCode1
	}

	name := args[0]
	result, err := change.SpecSync(rootDir, name)
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}

	fmt.Print(result.Prompt)
	ui.PrintSuccess(fmt.Sprintf("spec-sync ready — %d unsynced delta(s): %s", len(result.Areas), strings.Join(result.Areas, ", ")))
	return nil
}

func fileSymbol(filled bool) string {
	if filled {
		return ui.SymbolSuccess
	}
	return "○"
}
