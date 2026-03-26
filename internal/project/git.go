package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// WorktreeInfo holds metadata about a git worktree.
type WorktreeInfo struct {
	Path   string
	Branch string
	Head   string // commit SHA
}

// CreateWorktree creates a new git worktree under .cx/worktrees/<branchName>/.
// It creates a new branch from the current HEAD.
// Returns the absolute path to the created worktree.
func CreateWorktree(projectPath, branchName string) (string, error) {
	worktreePath := filepath.Join(projectPath, ".cx", "worktrees", branchName)

	if _, err := os.Stat(worktreePath); err == nil {
		return "", fmt.Errorf("worktree already exists: %s", worktreePath)
	}

	cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", branchName)
	cmd.Dir = projectPath
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("creating worktree: %w: %s", err, strings.TrimSpace(string(out)))
	}

	abs, err := filepath.Abs(worktreePath)
	if err != nil {
		return "", fmt.Errorf("resolving worktree path: %w", err)
	}
	return abs, nil
}

// RemoveWorktree removes a git worktree and its branch.
func RemoveWorktree(projectPath, branchName string) error {
	worktreePath := filepath.Join(".cx", "worktrees", branchName)

	rmCmd := exec.Command("git", "worktree", "remove", worktreePath, "--force")
	rmCmd.Dir = projectPath
	if out, err := rmCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("removing worktree: %w: %s", err, strings.TrimSpace(string(out)))
	}

	// Clean up the branch; non-fatal if it doesn't exist.
	branchCmd := exec.Command("git", "branch", "-D", branchName)
	branchCmd.Dir = projectPath
	branchCmd.CombinedOutput() //nolint:errcheck — intentionally non-fatal

	return nil
}

// ListWorktrees returns all worktrees under .cx/worktrees/.
func ListWorktrees(projectPath string) ([]WorktreeInfo, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = projectPath
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing worktrees: %w", err)
	}

	prefix := filepath.Join(projectPath, ".cx", "worktrees")

	var result []WorktreeInfo
	var current WorktreeInfo

	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			current = WorktreeInfo{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "HEAD "):
			current.Head = strings.TrimPrefix(line, "HEAD ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			current.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "":
			if current.Path != "" && strings.HasPrefix(current.Path, prefix) {
				result = append(result, current)
			}
			current = WorktreeInfo{}
		}
	}

	return result, nil
}

// CleanupWorktrees removes all worktrees whose branch name starts with the given prefix.
func CleanupWorktrees(projectPath, prefix string) error {
	worktrees, err := ListWorktrees(projectPath)
	if err != nil {
		return fmt.Errorf("listing worktrees for cleanup: %w", err)
	}

	var errs []string
	for _, wt := range worktrees {
		if strings.HasPrefix(wt.Branch, prefix) {
			if err := RemoveWorktree(projectPath, wt.Branch); err != nil {
				errs = append(errs, err.Error())
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("cleanup errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

func IsGitRepo() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("not a git repository")
	}
	return strings.TrimSpace(string(out)), nil
}

func GitUserName() (string, error) {
	out, err := exec.Command("git", "config", "user.name").Output()
	if err != nil {
		return "", fmt.Errorf("git user.name not configured")
	}
	return strings.TrimSpace(string(out)), nil
}

func HooksDir(rootDir string) string {
	return filepath.Join(rootDir, ".git", "hooks")
}

func InstallHook(rootDir, hookType string, force bool) (existsAlready bool, err error) {
	dir := HooksDir(rootDir)
	hookPath := filepath.Join(dir, hookType)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return false, fmt.Errorf("creating hooks dir: %w", err)
	}

	if _, err := os.Stat(hookPath); err == nil {
		hasCX, checkErr := HookContainsCX(rootDir, hookType)
		if checkErr != nil {
			return false, checkErr
		}
		if !hasCX && !force {
			return true, nil
		}
	}

	script := hookScript(hookType)

	tmp := hookPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(script), 0o755); err != nil {
		return false, fmt.Errorf("writing hook: %w", err)
	}
	if err := os.Rename(tmp, hookPath); err != nil {
		return false, fmt.Errorf("installing hook: %w", err)
	}
	return false, nil
}

func HookContainsCX(rootDir, hookType string) (bool, error) {
	hookPath := filepath.Join(HooksDir(rootDir), hookType)
	data, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return strings.Contains(string(data), "# CX:"), nil
}

func hookScript(hookType string) string {
	return fmt.Sprintf(`#!/bin/sh
# CX: auto-installed by cx init
# Hook type: %s

# Placeholder for CX hook integration
# Future: trigger cx doctor, memory indexing, etc.
`, hookType)
}
