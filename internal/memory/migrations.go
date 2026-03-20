package memory

import (
	"database/sql"
	"fmt"
)

type Migration struct {
	Version     int
	Description string
	Up          func(*sql.DB) error
}

var projectMigrations = []Migration{
	{Version: 1, Description: "initial project schema", Up: v1ProjectSchema},
}

var indexMigrations = []Migration{
	{Version: 1, Description: "initial index schema", Up: v1IndexSchema},
}

func Migrate(db *sql.DB, migrations []Migration) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version     INTEGER PRIMARY KEY,
		applied_at  TEXT DEFAULT (datetime('now')),
		description TEXT
	)`); err != nil {
		return fmt.Errorf("creating schema_migrations: %w", err)
	}

	var current int
	row := db.QueryRow("SELECT COALESCE(MAX(version), 0) FROM schema_migrations")
	if err := row.Scan(&current); err != nil {
		return fmt.Errorf("reading schema version: %w", err)
	}

	for _, m := range migrations {
		if m.Version <= current {
			continue
		}
		if err := m.Up(db); err != nil {
			return fmt.Errorf("migration v%d (%s): %w", m.Version, m.Description, err)
		}
		if _, err := db.Exec("INSERT INTO schema_migrations (version, description) VALUES (?, ?)",
			m.Version, m.Description); err != nil {
			return fmt.Errorf("recording migration v%d: %w", m.Version, err)
		}
	}
	return nil
}

func v1ProjectSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS memories (
			id          TEXT PRIMARY KEY,
			entity_type TEXT NOT NULL CHECK(entity_type IN ('observation', 'decision', 'session', 'agent_interaction')),
			subtype     TEXT,
			title       TEXT NOT NULL,
			content     TEXT NOT NULL,
			author      TEXT NOT NULL,
			source      TEXT,
			change_id   TEXT,
			file_refs   TEXT,
			spec_refs   TEXT,
			tags        TEXT,
			deprecates  TEXT,
			deprecated  INTEGER DEFAULT 0,
			status      TEXT,
			visibility  TEXT NOT NULL DEFAULT 'personal' CHECK(visibility IN ('personal', 'project')),
			shared_at   TEXT,
			created_at  TEXT NOT NULL,
			updated_at  TEXT DEFAULT (datetime('now')),
			archived_at TEXT
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
			title, content, tags, entity_type,
			content=memories, content_rowid=rowid
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id          TEXT PRIMARY KEY,
			mode        TEXT NOT NULL CHECK(mode IN ('build', 'continue', 'plan')),
			change_name TEXT,
			goal        TEXT,
			started_at  TEXT NOT NULL DEFAULT (datetime('now')),
			ended_at    TEXT,
			summary     TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS agent_runs (
			id              TEXT PRIMARY KEY,
			session_id      TEXT REFERENCES sessions(id) ON DELETE SET NULL,
			agent_type      TEXT NOT NULL,
			prompt_summary  TEXT,
			result_status   TEXT CHECK(result_status IN ('success', 'blocked', 'needs-input', NULL)),
			result_summary  TEXT,
			artifacts       TEXT,
			duration_ms     INTEGER,
			created_at      TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS memory_links (
			from_id       TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
			to_id         TEXT NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
			relation_type TEXT NOT NULL CHECK(relation_type IN ('related-to', 'caused-by', 'resolved-by', 'see-also')),
			created_at    TEXT DEFAULT (datetime('now')),
			PRIMARY KEY (from_id, to_id, relation_type)
		)`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("executing schema: %w", err)
		}
	}
	return nil
}

func v1IndexSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id         TEXT PRIMARY KEY,
			name       TEXT NOT NULL,
			path       TEXT NOT NULL UNIQUE,
			git_remote TEXT,
			last_synced TEXT,
			created_at TEXT DEFAULT (datetime('now')),
			updated_at TEXT DEFAULT (datetime('now'))
		)`,
		`CREATE TABLE IF NOT EXISTS memory_index (
			rowid       INTEGER PRIMARY KEY AUTOINCREMENT,
			project_id  TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
			local_id    TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			title       TEXT NOT NULL,
			tags        TEXT,
			created_at  TEXT NOT NULL,
			deprecated  INTEGER DEFAULT 0
		)`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS memory_index_fts USING fts5(
			title, tags,
			content=memory_index, content_rowid=rowid
		)`,
	}
	for _, stmt := range statements {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("executing schema: %w", err)
		}
	}
	return nil
}
