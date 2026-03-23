package brainstorm

import (
	"strings"

	"github.com/AngelMaldonado/cx/internal/templates"
)

func MasterfileTemplate(name string) string {
	tmpl := templates.MustContent("docs/masterfile.md")
	return strings.ReplaceAll(tmpl, "{{name}}", name)
}
