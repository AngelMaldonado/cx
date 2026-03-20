package memory

import (
	"database/sql"
	"fmt"
)

type SearchOpts struct {
	EntityType        string
	ChangeID          string
	Author            string
	IncludeDeprecated bool
	Limit             int
}

type MemoryResult struct {
	Memory
	Rank float64
}

type ProjectMemoryResult struct {
	Memory
	ProjectName string
	ProjectPath string
	Rank        float64
}

func SearchMemories(db *sql.DB, query string, opts SearchOpts) ([]MemoryResult, error) {
	q := `SELECT m.id, m.entity_type, COALESCE(m.subtype,''), m.title, m.content, m.author,
		COALESCE(m.source,''), COALESCE(m.change_id,''), COALESCE(m.file_refs,''), COALESCE(m.spec_refs,''),
		COALESCE(m.tags,''), COALESCE(m.deprecates,''), m.deprecated, COALESCE(m.status,''), m.visibility,
		COALESCE(m.shared_at,''), m.created_at, COALESCE(m.updated_at,''), COALESCE(m.archived_at,''),
		fts.rank
		FROM memories_fts fts
		JOIN memories m ON m.rowid = fts.rowid
		WHERE memories_fts MATCH ?`
	var args []interface{}
	args = append(args, query)

	if !opts.IncludeDeprecated {
		q += " AND m.deprecated = 0"
	}
	if opts.EntityType != "" {
		q += " AND m.entity_type = ?"
		args = append(args, opts.EntityType)
	}
	if opts.ChangeID != "" {
		q += " AND m.change_id = ?"
		args = append(args, opts.ChangeID)
	}
	if opts.Author != "" {
		q += " AND m.author = ?"
		args = append(args, opts.Author)
	}
	q += " ORDER BY fts.rank"
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	q += fmt.Sprintf(" LIMIT %d", limit)

	rows, err := db.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("searching memories: %w", err)
	}
	defer rows.Close()

	var results []MemoryResult
	for rows.Next() {
		var r MemoryResult
		if err := rows.Scan(&r.ID, &r.EntityType, &r.Subtype, &r.Title, &r.Content, &r.Author,
			&r.Source, &r.ChangeID, &r.FileRefs, &r.SpecRefs,
			&r.Tags, &r.Deprecates, &r.Deprecated, &r.Status, &r.Visibility,
			&r.SharedAt, &r.CreatedAt, &r.UpdatedAt, &r.ArchivedAt,
			&r.Rank); err != nil {
			return nil, fmt.Errorf("scanning search result: %w", err)
		}
		results = append(results, r)
	}
	return results, rows.Err()
}

func SearchAllProjects(globalDB *sql.DB, query string, opts SearchOpts) ([]ProjectMemoryResult, error) {
	rows, err := globalDB.Query("SELECT id, name, path FROM projects")
	if err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}
	defer rows.Close()

	type projectInfo struct {
		ID   string
		Name string
		Path string
	}
	var projects []projectInfo
	for rows.Next() {
		var p projectInfo
		if err := rows.Scan(&p.ID, &p.Name, &p.Path); err != nil {
			return nil, fmt.Errorf("scanning project: %w", err)
		}
		projects = append(projects, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var allResults []ProjectMemoryResult
	for _, p := range projects {
		pdb, err := OpenProjectDB(p.Path)
		if err != nil {
			continue // skip projects we can't open
		}
		results, err := SearchMemories(pdb, query, opts)
		pdb.Close()
		if err != nil {
			continue
		}
		for _, r := range results {
			allResults = append(allResults, ProjectMemoryResult{
				Memory:      r.Memory,
				ProjectName: p.Name,
				ProjectPath: p.Path,
				Rank:        r.Rank,
			})
		}
	}
	return allResults, nil
}
