package templates

import "embed"

//go:embed agents/*.md subagents/*.md docs/*.md
var FS embed.FS

// Content reads a template file and returns its content as a string.
// path is relative to internal/templates/ e.g. "docs/proposal.md"
func Content(path string) (string, error) {
	data, err := FS.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// MustContent is like Content but panics on error (for templates that must exist).
func MustContent(path string) string {
	s, err := Content(path)
	if err != nil {
		panic("templates: " + err.Error())
	}
	return s
}
