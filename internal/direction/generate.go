package direction

import (
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type preset struct {
	AlwaysSave []string
	NeverSave  []string
	Guidance   []string
}

var projectPresets = map[string]preset{
	"web-api": {
		AlwaysSave: []string{"API contract changes", "authentication/authorization decisions", "database schema migrations", "rate limiting and throttling rules"},
		NeverSave:  []string{"request/response payload examples that duplicate the spec", "temporary debugging endpoints"},
		Guidance:   []string{"Prefer backward-compatible API changes", "Document breaking changes in a migration guide", "All endpoints must have OpenAPI annotations"},
	},
	"frontend": {
		AlwaysSave: []string{"component hierarchy decisions", "state management approach", "accessibility requirements", "design system tokens"},
		NeverSave:  []string{"pixel-perfect layout tweaks", "temporary console.log debugging"},
		Guidance:   []string{"Components should be composable and reusable", "Follow WCAG 2.1 AA accessibility standards", "Prefer CSS modules or styled-components over global styles"},
	},
	"firmware": {
		AlwaysSave: []string{"memory layout decisions", "interrupt handler design", "power management strategy", "hardware abstraction boundaries"},
		NeverSave:  []string{"register dump debugging sessions", "temporary test harness code"},
		Guidance:   []string{"Minimize dynamic allocation — prefer static buffers", "Document hardware dependencies explicitly", "All ISRs must complete within documented time budgets"},
	},
	"cli": {
		AlwaysSave: []string{"command structure and flag design", "output format decisions (JSON, table, etc.)", "configuration file schema", "error message conventions"},
		NeverSave:  []string{"temporary debug output", "one-off scripting helpers"},
		Guidance:   []string{"Follow POSIX conventions for flags and exit codes", "Support both human-readable and machine-readable output", "Provide helpful error messages with suggested fixes"},
	},
	"full-stack": {
		AlwaysSave: []string{"API contract between frontend and backend", "authentication flow decisions", "database schema changes", "deployment architecture"},
		NeverSave:  []string{"temporary mock data", "debugging middleware"},
		Guidance:   []string{"Keep frontend and backend contracts in sync", "Document environment-specific configuration", "Prefer type-safe API boundaries"},
	},
	"other": {
		AlwaysSave: []string{"architectural decisions", "external interface contracts", "configuration schema"},
		NeverSave:  []string{"temporary debugging artifacts"},
		Guidance:   []string{"Document decisions as you make them", "Keep specs up to date with implementation"},
	},
}

var priorityGuidance = map[string][]string{
	"performance": {
		"Profile before optimizing — measure, don't guess",
		"Document performance budgets and benchmarks",
		"Save decisions about caching strategies and data structure choices",
	},
	"external-systems": {
		"Document all external API contracts and SLAs",
		"Save retry/backoff/circuit-breaker decisions",
		"Record authentication and credential management approaches",
	},
	"security": {
		"Document threat model and trust boundaries",
		"Save authentication and authorization architecture decisions",
		"Record secrets management and rotation policies",
	},
	"ux": {
		"Document interaction patterns and user flows",
		"Save accessibility requirements and testing approach",
		"Record design system decisions and component contracts",
	},
	"data-model": {
		"Document entity relationships and invariants",
		"Save migration strategy and backward compatibility decisions",
		"Record indexing and query pattern decisions",
	},
	"infrastructure": {
		"Document deployment topology and scaling strategy",
		"Save monitoring, alerting, and observability decisions",
		"Record disaster recovery and backup procedures",
	},
	"integration": {
		"Document integration points and data flow between systems",
		"Save message format and protocol decisions",
		"Record error handling and retry strategies for integrations",
	},
}

func GenerateDirection(projectType string, priorities []string) string {
	p, ok := projectPresets[projectType]
	if !ok {
		p = projectPresets["other"]
	}

	var sb strings.Builder
	sb.WriteString("# DIRECTION\n\n")
	sb.WriteString(fmt.Sprintf("Project type: **%s**\n\n", ProjectTypeLabel(projectType)))

	// Always Save
	sb.WriteString("## Always Save\n\n")
	sb.WriteString("These types of information should always be captured as memories:\n\n")
	for _, item := range p.AlwaysSave {
		sb.WriteString(fmt.Sprintf("- %s\n", item))
	}
	sb.WriteString("\n")

	// Never Save
	sb.WriteString("## Never Save\n\n")
	sb.WriteString("These should not be saved as memories:\n\n")
	for _, item := range p.NeverSave {
		sb.WriteString(fmt.Sprintf("- %s\n", item))
	}
	sb.WriteString("\n")

	// Type Guidance
	sb.WriteString("## Type Guidance\n\n")
	for _, item := range p.Guidance {
		sb.WriteString(fmt.Sprintf("- %s\n", item))
	}
	sb.WriteString("\n")

	// Priority Guidance
	if len(priorities) > 0 {
		sb.WriteString("## Priority Guidance\n\n")
		for _, pri := range priorities {
			if items, ok := priorityGuidance[pri]; ok {
				sb.WriteString(fmt.Sprintf("### %s\n\n", cases.Title(language.English).String(strings.ReplaceAll(pri, "-", " "))))
				for _, item := range items {
					sb.WriteString(fmt.Sprintf("- %s\n", item))
				}
				sb.WriteString("\n")
			}
		}
	}

	return sb.String()
}

func ProjectTypeLabel(slug string) string {
	labels := map[string]string{
		"web-api":    "Web API",
		"frontend":   "Frontend",
		"firmware":   "Firmware",
		"cli":        "CLI Tool",
		"full-stack": "Full-stack",
		"other":      "Other",
	}
	if label, ok := labels[slug]; ok {
		return label
	}
	return slug
}
