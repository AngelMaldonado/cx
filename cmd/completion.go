package cmd

import (
	"os"

	"github.com/amald/cx/internal/brainstorm"
	"github.com/amald/cx/internal/project"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for cx.

To load completions in the current shell session:

  bash:
    source <(cx completion bash)

  zsh:
    source <(cx completion zsh)

  fish:
    cx completion fish | source

  PowerShell:
    cx completion powershell | Out-String | Invoke-Expression

To load completions permanently, source the script in your shell profile.
`,
}

var completionBashCmd = &cobra.Command{
	Use:                   "bash",
	Short:                 "Generate bash completion script",
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenBashCompletion(os.Stdout)
	},
}

var completionZshCmd = &cobra.Command{
	Use:                   "zsh",
	Short:                 "Generate zsh completion script",
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenZshCompletion(os.Stdout)
	},
}

var completionFishCmd = &cobra.Command{
	Use:                   "fish",
	Short:                 "Generate fish completion script",
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenFishCompletion(os.Stdout, true)
	},
}

var completionPowershellCmd = &cobra.Command{
	Use:                   "powershell",
	Short:                 "Generate PowerShell completion script",
	DisableFlagsInUseLine: true,
	Args:                  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
	},
}

// masterfileNamesCompletion returns a ValidArgsFunction that completes with
// available masterfile names from docs/masterfiles/.
func masterfileNamesCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	rootDir, err := project.IsGitRepo()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	masterfiles, err := brainstorm.List(rootDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	names := make([]string, 0, len(masterfiles))
	for _, m := range masterfiles {
		names = append(names, m.Name)
	}
	return names, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	completionCmd.AddCommand(completionBashCmd)
	completionCmd.AddCommand(completionZshCmd)
	completionCmd.AddCommand(completionFishCmd)
	completionCmd.AddCommand(completionPowershellCmd)
	// Dynamic completion for decompose: complete with masterfile names.
	decomposeCmd.ValidArgsFunction = masterfileNamesCompletion

	// change new takes a free-form name; no file completion needed.
	changeNewCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}
