package brainstorm

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var namePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type MasterfileInfo struct {
	Name     string
	Path     string
	Modified bool // true if content differs from template
}

func Create(rootDir, name string) (string, error) {
	if len(name) > 40 {
		return "", fmt.Errorf("masterfile name must be at most 40 characters")
	}
	if !namePattern.MatchString(name) {
		return "", fmt.Errorf("masterfile name must be kebab-case (lowercase letters, numbers, hyphens)")
	}

	dir := filepath.Join(rootDir, "docs", "masterfiles")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("creating masterfiles directory: %w", err)
	}

	path := filepath.Join(dir, name+".md")
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("masterfile %q already exists", name)
	}

	if err := atomicWrite(path, []byte(MasterfileTemplate(name))); err != nil {
		return "", fmt.Errorf("writing masterfile: %w", err)
	}

	return path, nil
}

func List(rootDir string) ([]MasterfileInfo, error) {
	dir := filepath.Join(rootDir, "docs", "masterfiles")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading masterfiles directory: %w", err)
	}

	var masterfiles []MasterfileInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".md")
		path := filepath.Join(dir, entry.Name())

		masterfiles = append(masterfiles, MasterfileInfo{
			Name:     name,
			Path:     path,
			Modified: fileModified(path, MasterfileTemplate(name)),
		})
	}

	sort.Slice(masterfiles, func(i, j int) bool {
		return masterfiles[i].Name < masterfiles[j].Name
	})

	return masterfiles, nil
}

func fileModified(path, template string) bool {
	data, err := os.ReadFile(path)
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
