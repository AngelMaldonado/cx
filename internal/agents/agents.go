package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amald/cx/internal/skills"
)

type Agent struct {
	Slug       string
	Name       string
	Dir        string
	ConfigFile string
	SkillsDir  string
	AgentsDir  string // empty if tool doesn't support subagent definitions
}

func All() []Agent {
	return []Agent{
		{
			Slug:       "claude",
			Name:       "Claude Code",
			Dir:        ".claude",
			ConfigFile: "CLAUDE.md",
			SkillsDir:  ".claude/skills",
			AgentsDir:  ".claude/agents",
		},
		{
			Slug:       "gemini",
			Name:       "Gemini CLI",
			Dir:        ".gemini",
			ConfigFile: "GEMINI.md",
			SkillsDir:  ".gemini/skills",
			AgentsDir:  ".gemini/agents",
		},
		{
			Slug:       "codex",
			Name:       "Codex CLI",
			Dir:        ".codex",
			ConfigFile: "AGENTS.md",
			SkillsDir:  ".codex/skills",
			AgentsDir:  ".codex/agents",
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
	if agent.AgentsDir != "" {
		agentsDir := filepath.Join(rootDir, agent.AgentsDir)
		if err := os.MkdirAll(agentsDir, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", agent.AgentsDir, err)
		}
	}
	return nil
}

func WriteConfigFile(rootDir string, agent Agent) error {
	configPath := filepath.Join(rootDir, agent.ConfigFile)

	slugs := skills.Slugs()
	table := buildSkillTable(slugs)

	content := fmt.Sprintf(`# %s Configuration

## Skills

%s

## Usage

Each skill is a directory in `+"`%s/`"+` containing a `+"`SKILL.md`"+` file with:
- **YAML frontmatter**: name and description (used for auto-invocation)
- **Description**: What the skill does
- **Triggers**: When to activate the skill
- **Steps**: How to execute the skill
- **Rules**: Constraints and guidelines
`, agent.Name, table, agent.SkillsDir)

	return atomicWriteAgent(configPath, []byte(content))
}

func buildSkillTable(slugs []string) string {
	var sb strings.Builder
	sb.WriteString("| Skill | Path |\n")
	sb.WriteString("|-------|------|\n")
	for _, slug := range slugs {
		sb.WriteString(fmt.Sprintf("| %s | [SKILL.md](skills/%s/SKILL.md) |\n", slug, slug))
	}
	return sb.String()
}

func atomicWriteAgent(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
