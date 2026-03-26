package cmd

import (
	"fmt"
	"regexp"

	"github.com/AngelMaldonado/cx/internal/project"
	"github.com/AngelMaldonado/cx/internal/ui"
	"github.com/spf13/cobra"
)

var worktreeCmd = &cobra.Command{
	Use:   "worktree",
	Short: "Manage git worktrees for parallel change development",
}

var worktreeCreateCmd = &cobra.Command{
	Use:   "create <branch-name>",
	Short: "Create a new worktree under .cx/worktrees/<branch-name>",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorktreeCreate,
}

var worktreeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active worktrees under .cx/worktrees/",
	Args:  cobra.NoArgs,
	RunE:  runWorktreeList,
}

var worktreeCleanupCmd = &cobra.Command{
	Use:   "cleanup <change-name>",
	Short: "Remove all worktrees whose branch starts with <change-name>-",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorktreeCleanup,
}

func init() {
	worktreeCmd.AddCommand(worktreeCreateCmd)
	worktreeCmd.AddCommand(worktreeListCmd)
	worktreeCmd.AddCommand(worktreeCleanupCmd)
}

var kebabCase = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func runWorktreeCreate(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx worktree must be run inside a git repo")
		return errExitCode1
	}

	branchName := args[0]
	if !kebabCase.MatchString(branchName) {
		ui.PrintError(fmt.Sprintf("invalid branch name %q — must be kebab-case (e.g. my-feature)", branchName))
		return errExitCode1
	}

	path, err := project.CreateWorktree(rootDir, branchName)
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}

	ui.PrintSuccess(fmt.Sprintf("created worktree at %s", path))
	ui.PrintMuted(fmt.Sprintf("  branch: %s", branchName))
	return nil
}

func runWorktreeList(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx worktree must be run inside a git repo")
		return errExitCode1
	}

	worktrees, err := project.ListWorktrees(rootDir)
	if err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}

	if len(worktrees) == 0 {
		ui.PrintMuted("no active worktrees under .cx/worktrees/")
		return nil
	}

	ui.PrintHeader("active worktrees")
	for _, wt := range worktrees {
		ui.PrintItem(wt.Branch, wt.Path)
		ui.PrintMuted(fmt.Sprintf("      HEAD: %s", wt.Head))
	}
	fmt.Println()
	return nil
}

func runWorktreeCleanup(cmd *cobra.Command, args []string) error {
	rootDir, err := project.IsGitRepo()
	if err != nil {
		ui.PrintError("not a git repository — cx worktree must be run inside a git repo")
		return errExitCode1
	}

	changeName := args[0]
	prefix := changeName + "-"

	if err := project.CleanupWorktrees(rootDir, prefix); err != nil {
		ui.PrintError(err.Error())
		return errExitCode1
	}

	ui.PrintSuccess(fmt.Sprintf("cleaned up worktrees for change %q (prefix: %s)", changeName, prefix))
	return nil
}
