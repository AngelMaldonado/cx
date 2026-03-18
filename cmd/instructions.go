package cmd

import (
	"fmt"

	"github.com/amald/cx/internal/instructions"
	"github.com/amald/cx/internal/project"
	"github.com/amald/cx/internal/ui"
	"github.com/spf13/cobra"
)

var instructionsCmd = &cobra.Command{
	Use:   "instructions <artifact>",
	Short: "Get template, context, and dependency info for a change artifact",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstructions,
}

func runInstructions(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx instructions must be run inside a git repo")
		return errExitCode1
	}

	artifact := args[0]
	result, err := instructions.Build(rootDir, artifact)
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}

	fmt.Print(result)
	return nil
}
