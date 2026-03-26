package cmd

import (
	"fmt"

	"github.com/AngelMaldonado/cx/internal/agents"
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

	registry, err := project.LoadRegistry()
	if err != nil {
		return err
	}

	synced := 0
	var syncErrors []string
	for _, path := range registry.Projects {
		installedAgents := agents.DetectInstalled(path)
		for _, agent := range installedAgents {
			if err := agents.EnsureAgentDir(path, agent); err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("  %s: creating dirs: %v", agent.Name, err))
				continue
			}
			if err := agents.WriteConfigFile(path, agent); err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("  %s: writing config: %v", agent.Name, err))
				continue
			}
			if _, err := agents.WriteSkills(path, agent); err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("  %s: writing skills: %v", agent.Name, err))
				continue
			}
			if _, err := agents.WriteSubagents(path, agent); err != nil {
				syncErrors = append(syncErrors, fmt.Sprintf("  %s: writing subagents: %v", agent.Name, err))
				continue
			}
			synced++
		}
	}

	// Clear the disabled sentinel even if there were partial errors — the
	// sentinel controls the global lock, not individual project state.
	if err := project.ClearDisabled(); err != nil {
		return err
	}

	for _, e := range syncErrors {
		fmt.Println(e)
	}

	fmt.Printf("cx enabled. %d agent config(s) re-applied.\n", synced)
	return nil
}
