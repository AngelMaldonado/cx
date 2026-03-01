package agents

import (
	"bytes"
	"path/filepath"
	"strings"

	"github.com/amald/cx/internal/skills"
)

func WriteSkills(rootDir string, agent Agent) (int, error) {
	names := skills.Names()
	written := 0

	for _, name := range names {
		content, err := skills.Content(name)
		if err != nil {
			return written, err
		}
		dest := filepath.Join(rootDir, agent.SkillsDir, name)
		if err := atomicWriteAgent(dest, content); err != nil {
			return written, err
		}
		written++
	}

	return written, nil
}

func SkillMatchesEmbedded(onDisk []byte, name string) bool {
	embedded, err := skills.Content(name)
	if err != nil {
		return false
	}
	return bytes.Equal(onDisk, embedded)
}

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
