package agents

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/AngelMaldonado/cx/internal/skills"
)

// WriteSkills writes all embedded skills to the agent's skills directory
// using the <skill-name>/SKILL.md convention. Always overwrites.
func WriteSkills(rootDir string, agent Agent) (int, error) {
	names := skills.Names()
	written := 0

	for _, name := range names {
		content, err := skills.Content(name)
		if err != nil {
			return written, err
		}
		slug := strings.TrimSuffix(name, ".md")
		skillDir := filepath.Join(rootDir, agent.SkillsDir, slug)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			return written, err
		}
		dest := filepath.Join(skillDir, "SKILL.md")
		if err := atomicWriteAgent(dest, content); err != nil {
			return written, err
		}
		written++
	}

	return written, nil
}

// SkillMatchesEmbedded checks if the on-disk SKILL.md matches the embedded version.
func SkillMatchesEmbedded(onDisk []byte, slug string) bool {
	embedded, err := skills.Content(slug + ".md")
	if err != nil {
		return false
	}
	return bytes.Equal(onDisk, embedded)
}

// ValidateSkillSections checks that a SKILL.md contains all required sections.
func ValidateSkillSections(content []byte) []string {
	required := []string{"## Description", "## Triggers", "## Steps", "## Rules"}
	text := string(content)
	var missing []string
	for _, section := range required {
		if !strings.Contains(text, section) {
			missing = append(missing, section)
		}
	}
	return missing
}
