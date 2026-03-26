package cmd

import (
	"fmt"

	"github.com/AngelMaldonado/cx/internal/agents"
	"github.com/AngelMaldonado/cx/internal/project"
	"github.com/spf13/cobra"
)

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Suspend cx across all projects",
	RunE:  runDisable,
}

func runDisable(cmd *cobra.Command, args []string) error {
	if project.IsDisabled() {
		fmt.Println("cx is already disabled.")
		return nil
	}

	registry, err := project.LoadRegistry()
	if err != nil {
		return err
	}

	cleaned := 0
	restored := 0
	for _, path := range registry.Projects {
		installedAgents := agents.DetectInstalled(path)
		for _, agent := range installedAgents {
			// Remove all cx-managed files for this agent.
			if err := project.RemoveCXManagedFiles(path, agent); err != nil {
				return err
			}
			cleaned++
		}

		if len(installedAgents) == 0 {
			continue
		}

		// Restore the pre-init snapshot if one exists.
		ok, err := project.RestorePreInitState(path, installedAgents)
		if err != nil {
			return err
		}
		if ok {
			restored++
		}
	}

	if err := project.SetDisabled(""); err != nil {
		return err
	}

	fmt.Printf("cx disabled. %d agent(s) cleaned up", cleaned)
	if restored > 0 {
		fmt.Printf(", %d project(s) restored to pre-cx state", restored)
	}
	fmt.Println(".")
	return nil
}
