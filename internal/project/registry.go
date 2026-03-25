package project

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/AngelMaldonado/cx/internal/memory"
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

func preferencesPath() (string, error) {
	dir, err := GlobalCXDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "preferences.json"), nil
}

// ProjectID returns a short SHA-256-based ID for a path.
func ProjectID(path string) string {
	sum := sha256.Sum256([]byte(path))
	return fmt.Sprintf("%x", sum)[:12]
}

// migrateJSONIfNeeded checks for a legacy projects.json and imports its paths
// into the DB on first use. It is called automatically by DB-backed functions.
func migrateJSONIfNeeded() error {
	dir, err := GlobalCXDir()
	if err != nil {
		return err
	}
	jsonPath := filepath.Join(dir, "projects.json")
	data, err := os.ReadFile(jsonPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading legacy projects.json: %w", err)
	}

	var reg struct {
		Projects []string `json:"projects"`
	}
	if err := json.Unmarshal(data, &reg); err != nil {
		return fmt.Errorf("parsing legacy projects.json: %w", err)
	}

	db, err := memory.OpenGlobalIndexDB()
	if err != nil {
		return err
	}
	defer db.Close()

	for _, p := range reg.Projects {
		id := ProjectID(p)
		name := filepath.Base(p)
		_, err := db.Exec(
			`INSERT OR IGNORE INTO projects (id, name, path) VALUES (?, ?, ?)`,
			id, name, p,
		)
		if err != nil {
			return fmt.Errorf("importing %s: %w", p, err)
		}
	}

	// Rename the JSON file so we don't re-import on next run.
	_ = os.Rename(jsonPath, jsonPath+".bak")
	return nil
}

func LoadRegistry() (*ProjectRegistry, error) {
	if err := migrateJSONIfNeeded(); err != nil {
		return nil, err
	}

	db, err := memory.OpenGlobalIndexDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT path FROM projects ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("querying projects: %w", err)
	}
	defer rows.Close()

	var paths []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scanning project row: %w", err)
		}
		paths = append(paths, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating project rows: %w", err)
	}

	return &ProjectRegistry{Projects: paths}, nil
}

// SaveRegistry is kept for API compatibility but is a no-op — writes now happen
// per-operation in RegisterProject and RemoveProject.
func SaveRegistry(_ *ProjectRegistry) error {
	return nil
}

func RegisterProject(absPath string) (bool, error) {
	if err := migrateJSONIfNeeded(); err != nil {
		return false, err
	}

	db, err := memory.OpenGlobalIndexDB()
	if err != nil {
		return false, err
	}
	defer db.Close()

	id := ProjectID(absPath)
	name := filepath.Base(absPath)

	res, err := db.Exec(
		`INSERT OR IGNORE INTO projects (id, name, path) VALUES (?, ?, ?)`,
		id, name, absPath,
	)
	if err != nil {
		return false, fmt.Errorf("registering project: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func RemoveProject(absPath string) (bool, error) {
	if err := migrateJSONIfNeeded(); err != nil {
		return false, err
	}

	db, err := memory.OpenGlobalIndexDB()
	if err != nil {
		return false, err
	}
	defer db.Close()

	res, err := db.Exec(`DELETE FROM projects WHERE path = ?`, absPath)
	if err != nil {
		return false, fmt.Errorf("removing project: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func IsFirstEverInit() bool {
	if err := migrateJSONIfNeeded(); err != nil {
		// If we can't touch the DB, treat it as first init.
		return true
	}

	db, err := memory.OpenGlobalIndexDB()
	if err != nil {
		return true
	}
	defer db.Close()

	var count int
	row := db.QueryRow(`SELECT COUNT(*) FROM projects`)
	if err := row.Scan(&count); err != nil {
		return true
	}
	return count == 0
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
