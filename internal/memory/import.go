package memory

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type PullResult struct {
	Imported  int
	Skipped   int
	Conflicts []ConflictItem
}

type ConflictItem struct {
	ID              string
	LocalContent    string
	ImportedContent string
}

func Pull(db *sql.DB, docsDir string) (PullResult, error) {
	var result PullResult
	memDir := filepath.Join(docsDir, "memory")

	for _, subDir := range []string{"observations", "decisions"} {
		dir := filepath.Join(memDir, subDir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return PullResult{}, fmt.Errorf("reading %s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			filePath := filepath.Join(dir, entry.Name())
			m, err := parseMemoryFile(filePath)
			if err != nil {
				continue // skip unparseable files
			}

			// Check if exists in DB
			existing, getErr := GetMemory(db, m.ID)
			if getErr != nil {
				// Not in DB — import
				if err := SaveMemory(db, m); err != nil {
					return PullResult{}, fmt.Errorf("importing %s: %w", m.ID, err)
				}
				result.Imported++
				continue
			}

			// Exists — check for conflict
			if existing.Content == m.Content && existing.Title == m.Title {
				result.Skipped++
				continue
			}

			// Content differs — record conflict, skip
			result.Conflicts = append(result.Conflicts, ConflictItem{
				ID:              m.ID,
				LocalContent:    existing.Content,
				ImportedContent: m.Content,
			})
			result.Skipped++
		}
	}

	return result, nil
}

func RebuildFromMarkdown(db *sql.DB, docsDir string) error {
	memDir := filepath.Join(docsDir, "memory")

	for _, subDir := range []string{"observations", "decisions"} {
		dir := filepath.Join(memDir, subDir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("reading %s: %w", dir, err)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			filePath := filepath.Join(dir, entry.Name())
			m, err := parseMemoryFile(filePath)
			if err != nil {
				continue
			}
			if err := SaveMemory(db, m); err != nil {
				return fmt.Errorf("rebuilding %s: %w", m.ID, err)
			}
		}
	}
	return nil
}

func parseMemoryFile(path string) (Memory, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Memory{}, fmt.Errorf("reading %s: %w", path, err)
	}

	content := string(data)
	if !strings.HasPrefix(content, "---\n") {
		return Memory{}, fmt.Errorf("no frontmatter in %s", path)
	}

	rest := content[4:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return Memory{}, fmt.Errorf("no closing frontmatter in %s", path)
	}

	frontmatter := rest[:idx]
	body := strings.TrimSpace(rest[idx+4:])

	m := Memory{
		Content:   body,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Parse frontmatter key-value pairs
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		value := strings.TrimSpace(line[colonIdx+1:])

		switch key {
		case "id":
			m.ID = value
		case "entity_type":
			m.EntityType = value
		case "subtype":
			m.Subtype = value
		case "title":
			m.Title = value
		case "author":
			m.Author = value
		case "source":
			m.Source = value
		case "change_id":
			m.ChangeID = value
		case "file_refs":
			m.FileRefs = value
		case "spec_refs":
			m.SpecRefs = value
		case "tags":
			m.Tags = value
		case "deprecates":
			m.Deprecates = value
		case "status":
			m.Status = value
		case "visibility":
			m.Visibility = value
		case "created_at":
			m.CreatedAt = value
		case "updated_at":
			m.UpdatedAt = value
		}
	}

	if m.ID == "" {
		// Derive ID from filename
		m.ID = strings.TrimSuffix(filepath.Base(path), ".md")
	}
	if m.Visibility == "" {
		m.Visibility = "project"
	}
	if m.Author == "" {
		m.Author = "unknown"
	}

	return m, nil
}
