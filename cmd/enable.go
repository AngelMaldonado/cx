package cmd

import (
	"fmt"
	"os"

	"github.com/AngelMaldonado/cx/internal/project"
	"github.com/spf13/cobra"
)

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Restore cx across all projects",
	RunE:  runEnable,
}

func runEnable(cmd *cobra.Command, args []string) error {
	if !project.IsDisabled() {
		fmt.Println("cx is already enabled.")
		return nil
	}

	restored, warnings, err := project.RestoreAgentConfigs()
	if err != nil {
		// Don't clear sentinel — restore failed, system still disabled.
		return fmt.Errorf("restore failed, cx remains disabled: %w", err)
	}

	if err := project.ClearDisabled(); err != nil {
		return err
	}

	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, w)
	}

	for _, r := range restored {
		fmt.Printf("  restored: %s/%s\n", r.ProjectPath, r.ConfigFile)
	}

	fmt.Printf("cx enabled. %d config file(s) restored.\n", len(restored))
	return nil
}
