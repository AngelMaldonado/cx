package cmd

import (
	"fmt"

	"github.com/amald/cx/internal/brainstorm"
	"github.com/amald/cx/internal/project"
	"github.com/amald/cx/internal/ui"
	"github.com/spf13/cobra"
)

var decomposeCmd = &cobra.Command{
	Use:   "decompose <name>",
	Short: "Transform a masterfile into a change structure and archive it",
	Args:  cobra.ExactArgs(1),
	RunE:  runDecompose,
}

func runDecompose(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx decompose must be run inside a git repo")
		return errExitCode1
	}

	name := args[0]
	result, err := brainstorm.Decompose(rootDir, name)
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}

	ui.PrintSuccess(fmt.Sprintf("created docs/changes/%s/", name))
	ui.PrintMuted("  proposal.md")
	ui.PrintMuted("  design.md")
	ui.PrintMuted("  tasks.md")
	ui.PrintSuccess(fmt.Sprintf("archived masterfile → %s", result.ArchivePath))
	return nil
}
