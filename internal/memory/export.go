package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PushResult struct {
	Exported int
	Skipped  int
	Files    []string
}

func Push(db *sql.DB, docsDir string, all bool) (PushResult, error) {
	query := "SELECT id, entity_type, subtype, title, content, author, source, change_id, file_refs, spec_refs, tags, deprecates, deprecated, status, visibility, shared_at, created_at, updated_at, archived_at FROM memories WHERE visibility = 'project'"
	if !all {
		query += " AND shared_at IS NULL"
	}
	// Never export session rows
	query += " AND entity_type != 'session'"

	rows, err := db.Query(query)
	if err != nil {
		return PushResult{}, fmt.Errorf("querying memories for push: %w", err)
	}

	// Collect all rows before closing the cursor so we can write to the DB
	// after the read cursor is done (needed for in-memory SQLite connections).
	var toExport []Memory
	for rows.Next() {
		var m Memory
		var subtype, source, changeID, fileRefs, specRefs, tags, deprecates, status, sharedAt, updatedAt, archivedAt sql.NullString
		if err := rows.Scan(&m.ID, &m.EntityType, &subtype, &m.Title, &m.Content, &m.Author,
			&source, &changeID, &fileRefs, &specRefs, &tags, &deprecates,
			&m.Deprecated, &status, &m.Visibility, &sharedAt,
			&m.CreatedAt, &updatedAt, &archivedAt); err != nil {
			rows.Close()
			return PushResult{}, fmt.Errorf("scanning memory for push: %w", err)
		}
		m.Subtype = subtype.String
		m.Source = source.String
		m.ChangeID = changeID.String
		m.FileRefs = fileRefs.String
		m.SpecRefs = specRefs.String
		m.Tags = tags.String
		m.Deprecates = deprecates.String
		m.Status = status.String
		m.SharedAt = sharedAt.String
		m.UpdatedAt = updatedAt.String
		m.ArchivedAt = archivedAt.String
		toExport = append(toExport, m)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return PushResult{}, err
	}
	rows.Close()

	var result PushResult
	for _, m := range toExport {
		// Determine subdirectory
		subDir := "observations"
		if m.EntityType == "decision" {
			subDir = "decisions"
		}

		dir := filepath.Join(docsDir, "memory", subDir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return PushResult{}, fmt.Errorf("creating %s: %w", dir, err)
		}

		filePath := filepath.Join(dir, m.ID+".md")
		content := memoryToMarkdown(m)

		if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
			return PushResult{}, fmt.Errorf("writing %s: %w", filePath, err)
		}

		// Update shared_at
		if _, err := db.Exec("UPDATE memories SET shared_at = datetime('now') WHERE id = ?", m.ID); err != nil {
			return PushResult{}, fmt.Errorf("updating shared_at for %s: %w", m.ID, err)
		}

		result.Exported++
		result.Files = append(result.Files, filePath)
	}

	return result, nil
}

func memoryToMarkdown(m Memory) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("id: %s\n", m.ID))
	sb.WriteString(fmt.Sprintf("entity_type: %s\n", m.EntityType))
	if m.Subtype != "" {
		sb.WriteString(fmt.Sprintf("subtype: %s\n", m.Subtype))
	}
	sb.WriteString(fmt.Sprintf("title: %s\n", m.Title))
	sb.WriteString(fmt.Sprintf("author: %s\n", m.Author))
	if m.Source != "" {
		sb.WriteString(fmt.Sprintf("source: %s\n", m.Source))
	}
	if m.ChangeID != "" {
		sb.WriteString(fmt.Sprintf("change_id: %s\n", m.ChangeID))
	}
	if m.FileRefs != "" {
		sb.WriteString(fmt.Sprintf("file_refs: %s\n", m.FileRefs))
	}
	if m.SpecRefs != "" {
		sb.WriteString(fmt.Sprintf("spec_refs: %s\n", m.SpecRefs))
	}
	if m.Tags != "" {
		sb.WriteString(fmt.Sprintf("tags: %s\n", m.Tags))
	}
	if m.Deprecates != "" {
		sb.WriteString(fmt.Sprintf("deprecates: %s\n", m.Deprecates))
	}
	if m.Status != "" {
		sb.WriteString(fmt.Sprintf("status: %s\n", m.Status))
	}
	sb.WriteString(fmt.Sprintf("visibility: %s\n", m.Visibility))
	sb.WriteString(fmt.Sprintf("created_at: %s\n", m.CreatedAt))
	if m.UpdatedAt != "" {
		sb.WriteString(fmt.Sprintf("updated_at: %s\n", m.UpdatedAt))
	}
	sb.WriteString("---\n\n")
	sb.WriteString(m.Content)
	sb.WriteString("\n")
	return sb.String()
}
