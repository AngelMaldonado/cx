package brainstorm

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/amald/cx/internal/change"
)

// DecomposeResult describes what cx decompose created.
type DecomposeResult struct {
	ChangePath  string
	ArchivePath string
}

// Decompose scaffolds a change structure from a masterfile and archives the masterfile.
//
//  1. Validates docs/masterfiles/<name>.md exists
//  2. Creates docs/changes/<name>/ with empty template files (proposal.md, design.md, tasks.md)
//  3. Archives masterfile to docs/archive/<date>-masterfile-<name>.md
//
// The change docs are left empty for the implementation agent to fill in.
// The agent reads the archived masterfile, checks existing specs, and writes
// context-aware proposal/design/tasks content — this cannot be done mechanically.
func Decompose(rootDir, name string) (*DecomposeResult, error) {
	masterfilePath := filepath.Join(rootDir, "docs", "masterfiles", name+".md")
	if _, err := os.Stat(masterfilePath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("masterfile %q not found", name)
		}
		return nil, fmt.Errorf("reading masterfile: %w", err)
	}

	// Create change directory with empty templates
	changeDir := filepath.Join(rootDir, "docs", "changes", name)
	if _, err := os.Stat(changeDir); err == nil {
		return nil, fmt.Errorf("change %q already exists — cannot decompose", name)
	}
	if err := os.MkdirAll(changeDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating change directory: %w", err)
	}

	files := map[string]string{
		"proposal.md": change.ProposalTemplate(name),
		"design.md":   change.DesignTemplate(name),
		"tasks.md":    change.TasksTemplate(name),
	}

	for filename, content := range files {
		path := filepath.Join(changeDir, filename)
		if err := atomicWrite(path, []byte(content)); err != nil {
			return nil, fmt.Errorf("writing %s: %w", filename, err)
		}
	}

	specsDir := filepath.Join(changeDir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating specs directory: %w", err)
	}

	// Archive the masterfile
	archiveDir := filepath.Join(rootDir, "docs", "archive")
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating archive directory: %w", err)
	}
	date := time.Now().Format("2006-01-02")
	archivePath := filepath.Join(archiveDir, fmt.Sprintf("%s-masterfile-%s.md", date, name))
	if err := os.Rename(masterfilePath, archivePath); err != nil {
		return nil, fmt.Errorf("archiving masterfile: %w", err)
	}

	return &DecomposeResult{
		ChangePath:  changeDir,
		ArchivePath: archivePath,
	}, nil
}
