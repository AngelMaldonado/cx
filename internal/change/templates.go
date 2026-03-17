package change

import (
	"strings"

	"github.com/amald/cx/internal/templates"
)

func ProposalTemplate(name string) string {
	tmpl := templates.MustContent("docs/proposal.md")
	return strings.ReplaceAll(tmpl, "{{name}}", name)
}

func DesignTemplate(name string) string {
	tmpl := templates.MustContent("docs/design.md")
	return strings.ReplaceAll(tmpl, "{{name}}", name)
}

func TasksTemplate(name string) string {
	tmpl := templates.MustContent("docs/tasks.md")
	return strings.ReplaceAll(tmpl, "{{name}}", name)
}
