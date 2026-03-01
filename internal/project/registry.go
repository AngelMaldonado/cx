package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type ProjectRegistry struct {
	Projects []string `json:"projects"`
}

type Preferences struct {
	AutoUpdateCheck bool      `json:"auto_update_check"`
	LastUpdateCheck time.Time `json:"last_update_check,omitempty"`
}

func GlobalCXDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".cx")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating ~/.cx: %w", err)
	}
	return dir, nil
}

func registryPath() (string, error) {
	dir, err := GlobalCXDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "projects.json"), nil
}

func preferencesPath() (string, error) {
	dir, err := GlobalCXDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "preferences.json"), nil
}

func LoadRegistry() (*ProjectRegistry, error) {
	path, err := registryPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &ProjectRegistry{}, nil
		}
		return nil, err
	}
	var reg ProjectRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parsing projects.json: %w", err)
	}
	return &reg, nil
}

func SaveRegistry(reg *ProjectRegistry) error {
	path, err := registryPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, data)
}

func RegisterProject(absPath string) (bool, error) {
	reg, err := LoadRegistry()
	if err != nil {
		return false, err
	}
	for _, p := range reg.Projects {
		if p == absPath {
			return false, nil // already registered
		}
	}
	reg.Projects = append(reg.Projects, absPath)
	if err := SaveRegistry(reg); err != nil {
		return false, err
	}
	return true, nil
}

func IsFirstEverInit() bool {
	path, err := registryPath()
	if err != nil {
		return true
	}
	_, err = os.Stat(path)
	return os.IsNotExist(err)
}

func LoadPreferences() (*Preferences, error) {
	path, err := preferencesPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Preferences{}, nil
		}
		return nil, err
	}
	var prefs Preferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, fmt.Errorf("parsing preferences.json: %w", err)
	}
	return &prefs, nil
}

func SavePreferences(prefs *Preferences) error {
	path, err := preferencesPath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	return atomicWrite(path, data)
}
