package skills

import (
	"embed"
	"sort"
	"strings"
)

//go:embed data/*.md
var FS embed.FS

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

func Content(name string) ([]byte, error) {
	return FS.ReadFile("data/" + name)
}
