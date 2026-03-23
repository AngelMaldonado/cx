package instructions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AngelMaldonado/cx/internal/config"
	"github.com/AngelMaldonado/cx/internal/templates"
)

// Build returns a formatted multi-section string for a given artifact.
func Build(rootDir, artifact string) (string, error) {
	// Validate artifact exists in graph
	var found bool
	for _, a := range ArtifactGraph {
		if a.ID == artifact {
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("unknown artifact %q — valid artifacts: proposal, specs, design, tasks, verify", artifact)
	}

	var sections []string

	// 1. Template
	templatePath := "docs/" + artifact + ".md"
	if artifact == "specs" {
		templatePath = "docs/delta-spec.md"
	}
	tmpl, err := templates.Content(templatePath)
	if err != nil {
		return "", fmt.Errorf("loading template for %s: %w", artifact, err)
	}
	sections = append(sections, fmt.Sprintf("## Template\n\n%s", tmpl))

	// 2. Project context from cx.yaml
	cfg, err := config.Load(rootDir)
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	if cfg.Context != "" {
		sections = append(sections, fmt.Sprintf("## Project Context\n\n%s", strings.TrimSpace(cfg.Context)))
	}

	// 3. Rules for this artifact
	if rules, ok := cfg.Rules[artifact]; ok && len(rules) > 0 {
		var ruleLines []string
		for _, r := range rules {
			ruleLines = append(ruleLines, "- "+r)
		}
		sections = append(sections, fmt.Sprintf("## Rules for %s\n\n%s", artifact, strings.Join(ruleLines, "\n")))
	}

	// 4. Dependencies
	deps := DependenciesOf(artifact)
	if len(deps) > 0 {
		sections = append(sections, fmt.Sprintf("## Dependencies\n\nThis artifact requires: %s", strings.Join(deps, ", ")))
	} else {
		sections = append(sections, "## Dependencies\n\nThis artifact has no dependencies — it can be created first.")
	}

	// 5. What this unlocks
	unlocks := UnlocksOf(artifact)
	if len(unlocks) > 0 {
		sections = append(sections, fmt.Sprintf("## Unlocks\n\nCompleting this artifact enables: %s", strings.Join(unlocks, ", ")))
	}

	// 6. Spec index
	indexPath := filepath.Join(rootDir, "docs", "specs", "index.md")
	indexData, err := os.ReadFile(indexPath)
	if err != nil {
		sections = append(sections, "## Spec Index\n\n(no specs found — run cx init)")
	} else {
		sections = append(sections, fmt.Sprintf("## Spec Index\n\n%s", strings.TrimSpace(string(indexData))))
	}

	return strings.Join(sections, "\n\n---\n\n"), nil
}
