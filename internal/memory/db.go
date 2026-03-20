package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func OpenProjectDB(projectPath string) (*sql.DB, error) {
	dbPath := filepath.Join(projectPath, ".cx", "memory.db")
	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}
	if err := Migrate(db, projectMigrations); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating %s: %w", dbPath, err)
	}
	return db, nil
}

func OpenGlobalIndexDB() (*sql.DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}
	dir := filepath.Join(home, ".cx")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating ~/.cx: %w", err)
	}
	dbPath := filepath.Join(dir, "index.db")
	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}
	if err := Migrate(db, indexMigrations); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating %s: %w", dbPath, err)
	}
	return db, nil
}

func OpenPersonalDB() (*sql.DB, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("getting home directory: %w", err)
	}
	dir := filepath.Join(home, ".cx")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating ~/.cx: %w", err)
	}
	dbPath := filepath.Join(dir, "memory.db")
	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}
	if err := Migrate(db, personalMigrations); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrating %s: %w", dbPath, err)
	}
	return db, nil
}

func openDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating directory for %s: %w", path, err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("setting WAL mode on %s: %w", path, err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enabling foreign keys on %s: %w", path, err)
	}
	return db, nil
}
