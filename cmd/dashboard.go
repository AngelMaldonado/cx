package cmd

import (
	"github.com/amald/cx/internal/project"
	"github.com/amald/cx/internal/tui"
	"github.com/amald/cx/internal/ui"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Short:   "Open the interactive TUI dashboard",
	Long:    "Launch an interactive terminal dashboard showing memories, sessions, agent runs, and sync status.",
	Aliases: []string{"dash", "ui"},
	RunE:    runDashboard,
}

func runDashboard(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx dashboard must be run inside a git repo")
		return errExitCode1
	}

	return tui.Run(rootDir)
}
