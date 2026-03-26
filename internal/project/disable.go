package project

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/AngelMaldonado/cx/internal/agents"
)

// disabledPayload is the JSON structure written to ~/.cx/disabled.
type disabledPayload struct {
	DisabledAt time.Time `json:"disabled_at"`
	Reason     string    `json:"reason"`
}

// RestoredFile describes a single successfully restored agent config.
type RestoredFile struct {
	ProjectPath string
	ConfigFile  string
}

// sentinelPath returns the path to the disabled sentinel file.
func sentinelPath() (string, error) {
	dir, err := GlobalCXDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "disabled"), nil
}

// backupDirPath returns the path to the agent-backups directory.
func backupDirPath() (string, error) {
	dir, err := GlobalCXDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "agent-backups"), nil
}

// IsDisabled returns true if ~/.cx/disabled exists.
func IsDisabled() bool {
	p, err := sentinelPath()
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// SetDisabled creates ~/.cx/disabled with a JSON timestamp payload.
// reason may be empty.
func SetDisabled(reason string) error {
	p, err := sentinelPath()
	if err != nil {
		return err
	}
	payload := disabledPayload{
		DisabledAt: time.Now().UTC(),
		Reason:     reason,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshalling disabled payload: %w", err)
	}
	return os.WriteFile(p, data, 0o644)
}

// ClearDisabled removes ~/.cx/disabled. No-op if it does not exist.
func ClearDisabled() error {
	p, err := sentinelPath()
	if err != nil {
		return err
	}
	err = os.Remove(p)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// BackupAgentConfig writes content to ~/.cx/agent-backups/<projectID>-<configFile>.
// projectPath is the absolute path to the project root.
// configFile is the bare filename (e.g., "CLAUDE.md").
func BackupAgentConfig(projectPath, configFile, content string) error {
	bd, err := backupDirPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(bd, 0o755); err != nil {
		return fmt.Errorf("creating agent-backups dir: %w", err)
	}
	name := ProjectID(projectPath) + "-" + configFile
	dest := filepath.Join(bd, name)
	// If backup already exists, don't overwrite (protects against partial-failure retry).
	if _, err := os.Stat(dest); err == nil {
		return nil
	}
	return os.WriteFile(dest, []byte(content), 0o644)
}

// RestoreAgentConfigs reads all files in ~/.cx/agent-backups/, resolves each
// backup to its target project path via LoadRegistry(), restores the content,
// and deletes the backup on success.
// Returns the list of successfully restored files and any non-fatal warnings.
func RestoreAgentConfigs() ([]RestoredFile, []string, error) {
	bd, err := backupDirPath()
	if err != nil {
		return nil, nil, err
	}

	entries, err := os.ReadDir(bd)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("reading agent-backups dir: %w", err)
	}

	reg, err := LoadRegistry()
	if err != nil {
		return nil, nil, fmt.Errorf("loading registry: %w", err)
	}

	var restored []RestoredFile
	var warnings []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		// filename format: <12-char-id>-<configFile>
		if len(filename) <= 13 {
			warnings = append(warnings, fmt.Sprintf("warning: unexpected backup filename %q; skipping", filename))
			continue
		}
		id := filename[:12]
		if filename[12] != '-' {
			warnings = append(warnings, fmt.Sprintf("warning: unexpected backup filename %q; skipping", filename))
			continue
		}
		configFile := filename[13:]

		// Find a registered project whose ID matches.
		var matchedPath string
		for _, p := range reg.Projects {
			if ProjectID(p) == id {
				matchedPath = p
				break
			}
		}
		if matchedPath == "" {
			warnings = append(warnings, fmt.Sprintf("warning: no registered project matches backup %s; backup retained", filename))
			continue
		}

		backupFile := filepath.Join(bd, filename)
		content, err := os.ReadFile(backupFile)
		if err != nil {
			return restored, warnings, fmt.Errorf("reading backup %s: %w", filename, err)
		}

		dest := filepath.Join(matchedPath, configFile)
		// Write atomically via temp file in the same directory.
		dir := filepath.Dir(dest)
		tmp, err := os.CreateTemp(dir, ".cx-restore-*")
		if err != nil {
			// Non-fatal: project directory may have been deleted; skip and continue.
			warnings = append(warnings, fmt.Sprintf("warning: skipping restore of %s (project dir unavailable): %v", filename, err))
			continue
		}
		tmpName := tmp.Name()
		_, writeErr := tmp.Write(content)
		closeErr := tmp.Close()
		if writeErr != nil || closeErr != nil {
			_ = os.Remove(tmpName)
			if writeErr != nil {
				return restored, warnings, fmt.Errorf("writing temp restore file: %w", writeErr)
			}
			return restored, warnings, fmt.Errorf("closing temp restore file: %w", closeErr)
		}
		if err := os.Rename(tmpName, dest); err != nil {
			_ = os.Remove(tmpName)
			return restored, warnings, fmt.Errorf("renaming restore file to %s: %w", dest, err)
		}

		// Delete the backup on success.
		if err := os.Remove(backupFile); err != nil {
			// Non-fatal: warn but continue.
			warnings = append(warnings, fmt.Sprintf("warning: could not delete backup %s: %v", filename, err))
		}

		restored = append(restored, RestoredFile{
			ProjectPath: matchedPath,
			ConfigFile:  configFile,
		})
	}

	return restored, warnings, nil
}

// preInitBackupDir returns the path to the pre-init backup directory for the given project.
// Layout: ~/.cx/agent-backups/<projectID>/pre-init/
func preInitBackupDir(projectPath string) (string, error) {
	bd, err := backupDirPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(bd, ProjectID(projectPath), "pre-init"), nil
}

// BackupPreInitState snapshots the agent root directories (e.g., .claude/) that
// already exist in the project BEFORE cx writes any files. This captures the
// true pre-cx baseline so disable can restore it later.
//
// If a pre-init backup already exists for this project, the function is a no-op
// to protect the original baseline from being overwritten on subsequent inits.
func BackupPreInitState(projectPath string, agentList []agents.Agent) error {
	destBase, err := preInitBackupDir(projectPath)
	if err != nil {
		return err
	}

	// If the pre-init backup already exists, skip — the first init wins.
	if _, err := os.Stat(destBase); err == nil {
		return nil
	}

	// Check whether any agent roots actually exist before creating the dir.
	anyExist := false
	for _, agent := range agentList {
		agentRoot := filepath.Join(projectPath, agent.Dir)
		if _, err := os.Stat(agentRoot); err == nil {
			anyExist = true
			break
		}
	}
	if !anyExist {
		// Nothing to back up — project was initialized fresh.
		return nil
	}

	if err := os.MkdirAll(destBase, 0o755); err != nil {
		return fmt.Errorf("creating pre-init backup dir: %w", err)
	}

	for _, agent := range agentList {
		agentRoot := filepath.Join(projectPath, agent.Dir)
		if _, err := os.Stat(agentRoot); err != nil {
			// This agent's root doesn't exist; nothing to back up.
			continue
		}
		destAgentRoot := filepath.Join(destBase, agent.Dir)
		if err := copyDirTree(agentRoot, destAgentRoot); err != nil {
			return fmt.Errorf("backing up %s: %w", agent.Dir, err)
		}
	}

	return nil
}

// RestorePreInitState copies the pre-init backup back into the project directory,
// overwriting any current contents. Returns (true, nil) if a backup was found and
// restored, (false, nil) if no backup existed (clean-init case).
func RestorePreInitState(projectPath string, agentList []agents.Agent) (bool, error) {
	srcBase, err := preInitBackupDir(projectPath)
	if err != nil {
		return false, err
	}

	if _, err := os.Stat(srcBase); os.IsNotExist(err) {
		return false, nil
	}

	for _, agent := range agentList {
		srcAgentRoot := filepath.Join(srcBase, agent.Dir)
		if _, err := os.Stat(srcAgentRoot); err != nil {
			// No backup for this agent root — nothing to restore.
			continue
		}
		destAgentRoot := filepath.Join(projectPath, agent.Dir)
		if err := copyDirTree(srcAgentRoot, destAgentRoot); err != nil {
			return false, fmt.Errorf("restoring %s: %w", agent.Dir, err)
		}
	}

	return true, nil
}

// RemoveCXManagedFiles deletes the files that cx wrote for the given agent:
// the config file, all skill files in the skills dir, and all subagent files
// in the agents dir. It does NOT remove the directories themselves so that
// any user files placed alongside cx files are preserved.
func RemoveCXManagedFiles(projectPath string, agent agents.Agent) error {
	// Remove the config file (e.g., CLAUDE.md).
	configPath := filepath.Join(projectPath, agent.ConfigFile)
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("removing %s: %w", agent.ConfigFile, err)
	}

	// Remove all skill subdirectories inside SkillsDir that were written by cx.
	// cx writes <slug>/SKILL.md; we remove the whole slug directory.
	skillsDir := filepath.Join(projectPath, agent.SkillsDir)
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			skillMD := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
			if _, err := os.Stat(skillMD); err == nil {
				if err := os.RemoveAll(filepath.Join(skillsDir, entry.Name())); err != nil {
					return fmt.Errorf("removing skill dir %s: %w", entry.Name(), err)
				}
			}
		}
	}

	// Remove all subagent files in the agents dir.
	if agent.AgentsDir != "" {
		agentsDir := filepath.Join(projectPath, agent.AgentsDir)
		if entries, err := os.ReadDir(agentsDir); err == nil {
			for _, entry := range entries {
				if entry.IsDir() {
					continue
				}
				if err := os.Remove(filepath.Join(agentsDir, entry.Name())); err != nil && !os.IsNotExist(err) {
					return fmt.Errorf("removing subagent file %s: %w", entry.Name(), err)
				}
			}
		}
	}

	return nil
}

// copyDirTree recursively copies the directory tree rooted at src to dst.
// dst is created if it does not exist. Existing files at dst are overwritten.
func copyDirTree(src, dst string) error {
	srcFS := os.DirFS(src)
	return fs.WalkDir(srcFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, path)
		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}
		// Open the source file via the srcFS to stay rooted at src.
		srcFile, err := srcFS.Open(path)
		if err != nil {
			return fmt.Errorf("opening %s: %w", path, err)
		}
		defer srcFile.Close()

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return err
		}

		// Write atomically via a temp file.
		tmp := destPath + ".tmp"
		out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			return fmt.Errorf("creating temp file %s: %w", tmp, err)
		}
		_, copyErr := io.Copy(out, srcFile)
		closeErr := out.Close()
		if copyErr != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("copying to %s: %w", tmp, copyErr)
		}
		if closeErr != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("closing temp file %s: %w", tmp, closeErr)
		}
		if err := os.Rename(tmp, destPath); err != nil {
			_ = os.Remove(tmp)
			return fmt.Errorf("renaming %s to %s: %w", tmp, destPath, err)
		}
		return nil
	})
}
