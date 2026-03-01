package skills

import (
	"embed"
	"sort"
	"strings"
)

//go:embed data/*.md
var FS embed.FS

// Names returns embedded filenames (e.g., "cx-brainstorm.md").
func Names() []string {
	entries, err := FS.ReadDir("data")
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// Slugs returns skill slugs (e.g., "cx-brainstorm") derived from filenames.
func Slugs() []string {
	names := Names()
	slugs := make([]string, len(names))
	for i, n := range names {
		slugs[i] = strings.TrimSuffix(n, ".md")
	}
	return slugs
}

// Content reads the embedded file by its filename (e.g., "cx-brainstorm.md").
func Content(name string) ([]byte, error) {
	return FS.ReadFile("data/" + name)
}
