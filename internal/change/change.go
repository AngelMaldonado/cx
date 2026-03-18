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
	Name         string
	Path         string
	HasProposal  bool
	HasDesign    bool
	HasTasks     bool
	HasVerify    bool
	VerifyStatus string // "PASS", "FAIL", or "PENDING"
	DeltaSpecs   []string
	SyncedDeltas []string
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

	specsDir := filepath.Join(dir, "specs")
	if err := os.MkdirAll(specsDir, 0o755); err != nil {
		return fmt.Errorf("creating specs directory: %w", err)
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
					// Check if delta is synced
					deltaPath := filepath.Join(specsDir, se.Name(), "spec.md")
					if deltaData, err := os.ReadFile(deltaPath); err == nil {
						if strings.Contains(string(deltaData), "synced: true") {
							info.SyncedDeltas = append(info.SyncedDeltas, se.Name())
						}
					}
				}
			}
		}

		// Check verify.md
		verifyPath := filepath.Join(dir, "verify.md")
		if verifyData, err := os.ReadFile(verifyPath); err == nil {
			info.HasVerify = true
			content := string(verifyData)
			hasPASS := strings.Contains(content, "PASS")
			hasCRITICAL := false
			for _, line := range strings.Split(content, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "CRITICAL") {
					hasCRITICAL = true
					break
				}
			}
			if hasPASS && !hasCRITICAL {
				info.VerifyStatus = "PASS"
			} else {
				info.VerifyStatus = "FAIL"
			}
		} else {
			info.VerifyStatus = "PENDING"
		}

		changes = append(changes, info)
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Name < changes[j].Name
	})

	return changes, nil
}

type ArchiveResult struct {
	ArchivePath       string
	BootstrappedSpecs []string
}

type ArchiveOptions struct {
	SkipSpecs bool
}

func Archive(rootDir, name string, opts ArchiveOptions) (*ArchiveResult, error) {
	changesDir := filepath.Join(rootDir, "docs", "changes", name)
	if _, err := os.Stat(changesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("change %q does not exist", name)
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
		return nil, fmt.Errorf("change %q is incomplete — missing: %s", name, strings.Join(missing, ", "))
	}

	// Verify gate (skip if --skip-specs)
	if !opts.SkipSpecs {
		verifyPath := filepath.Join(changesDir, "verify.md")
		verifyData, err := os.ReadFile(verifyPath)
		if err != nil {
			return nil, fmt.Errorf("change %q has no verify.md — run cx change verify first", name)
		}
		content := string(verifyData)
		hasPASS := strings.Contains(content, "PASS")
		hasCRITICAL := false
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "CRITICAL") {
				hasCRITICAL = true
				break
			}
		}
		if !hasPASS || hasCRITICAL {
			return nil, fmt.Errorf("change %q verify.md does not have PASS status — review must pass before archiving", name)
		}
	}

	// Bootstrap missing canonical specs
	var bootstrapped []string
	deltaSpecsDir := filepath.Join(changesDir, "specs")
	if entries, err := os.ReadDir(deltaSpecsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			area := entry.Name()
			// Skip already-synced deltas
			deltaPath := filepath.Join(deltaSpecsDir, area, "spec.md")
			if deltaData, err := os.ReadFile(deltaPath); err == nil {
				if strings.Contains(string(deltaData), "synced: true") {
					continue
				}
			}
			canonicalSpec := filepath.Join(rootDir, "docs", "specs", area, "spec.md")
			if _, err := os.Stat(canonicalSpec); os.IsNotExist(err) {
				if err := os.MkdirAll(filepath.Dir(canonicalSpec), 0o755); err != nil {
					return nil, fmt.Errorf("creating spec directory for %s: %w", area, err)
				}
				if err := atomicWrite(canonicalSpec, []byte(SpecTemplate(area))); err != nil {
					return nil, fmt.Errorf("writing spec scaffold for %s: %w", area, err)
				}
				bootstrapped = append(bootstrapped, area)
			}
		}
		sort.Strings(bootstrapped)
	}

	// Move to archive
	date := time.Now().Format("2006-01-02")
	archivePath := filepath.Join("docs", "archive", date+"-"+name)
	archiveDir := filepath.Join(rootDir, archivePath)
	if err := os.MkdirAll(filepath.Dir(archiveDir), 0o755); err != nil {
		return nil, fmt.Errorf("creating archive directory: %w", err)
	}
	if err := os.Rename(changesDir, archiveDir); err != nil {
		return nil, fmt.Errorf("archiving change: %w", err)
	}

	return &ArchiveResult{
		ArchivePath:       archivePath,
		BootstrappedSpecs: bootstrapped,
	}, nil
}

type SpecSyncResult struct {
	Areas  []string
	Prompt string
}

func SpecSync(rootDir, name string) (*SpecSyncResult, error) {
	changesDir := filepath.Join(rootDir, "docs", "changes", name)
	if _, err := os.Stat(changesDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("change %q does not exist", name)
	}

	// Proposal must be filled
	if !fileModified(changesDir, "proposal.md", ProposalTemplate(name)) {
		return nil, fmt.Errorf("change %q proposal.md is not filled — required before spec sync", name)
	}

	deltaSpecsDir := filepath.Join(changesDir, "specs")
	entries, err := os.ReadDir(deltaSpecsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("change %q has no specs/ directory", name)
		}
		return nil, fmt.Errorf("reading specs directory: %w", err)
	}

	var areas []string
	var promptSections []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		area := entry.Name()
		deltaPath := filepath.Join(deltaSpecsDir, area, "spec.md")
		deltaData, err := os.ReadFile(deltaPath)
		if err != nil {
			continue
		}

		// Skip already synced
		if strings.Contains(string(deltaData), "synced: true") {
			continue
		}

		areas = append(areas, area)

		// Read canonical spec if it exists
		canonicalPath := filepath.Join(rootDir, "docs", "specs", area, "spec.md")
		canonicalData, _ := os.ReadFile(canonicalPath)

		// Build merge prompt section
		section := fmt.Sprintf("### Spec Area: %s\n\n", area)
		if len(canonicalData) > 0 {
			section += fmt.Sprintf("#### Canonical Spec (docs/specs/%s/spec.md)\n\n%s\n\n", area, strings.TrimSpace(string(canonicalData)))
		} else {
			section += fmt.Sprintf("#### Canonical Spec (docs/specs/%s/spec.md)\n\n(empty — new spec area)\n\n", area)
		}
		section += fmt.Sprintf("#### Delta Spec (changes/%s/specs/%s/spec.md)\n\n%s\n", name, area, strings.TrimSpace(string(deltaData)))
		promptSections = append(promptSections, section)
	}

	if len(areas) == 0 {
		return nil, fmt.Errorf("change %q has no unsynced delta specs", name)
	}

	prompt := "# Spec Sync: " + name + "\n\n"
	prompt += "Merge each delta spec into the corresponding canonical spec.\n"
	prompt += "ADDED requirements should be appended. MODIFIED requirements should replace the original. REMOVED requirements should be deleted.\n\n"
	prompt += strings.Join(promptSections, "\n---\n\n")

	return &SpecSyncResult{
		Areas:  areas,
		Prompt: prompt,
	}, nil
}

func MarkDeltaSynced(rootDir, name, area string) error {
	deltaPath := filepath.Join(rootDir, "docs", "changes", name, "specs", area, "spec.md")
	data, err := os.ReadFile(deltaPath)
	if err != nil {
		return fmt.Errorf("reading delta spec: %w", err)
	}

	content := string(data)
	if strings.Contains(content, "synced: true") {
		return nil // already synced
	}

	// Insert synced: true into frontmatter
	if strings.HasPrefix(content, "---\n") {
		rest := content[4:]
		if idx := strings.Index(rest, "\n---"); idx >= 0 {
			newContent := "---\n" + rest[:idx] + "\nsynced: true" + rest[idx:]
			return atomicWrite(deltaPath, []byte(newContent))
		}
	}

	// No frontmatter — prepend one
	newContent := "---\nsynced: true\n---\n" + content
	return atomicWrite(deltaPath, []byte(newContent))
}

func fileModified(dir, filename, template string) bool {
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return false
	}
	body := stripFrontmatter(string(data))
	tmplBody := stripFrontmatter(template)
	return strings.TrimSpace(body) != strings.TrimSpace(tmplBody)
}

func stripFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---\n") {
		return content
	}
	rest := content[4:] // skip opening "---\n"
	if idx := strings.Index(rest, "\n---"); idx >= 0 {
		return rest[idx+4:] // skip past closing "\n---"
	}
	return "" // only frontmatter, no body
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
