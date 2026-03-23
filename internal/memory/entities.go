package memory

import (
	"database/sql"
	"fmt"
	"time"
)

type Memory struct {
	ID         string
	EntityType string // observation, decision, session, agent_interaction
	Subtype    string
	Title      string
	Content    string
	Author     string
	Source     string
	ChangeID   string
	FileRefs   string // JSON array
	SpecRefs   string // JSON array
	Tags       string // comma-separated
	Deprecates string
	Deprecated int
	Status     string
	Visibility string // personal, project
	SharedAt   string
	CreatedAt  string
	UpdatedAt  string
	ArchivedAt string
}

type Session struct {
	ID         string
	Mode       string // build, continue, plan
	ChangeName string
	Goal       string
	StartedAt  string
	EndedAt    string
	Summary    string
}

type AgentRun struct {
	ID            string
	SessionID     string
	AgentType     string
	PromptSummary string
	ResultStatus  string // success, blocked, needs-input
	ResultSummary string
	Artifacts     string // JSON array
	DurationMs    int
	CreatedAt     string
}

type MemoryLink struct {
	FromID       string
	ToID         string
	RelationType string // related-to, caused-by, resolved-by, see-also
	CreatedAt    string
}

type ListOpts struct {
	EntityType        string
	ChangeID          string
	Recent            time.Duration
	IncludeDeprecated bool
	IncludeArchived   bool
	Limit             int
}

func SaveMemory(db *sql.DB, m Memory) error {
	if m.ID == "" || m.Title == "" || m.EntityType == "" || m.Author == "" || m.CreatedAt == "" {
		return fmt.Errorf("memory requires id, title, entity_type, author, and created_at")
	}
	if m.Visibility == "" {
		m.Visibility = "personal"
	}
	if m.UpdatedAt == "" {
		m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	var sharedAt interface{}
	if m.SharedAt != "" {
		sharedAt = m.SharedAt
	}
	var archivedAt interface{}
	if m.ArchivedAt != "" {
		archivedAt = m.ArchivedAt
	}
	_, err := db.Exec(`INSERT OR REPLACE INTO memories
		(id, entity_type, subtype, title, content, author, source, change_id,
		 file_refs, spec_refs, tags, deprecates, deprecated, status, visibility,
		 shared_at, created_at, updated_at, archived_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		m.ID, m.EntityType, m.Subtype, m.Title, m.Content, m.Author, m.Source, m.ChangeID,
		m.FileRefs, m.SpecRefs, m.Tags, m.Deprecates, m.Deprecated, m.Status, m.Visibility,
		sharedAt, m.CreatedAt, m.UpdatedAt, archivedAt)
	if err != nil {
		return fmt.Errorf("saving memory: %w", err)
	}

	// Handle deprecation chain
	if m.Deprecates != "" {
		if _, err := db.Exec("UPDATE memories SET deprecated = 1 WHERE id = ?", m.Deprecates); err != nil {
			return fmt.Errorf("deprecating %s: %w", m.Deprecates, err)
		}
		// If both are decisions, set status=superseded on the old one
		if m.EntityType == "decision" {
			if _, err := db.Exec("UPDATE memories SET status = 'superseded' WHERE id = ? AND entity_type = 'decision'",
				m.Deprecates); err != nil {
				return fmt.Errorf("superseding %s: %w", m.Deprecates, err)
			}
		}
	}

	// Update FTS index
	// Delete stale FTS entry before re-inserting
	db.Exec("DELETE FROM memories_fts WHERE rowid = (SELECT rowid FROM memories WHERE id = ?)", m.ID)
	if _, err := db.Exec("INSERT INTO memories_fts(rowid, title, content, tags, entity_type) SELECT rowid, title, content, tags, entity_type FROM memories WHERE id = ?", m.ID); err != nil {
		return fmt.Errorf("updating FTS index for %s: %w", m.ID, err)
	}

	return nil
}

func GetMemory(db *sql.DB, id string) (Memory, error) {
	var m Memory
	err := db.QueryRow(`SELECT id, entity_type, COALESCE(subtype,''), title, content, author,
		COALESCE(source,''), COALESCE(change_id,''), COALESCE(file_refs,''), COALESCE(spec_refs,''),
		COALESCE(tags,''), COALESCE(deprecates,''), deprecated, COALESCE(status,''), visibility,
		COALESCE(shared_at,''), created_at, COALESCE(updated_at,''), COALESCE(archived_at,'')
		FROM memories WHERE id = ?`, id).Scan(
		&m.ID, &m.EntityType, &m.Subtype, &m.Title, &m.Content, &m.Author,
		&m.Source, &m.ChangeID, &m.FileRefs, &m.SpecRefs,
		&m.Tags, &m.Deprecates, &m.Deprecated, &m.Status, &m.Visibility,
		&m.SharedAt, &m.CreatedAt, &m.UpdatedAt, &m.ArchivedAt)
	if err != nil {
		return Memory{}, fmt.Errorf("getting memory %s: %w", id, err)
	}
	return m, nil
}

func ListMemories(db *sql.DB, opts ListOpts) ([]Memory, error) {
	query := "SELECT id, entity_type, COALESCE(subtype,''), title, content, author, COALESCE(source,''), COALESCE(change_id,''), COALESCE(file_refs,''), COALESCE(spec_refs,''), COALESCE(tags,''), COALESCE(deprecates,''), deprecated, COALESCE(status,''), visibility, COALESCE(shared_at,''), created_at, COALESCE(updated_at,''), COALESCE(archived_at,'') FROM memories WHERE 1=1"
	var args []interface{}

	if !opts.IncludeDeprecated {
		query += " AND deprecated = 0"
	}
	if !opts.IncludeArchived {
		query += " AND archived_at IS NULL"
	}
	if opts.EntityType != "" {
		query += " AND entity_type = ?"
		args = append(args, opts.EntityType)
	}
	if opts.ChangeID != "" {
		query += " AND change_id = ?"
		args = append(args, opts.ChangeID)
	}
	if opts.Recent > 0 {
		cutoff := time.Now().Add(-opts.Recent).UTC().Format(time.RFC3339)
		query += " AND created_at >= ?"
		args = append(args, cutoff)
	}
	query += " ORDER BY created_at DESC"
	if opts.Limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", opts.Limit)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing memories: %w", err)
	}
	defer rows.Close()

	var memories []Memory
	for rows.Next() {
		var m Memory
		if err := rows.Scan(&m.ID, &m.EntityType, &m.Subtype, &m.Title, &m.Content, &m.Author,
			&m.Source, &m.ChangeID, &m.FileRefs, &m.SpecRefs,
			&m.Tags, &m.Deprecates, &m.Deprecated, &m.Status, &m.Visibility,
			&m.SharedAt, &m.CreatedAt, &m.UpdatedAt, &m.ArchivedAt); err != nil {
			return nil, fmt.Errorf("scanning memory row: %w", err)
		}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

func SaveSession(db *sql.DB, s Session) error {
	if s.ID == "" || s.Mode == "" {
		return fmt.Errorf("session requires id and mode")
	}
	if s.StartedAt == "" {
		s.StartedAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := db.Exec(`INSERT OR REPLACE INTO sessions (id, mode, change_name, goal, started_at, ended_at, summary)
		VALUES (?, ?, ?, ?, ?, ?, ?)`, s.ID, s.Mode, s.ChangeName, s.Goal, s.StartedAt, s.EndedAt, s.Summary)
	if err != nil {
		return fmt.Errorf("saving session: %w", err)
	}
	return nil
}

func GetLatestSession(db *sql.DB) (Session, error) {
	var s Session
	err := db.QueryRow(`SELECT id, mode, COALESCE(change_name,''), COALESCE(goal,''),
		started_at, COALESCE(ended_at,''), COALESCE(summary,'')
		FROM sessions ORDER BY started_at DESC LIMIT 1`).Scan(
		&s.ID, &s.Mode, &s.ChangeName, &s.Goal, &s.StartedAt, &s.EndedAt, &s.Summary)
	if err != nil {
		return Session{}, fmt.Errorf("getting latest session: %w", err)
	}
	return s, nil
}

func SaveAgentRun(db *sql.DB, run AgentRun) error {
	if run.ID == "" || run.AgentType == "" {
		return fmt.Errorf("agent run requires id and agent_type")
	}
	if run.CreatedAt == "" {
		run.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	var sessionID interface{}
	if run.SessionID != "" {
		sessionID = run.SessionID
	}
	_, err := db.Exec(`INSERT OR REPLACE INTO agent_runs
		(id, session_id, agent_type, prompt_summary, result_status, result_summary, artifacts, duration_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		run.ID, sessionID, run.AgentType, run.PromptSummary, run.ResultStatus,
		run.ResultSummary, run.Artifacts, run.DurationMs, run.CreatedAt)
	if err != nil {
		return fmt.Errorf("saving agent run: %w", err)
	}
	return nil
}

func ListAgentRuns(db *sql.DB, sessionID string) ([]AgentRun, error) {
	return ListAgentRunsLimit(db, sessionID, 100)
}

// ListAgentRunsLimit returns agent runs filtered by sessionID (empty = all),
// ordered by created_at DESC, capped at limit rows (0 = use default of 100).
func ListAgentRunsLimit(db *sql.DB, sessionID string, limit int) ([]AgentRun, error) {
	if limit <= 0 {
		limit = 100
	}
	query := "SELECT id, COALESCE(session_id,''), agent_type, COALESCE(prompt_summary,''), COALESCE(result_status,''), COALESCE(result_summary,''), COALESCE(artifacts,''), COALESCE(duration_ms,0), created_at FROM agent_runs"
	var args []interface{}
	if sessionID != "" {
		query += " WHERE session_id = ?"
		args = append(args, sessionID)
	}
	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d", limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing agent runs: %w", err)
	}
	defer rows.Close()

	var runs []AgentRun
	for rows.Next() {
		var r AgentRun
		if err := rows.Scan(&r.ID, &r.SessionID, &r.AgentType, &r.PromptSummary,
			&r.ResultStatus, &r.ResultSummary, &r.Artifacts, &r.DurationMs, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning agent run row: %w", err)
		}
		runs = append(runs, r)
	}
	return runs, rows.Err()
}

func SaveMemoryLink(db *sql.DB, link MemoryLink) error {
	if link.FromID == "" || link.ToID == "" || link.RelationType == "" {
		return fmt.Errorf("memory link requires from_id, to_id, and relation_type")
	}
	_, err := db.Exec(`INSERT OR IGNORE INTO memory_links (from_id, to_id, relation_type) VALUES (?, ?, ?)`,
		link.FromID, link.ToID, link.RelationType)
	if err != nil {
		return fmt.Errorf("saving memory link: %w", err)
	}
	return nil
}

// ListMemoryLinks returns memory links ordered by from_id, limited to 500 rows.
// Returns an empty slice (not an error) if the table is empty.
func ListMemoryLinks(db *sql.DB) ([]MemoryLink, error) {
	rows, err := db.Query(`SELECT from_id, to_id, relation_type, COALESCE(created_at,'')
		FROM memory_links ORDER BY from_id LIMIT 500`)
	if err != nil {
		return nil, fmt.Errorf("listing memory links: %w", err)
	}
	defer rows.Close()

	var links []MemoryLink
	for rows.Next() {
		var l MemoryLink
		if err := rows.Scan(&l.FromID, &l.ToID, &l.RelationType, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning memory link row: %w", err)
		}
		links = append(links, l)
	}
	return links, rows.Err()
}

// DeprecateMemory marks the memory with the given id as deprecated (deprecated=1).
// It returns an error if the memory does not exist or is already deprecated.
// After updating the row it removes the FTS entry so deprecated memories no
// longer appear in full-text search results.
func DeprecateMemory(db *sql.DB, id string) error {
	result, err := db.Exec(
		"UPDATE memories SET deprecated = 1, updated_at = datetime('now') WHERE id = ? AND deprecated = 0",
		id,
	)
	if err != nil {
		return fmt.Errorf("deprecating memory %s: %w", id, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("checking rows affected: %w", err)
	}
	if rows == 0 {
		// Distinguish between not found and already deprecated
		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM memories WHERE id = ?", id).Scan(&count); err != nil {
			return fmt.Errorf("checking memory existence: %w", err)
		}
		if count == 0 {
			return fmt.Errorf("memory %q not found", id)
		}
		return fmt.Errorf("memory %q is already deprecated", id)
	}

	// Remove from FTS index so deprecated memories don't appear in searches.
	db.Exec("DELETE FROM memories_fts WHERE rowid = (SELECT rowid FROM memories WHERE id = ?)", id)

	return nil
}

// SessionListOpts controls filtering for ListSessions.
type SessionListOpts struct {
	ChangeName string
	Mode       string // build, continue, plan
	Recent     time.Duration
	Limit      int
}

// ListSessions returns sessions ordered by started_at DESC.
// Filters by ChangeName, Mode, and Recent if non-zero.
// Defaults to a limit of 50 if Limit is not specified.
func ListSessions(db *sql.DB, opts SessionListOpts) ([]Session, error) {
	query := `SELECT id, mode, COALESCE(change_name,''), COALESCE(goal,''),
		started_at, COALESCE(ended_at,''), COALESCE(summary,'')
		FROM sessions WHERE 1=1`
	var args []interface{}

	if opts.ChangeName != "" {
		query += " AND change_name = ?"
		args = append(args, opts.ChangeName)
	}
	if opts.Mode != "" {
		query += " AND mode = ?"
		args = append(args, opts.Mode)
	}
	if opts.Recent > 0 {
		cutoff := time.Now().Add(-opts.Recent).UTC().Format(time.RFC3339)
		query += " AND started_at >= ?"
		args = append(args, cutoff)
	}
	query += " ORDER BY started_at DESC"

	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing sessions: %w", err)
	}
	defer rows.Close()

	var sessions []Session
	for rows.Next() {
		var s Session
		if err := rows.Scan(&s.ID, &s.Mode, &s.ChangeName, &s.Goal,
			&s.StartedAt, &s.EndedAt, &s.Summary); err != nil {
			return nil, fmt.Errorf("scanning session row: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// PersonalNote mirrors the personal_notes table in ~/.cx/memory.db.
type PersonalNote struct {
	ID        string
	TopicKey  string
	Title     string
	Content   string
	Tags      string
	Projects  string
	CreatedAt string
	UpdatedAt string
}

// ListPersonalNotes returns personal notes ordered by updated_at DESC.
// Defaults to a limit of 50 if limit is <= 0.
func ListPersonalNotes(db *sql.DB, limit int) ([]PersonalNote, error) {
	if limit <= 0 {
		limit = 50
	}
	query := fmt.Sprintf(`SELECT id, COALESCE(topic_key,''), title, content,
		COALESCE(tags,''), COALESCE(projects,''), created_at, COALESCE(updated_at,'')
		FROM personal_notes ORDER BY updated_at DESC LIMIT %d`, limit)

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("listing personal notes: %w", err)
	}
	defer rows.Close()

	var notes []PersonalNote
	for rows.Next() {
		var n PersonalNote
		if err := rows.Scan(&n.ID, &n.TopicKey, &n.Title, &n.Content,
			&n.Tags, &n.Projects, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning personal note row: %w", err)
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

// MemoryStats holds aggregate counts for the dashboard home stats panel.
type MemoryStats struct {
	TotalObservations int
	TotalDecisions    int
	TotalSessions     int
	TotalAgentRuns    int
	TotalLinks        int
}

// CountMemories returns aggregate counts from memories (by entity_type),
// sessions, agent_runs, and memory_links for the dashboard home view.
func CountMemories(db *sql.DB) (MemoryStats, error) {
	var stats MemoryStats

	// Count non-deprecated memories grouped by entity_type
	rows, err := db.Query(`SELECT entity_type, COUNT(*) FROM memories WHERE deprecated = 0 GROUP BY entity_type`)
	if err != nil {
		return stats, fmt.Errorf("counting memories by type: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var entityType string
		var count int
		if err := rows.Scan(&entityType, &count); err != nil {
			return stats, fmt.Errorf("scanning memory count row: %w", err)
		}
		switch entityType {
		case "observation":
			stats.TotalObservations = count
		case "decision":
			stats.TotalDecisions = count
		}
	}
	if err := rows.Err(); err != nil {
		return stats, err
	}

	if err := db.QueryRow(`SELECT COUNT(*) FROM sessions`).Scan(&stats.TotalSessions); err != nil {
		return stats, fmt.Errorf("counting sessions: %w", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM agent_runs`).Scan(&stats.TotalAgentRuns); err != nil {
		return stats, fmt.Errorf("counting agent runs: %w", err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM memory_links`).Scan(&stats.TotalLinks); err != nil {
		return stats, fmt.Errorf("counting memory links: %w", err)
	}

	return stats, nil
}
