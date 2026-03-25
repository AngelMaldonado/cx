package project

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestIsDisabled verifies that IsDisabled returns true when the sentinel file
// exists and false when it does not.
func TestIsDisabled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if IsDisabled() {
		t.Fatal("expected IsDisabled() == false before sentinel created")
	}

	// Manually create the sentinel file.
	dir, err := GlobalCXDir()
	if err != nil {
		t.Fatalf("GlobalCXDir: %v", err)
	}
	sentinel := filepath.Join(dir, "disabled")
	if err := os.WriteFile(sentinel, []byte("{}"), 0o644); err != nil {
		t.Fatalf("writing sentinel: %v", err)
	}

	if !IsDisabled() {
		t.Fatal("expected IsDisabled() == true after sentinel created")
	}

	if err := os.Remove(sentinel); err != nil {
		t.Fatalf("removing sentinel: %v", err)
	}

	if IsDisabled() {
		t.Fatal("expected IsDisabled() == false after sentinel removed")
	}
}

// TestSetClearDisabled verifies the round-trip of SetDisabled and ClearDisabled.
func TestSetClearDisabled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if err := SetDisabled(""); err != nil {
		t.Fatalf("SetDisabled: %v", err)
	}

	if !IsDisabled() {
		t.Fatal("expected IsDisabled() == true after SetDisabled")
	}

	// Verify the file contains the expected JSON key.
	dir, err := GlobalCXDir()
	if err != nil {
		t.Fatalf("GlobalCXDir: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "disabled"))
	if err != nil {
		t.Fatalf("reading sentinel: %v", err)
	}
	if !strings.Contains(string(data), "disabled_at") {
		t.Fatalf("sentinel content missing 'disabled_at': %s", data)
	}

	if err := ClearDisabled(); err != nil {
		t.Fatalf("ClearDisabled: %v", err)
	}

	if IsDisabled() {
		t.Fatal("expected IsDisabled() == false after ClearDisabled")
	}

	// Calling ClearDisabled again should be a no-op (not an error).
	if err := ClearDisabled(); err != nil {
		t.Fatalf("ClearDisabled (second call): %v", err)
	}
}

// TestBackupAgentConfig verifies that BackupAgentConfig writes the backup file
// at the expected path with the expected content.
func TestBackupAgentConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	fakeProjectPath := "/tmp/fake-project-abc"
	configFile := "CLAUDE.md"
	content := "original content"

	if err := BackupAgentConfig(fakeProjectPath, configFile, content); err != nil {
		t.Fatalf("BackupAgentConfig: %v", err)
	}

	dir, err := GlobalCXDir()
	if err != nil {
		t.Fatalf("GlobalCXDir: %v", err)
	}
	expectedName := ProjectID(fakeProjectPath) + "-" + configFile
	backupPath := filepath.Join(dir, "agent-backups", expectedName)

	data, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("reading backup file: %v", err)
	}
	if string(data) != content {
		t.Fatalf("backup content = %q, want %q", string(data), content)
	}
}

// TestRestoreAgentConfigs verifies the full backup-and-restore cycle.
func TestRestoreAgentConfigs(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	// Create a temp project directory with a config file.
	projectDir := t.TempDir()
	configFile := "CLAUDE.md"
	originalContent := "# original claude config"

	configPath := filepath.Join(projectDir, configFile)
	if err := os.WriteFile(configPath, []byte("stub content"), 0o644); err != nil {
		t.Fatalf("writing placeholder config: %v", err)
	}

	// Register the project so LoadRegistry can find it.
	if _, err := RegisterProject(projectDir); err != nil {
		t.Fatalf("RegisterProject: %v", err)
	}

	// Write a backup as if cx disable had run.
	if err := BackupAgentConfig(projectDir, configFile, originalContent); err != nil {
		t.Fatalf("BackupAgentConfig: %v", err)
	}

	// Verify the backup file exists before restore.
	dir, err := GlobalCXDir()
	if err != nil {
		t.Fatalf("GlobalCXDir: %v", err)
	}
	backupName := ProjectID(projectDir) + "-" + configFile
	backupPath := filepath.Join(dir, "agent-backups", backupName)
	if _, err := os.Stat(backupPath); err != nil {
		t.Fatalf("backup file missing before restore: %v", err)
	}

	restored, warnings, err := RestoreAgentConfigs()
	if err != nil {
		t.Fatalf("RestoreAgentConfigs: %v", err)
	}

	for _, w := range warnings {
		t.Logf("warning: %s", w)
	}

	if len(restored) != 1 {
		t.Fatalf("expected 1 restored file, got %d", len(restored))
	}
	if restored[0].ProjectPath != projectDir {
		t.Errorf("restored ProjectPath = %q, want %q", restored[0].ProjectPath, projectDir)
	}
	if restored[0].ConfigFile != configFile {
		t.Errorf("restored ConfigFile = %q, want %q", restored[0].ConfigFile, configFile)
	}

	// Verify the config file was restored with original content.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading restored config: %v", err)
	}
	if string(data) != originalContent {
		t.Fatalf("restored content = %q, want %q", string(data), originalContent)
	}

	// Verify the backup file was deleted.
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Fatalf("expected backup file to be deleted after restore, but stat returned: %v", err)
	}
}
