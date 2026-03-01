package project

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

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
