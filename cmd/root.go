package cmd

import (
	"fmt"
	"os"

	"github.com/amald/cx/internal/ui"
	"github.com/spf13/cobra"
)

var Version = "dev"

var errExitCode1 = fmt.Errorf("")

var rootCmd = &cobra.Command{
	Use:           "cx",
	Short:         "CX — AI-native project knowledge system",
	Version:       Version,
	SilenceErrors: true,
	SilenceUsage:  true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		if err != errExitCode1 {
			ui.PrintError(err.Error())
		}
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(doctorCmd)
}
