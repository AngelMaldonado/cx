package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Agent struct {
	Slug       string
	Name       string
	Dir        string
	ConfigFile string
	SkillsDir  string
}

func All() []Agent {
	return []Agent{
		{
			Slug:       "claude",
			Name:       "Claude Code",
			Dir:        ".claude",
			ConfigFile: "CLAUDE.md",
			SkillsDir:  ".claude/skills",
		},
		{
			Slug:       "gemini",
			Name:       "Gemini CLI",
			Dir:        ".gemini",
			ConfigFile: "GEMINI.md",
			SkillsDir:  ".gemini/skills",
		},
		{
			Slug:       "codex",
			Name:       "Codex CLI",
			Dir:        ".codex",
			ConfigFile: "AGENTS.md",
			SkillsDir:  ".codex/skills",
		},
	}
}

func BySlug(slug string) (Agent, bool) {
	for _, a := range All() {
		if a.Slug == slug {
			return a, true
		}
	}
	return Agent{}, false
}

func DetectInstalled(rootDir string) []Agent {
	var found []Agent
	for _, a := range All() {
		dir := filepath.Join(rootDir, a.Dir)
		if _, err := os.Stat(dir); err == nil {
			found = append(found, a)
		}
	}
	return found
}

func EnsureAgentDir(rootDir string, agent Agent) error {
	dir := filepath.Join(rootDir, agent.Dir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", agent.Dir, err)
	}
	skillsDir := filepath.Join(rootDir, agent.SkillsDir)
	if err := os.MkdirAll(skillsDir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", agent.SkillsDir, err)
	}
	return nil
}

func WriteConfigFile(rootDir string, agent Agent) error {
	configPath := filepath.Join(rootDir, agent.ConfigFile)

	skillNames := skillFileNames()
	table := buildSkillTable(skillNames)

	content := fmt.Sprintf(`# %s Configuration

## Skills

%s

## Usage

Skills are located in `+"`%s/`"+`. Each skill file defines:
- **Description**: What the skill does
- **Triggers**: When to activate the skill
- **Steps**: How to execute the skill
- **Rules**: Constraints and guidelines
`, agent.Name, table, agent.SkillsDir)

	return atomicWriteAgent(configPath, []byte(content))
}

func buildSkillTable(names []string) string {
	var sb strings.Builder
	sb.WriteString("| Skill | File |\n")
	sb.WriteString("|-------|------|\n")
	for _, name := range names {
		slug := strings.TrimSuffix(name, ".md")
		sb.WriteString(fmt.Sprintf("| %s | [%s](skills/%s) |\n", slug, name, name))
	}
	return sb.String()
}

func skillFileNames() []string {
	return []string{
		"cx-brainstorm.md",
		"cx-change.md",
		"cx-conflict-resolve.md",
		"cx-contract.md",
		"cx-doctor.md",
		"cx-linear.md",
		"cx-memory.md",
		"cx-prime.md",
		"cx-refine.md",
		"cx-review.md",
		"cx-scout.md",
		"cx-supervise.md",
	}
}

func atomicWriteAgent(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
