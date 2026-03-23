package change

import (
	"strings"

	"github.com/AngelMaldonado/cx/internal/templates"
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

func SpecTemplate(area string) string {
	tmpl := templates.MustContent("docs/spec.md")
	return strings.ReplaceAll(tmpl, "{{name}}", area)
}

func DeltaSpecTemplate(name, area string) string {
	tmpl := templates.MustContent("docs/delta-spec.md")
	result := strings.ReplaceAll(tmpl, "{{name}}", name)
	return strings.ReplaceAll(result, "{{area}}", area)
}

func VerifyTemplate(name string) string {
	tmpl := templates.MustContent("docs/verify.md")
	return strings.ReplaceAll(tmpl, "{{name}}", name)
}

func ConfigTemplate() string {
	return templates.MustContent("docs/cx.yaml")
}
