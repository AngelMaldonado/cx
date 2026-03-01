package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/amald/cx/internal/project"
	"github.com/amald/cx/internal/ui"
	"github.com/spf13/cobra"
)

var projectsCmd = &cobra.Command{
	Use:   "projects",
	Short: "List all cx-initialized projects",
	RunE:  runProjects,
}

var projectsRemoveCmd = &cobra.Command{
	Use:   "remove <path>",
	Short: "Remove a project from the global registry",
	Args:  cobra.ExactArgs(1),
	RunE:  runProjectsRemove,
}

func init() {
	projectsCmd.AddCommand(projectsRemoveCmd)
}

func runProjects(cmd *cobra.Command, args []string) error {
	reg, err := project.LoadRegistry()
	if err != nil {
		ui.PrintError(fmt.Sprintf("loading registry: %v", err))
		return err
	}

	if len(reg.Projects) == 0 {
		ui.PrintMuted("no projects registered — run cx init in a project")
		return nil
	}

	for _, p := range reg.Projects {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			ui.PrintMuted(fmt.Sprintf("%s (not found)", p))
		} else {
			fmt.Printf("    %s\n", p)
		}
	}
	return nil
}

func runProjectsRemove(cmd *cobra.Command, args []string) error {
	absPath, err := filepath.Abs(args[0])
	if err != nil {
		ui.PrintError(fmt.Sprintf("resolving path: %v", err))
		return err
	}

	removed, err := project.RemoveProject(absPath)
	if err != nil {
		ui.PrintError(fmt.Sprintf("removing project: %v", err))
		return err
	}

	if !removed {
		ui.PrintWarning(fmt.Sprintf("%s not found in registry", absPath))
		return nil
	}

	ui.PrintSuccess(fmt.Sprintf("removed %s", absPath))
	return nil
}
