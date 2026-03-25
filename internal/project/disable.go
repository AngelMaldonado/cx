package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
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

