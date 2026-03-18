package change

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var namePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type ChangeInfo struct {
	Name        string
	Path        string
	HasProposal bool
	HasDesign   bool
	HasTasks    bool
	DeltaSpecs  []string
}

func Create(rootDir, name string) error {
	if len(name) > 40 {
		return fmt.Errorf("change name must be at most 40 characters")
	}
	if !namePattern.MatchString(name) {
		return fmt.Errorf("change name must be kebab-case (lowercase letters, numbers, hyphens)")
	}

	dir := filepath.Join(rootDir, "docs", "changes", name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("change %q already exists", name)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating change directory: %w", err)
	}

	files := map[string]string{
		"proposal.md": ProposalTemplate(name),
		"design.md":   DesignTemplate(name),
		"tasks.md":    TasksTemplate(name),
	}

	for filename, content := range files {
		path := filepath.Join(dir, filename)
		if err := atomicWrite(path, []byte(content)); err != nil {
			return fmt.Errorf("writing %s: %w", filename, err)
		}
	}

	return nil
}

func ListChanges(rootDir string) ([]ChangeInfo, error) {
	changesDir := filepath.Join(rootDir, "docs", "changes")
	entries, err := os.ReadDir(changesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading changes directory: %w", err)
	}

	var changes []ChangeInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		dir := filepath.Join(changesDir, name)

		info := ChangeInfo{
			Name:        name,
			Path:        dir,
			HasProposal: fileModified(dir, "proposal.md", ProposalTemplate(name)),
			HasDesign:   fileModified(dir, "design.md", DesignTemplate(name)),
			HasTasks:    fileModified(dir, "tasks.md", TasksTemplate(name)),
		}

		specsDir := filepath.Join(dir, "specs")
		if specEntries, err := os.ReadDir(specsDir); err == nil {
			for _, se := range specEntries {
				if se.IsDir() {
					info.DeltaSpecs = append(info.DeltaSpecs, se.Name())
				}
			}
		}

		changes = append(changes, info)
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Name < changes[j].Name
	})

	return changes, nil
}

func Archive(rootDir, name string) error {
	changesDir := filepath.Join(rootDir, "docs", "changes", name)
	if _, err := os.Stat(changesDir); os.IsNotExist(err) {
		return fmt.Errorf("change %q does not exist", name)
	}

	info := ChangeInfo{
		Name:        name,
		Path:        changesDir,
		HasProposal: fileModified(changesDir, "proposal.md", ProposalTemplate(name)),
		HasDesign:   fileModified(changesDir, "design.md", DesignTemplate(name)),
		HasTasks:    fileModified(changesDir, "tasks.md", TasksTemplate(name)),
	}

	var missing []string
	if !info.HasProposal {
		missing = append(missing, "proposal.md")
	}
	if !info.HasDesign {
		missing = append(missing, "design.md")
	}
	if !info.HasTasks {
		missing = append(missing, "tasks.md")
	}
	if len(missing) > 0 {
		return fmt.Errorf("change %q is incomplete — missing: %s", name, strings.Join(missing, ", "))
	}

	date := time.Now().Format("2006-01-02")
	archiveDir := filepath.Join(rootDir, "docs", "archive", date+"-"+name)
	if err := os.MkdirAll(filepath.Dir(archiveDir), 0o755); err != nil {
		return fmt.Errorf("creating archive directory: %w", err)
	}
	if err := os.Rename(changesDir, archiveDir); err != nil {
		return fmt.Errorf("archiving change: %w", err)
	}

	return nil
}

func fileModified(dir, filename, template string) bool {
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(data)) != strings.TrimSpace(template)
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
