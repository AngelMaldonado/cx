package cmd

import (
	"fmt"
	"strings"

	"github.com/amald/cx/internal/brainstorm"
	"github.com/amald/cx/internal/project"
	"github.com/amald/cx/internal/ui"
	"github.com/spf13/cobra"
)

var brainstormCmd = &cobra.Command{
	Use:   "brainstorm",
	Short: "Manage masterfiles for planning and ideation",
}

var brainstormNewCmd = &cobra.Command{
	Use:   "new <name>",
	Short: "Create a new masterfile for brainstorming",
	Args:  cobra.ExactArgs(1),
	RunE:  runBrainstormNew,
}

var brainstormStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "List active masterfiles with completion state",
	Args:  cobra.NoArgs,
	RunE:  runBrainstormStatus,
}

func init() {
	brainstormCmd.AddCommand(brainstormNewCmd)
	brainstormCmd.AddCommand(brainstormStatusCmd)
}

func runBrainstormNew(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx brainstorm must be run inside a git repo")
		return errExitCode1
	}

	name := args[0]
	path, err := brainstorm.Create(rootDir, name)
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}

	ui.PrintSuccess(fmt.Sprintf("created %s", path))
	return nil
}

func runBrainstormStatus(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx brainstorm must be run inside a git repo")
		return errExitCode1
	}

	masterfiles, err := brainstorm.List(rootDir)
	if err != nil {
		ui.PrintError(fmt.Sprintf("listing masterfiles: %v", err))
		return errExitCode1
	}

	if len(masterfiles) == 0 {
		ui.PrintMuted("no active masterfiles")
		return nil
	}

	for _, m := range masterfiles {
		status := "○ template"
		if m.Modified {
			status = strings.Replace(ui.SymbolSuccess, "", "✓", 1)
			status = ui.SymbolSuccess + " modified"
		}
		fmt.Printf("    %s  %s\n", status, m.Name)
	}

	return nil
}
