package verify

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/amald/cx/internal/templates"
)

var reqPattern = regexp.MustCompile(`^### (REQ-[A-Za-z0-9-]+):`)

// BuildPrompt reads delta specs and change docs to produce a structured
// verification prompt.
func BuildPrompt(rootDir, changeName string) (string, error) {
	changeDir := filepath.Join(rootDir, "docs", "changes", changeName)

	// Read proposal and design for intent context
	proposal, _ := os.ReadFile(filepath.Join(changeDir, "proposal.md"))
	design, _ := os.ReadFile(filepath.Join(changeDir, "design.md"))

	// Extract REQ-NNN lines from delta specs
	var reqs []string
	specsDir := filepath.Join(changeDir, "specs")
	if areas, err := os.ReadDir(specsDir); err == nil {
		for _, area := range areas {
			if !area.IsDir() {
				continue
			}
			specPath := filepath.Join(specsDir, area.Name(), "spec.md")
			data, err := os.ReadFile(specPath)
			if err != nil {
				continue
			}
			for _, line := range strings.Split(string(data), "\n") {
				if matches := reqPattern.FindStringSubmatch(line); len(matches) > 1 {
					reqs = append(reqs, matches[1])
				}
			}
		}
	}

	// Build the prompt
	var sections []string

	// Completeness checklist
	if len(reqs) > 0 {
		var items []string
		for _, req := range reqs {
			items = append(items, fmt.Sprintf("- [ ] %s", req))
		}
		sections = append(sections, fmt.Sprintf("## Completeness\n\nVerify each requirement is implemented:\n\n%s", strings.Join(items, "\n")))
	} else {
		sections = append(sections, "## Completeness\n\nNo REQ-NNN requirements found in delta specs. Verify implementation covers all behaviors described in the change docs.")
	}

	// Correctness
	sections = append(sections, "## Correctness\n\nVerify the implementation matches the intent described in the proposal and design:\n- Does it solve the stated problem?\n- Are edge cases handled?\n- Are error paths correct?")

	// Coherence
	sections = append(sections, "## Coherence\n\nVerify design decisions are reflected in the code:\n- Are architectural patterns consistent?\n- Do naming conventions match?\n- Is the implementation aligned with the technical decisions?")

	// Context
	if len(proposal) > 0 {
		sections = append(sections, fmt.Sprintf("## Proposal Context\n\n%s", strings.TrimSpace(string(proposal))))
	}
	if len(design) > 0 {
		sections = append(sections, fmt.Sprintf("## Design Context\n\n%s", strings.TrimSpace(string(design))))
	}

	return strings.Join(sections, "\n\n---\n\n"), nil
}

// Record writes a verify.md stub to the change directory. Skips if already exists.
func Record(rootDir, changeName string) error {
	changeDir := filepath.Join(rootDir, "docs", "changes", changeName)
	verifyPath := filepath.Join(changeDir, "verify.md")

	if _, err := os.Stat(verifyPath); err == nil {
		return nil // already exists, skip
	}

	tmpl, err := templates.Content("docs/verify.md")
	if err != nil {
		return fmt.Errorf("loading verify template: %w", err)
	}

	content := strings.ReplaceAll(tmpl, "{{name}}", changeName)

	// atomic write pattern from internal/change
	tmp := verifyPath + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, verifyPath)
}
