package agents

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amald/cx/internal/skills"
)

// SyncStatus reports what would change for a single agent during sync.
type SyncStatus struct {
	Agent          Agent
	ConfigChanged  bool
	SkillsChanged  []string // slugs of skills that differ from embedded
	SkillsMissing  []string // slugs of skills not on disk
	SubagentsChanged []string // slugs of subagents that differ
	SubagentsMissing []string // slugs of subagents not on disk
}

// UpToDate returns true if nothing needs syncing for this agent.
func (s SyncStatus) UpToDate() bool {
	return !s.ConfigChanged &&
		len(s.SkillsChanged) == 0 &&
		len(s.SkillsMissing) == 0 &&
		len(s.SubagentsChanged) == 0 &&
		len(s.SubagentsMissing) == 0
}

// Summary returns a human-readable summary of changes.
func (s SyncStatus) Summary() string {
	var parts []string
	if s.ConfigChanged {
		parts = append(parts, "config")
	}
	changed := len(s.SkillsChanged) + len(s.SkillsMissing)
	if changed > 0 {
		parts = append(parts, fmt.Sprintf("%d skills", changed))
	}
	saChanged := len(s.SubagentsChanged) + len(s.SubagentsMissing)
	if saChanged > 0 {
		parts = append(parts, fmt.Sprintf("%d subagents", saChanged))
	}
	if len(parts) == 0 {
		return "up to date"
	}
	return strings.Join(parts, " + ") + " updated"
}

// CheckSyncStatus compares on-disk state to what would be generated for each agent.
func CheckSyncStatus(rootDir string, installed []Agent) []SyncStatus {
	var results []SyncStatus
	for _, agent := range installed {
		results = append(results, checkAgentSync(rootDir, agent))
	}
	return results
}

func checkAgentSync(rootDir string, agent Agent) SyncStatus {
	status := SyncStatus{Agent: agent}

	// Check config file
	status.ConfigChanged = configDrifted(rootDir, agent)

	// Check skills
	status.SkillsChanged, status.SkillsMissing = skillsDrifted(rootDir, agent)

	// Check subagents
	if agent.AgentsDir != "" {
		status.SubagentsChanged, status.SubagentsMissing = subagentsDrifted(rootDir, agent)
	}

	return status
}

func configDrifted(rootDir string, agent Agent) bool {
	configPath := filepath.Join(rootDir, agent.ConfigFile)
	onDisk, err := os.ReadFile(configPath)
	if err != nil {
		return true // missing = needs sync
	}

	// Generate what the config would look like
	slugs := skills.Slugs()
	skillTable := buildSkillTable(slugs)
	expected := generateConfigContent(agent, skillTable)

	return !bytes.Equal(onDisk, []byte(expected))
}

func skillsDrifted(rootDir string, agent Agent) (changed, missing []string) {
	names := skills.Names()
	for _, name := range names {
		slug := strings.TrimSuffix(name, ".md")
		dest := filepath.Join(rootDir, agent.SkillsDir, slug, "SKILL.md")
		onDisk, err := os.ReadFile(dest)
		if err != nil {
			missing = append(missing, slug)
			continue
		}
		if !SkillMatchesEmbedded(onDisk, slug) {
			changed = append(changed, slug)
		}
	}
	return
}

func subagentsDrifted(rootDir string, agent Agent) (changed, missing []string) {
	subagents := CXSubagents()
	agentsDir := filepath.Join(rootDir, agent.AgentsDir)

	for _, sa := range subagents {
		var expectedContent string
		var ext string
		switch agent.Slug {
		case "claude":
			expectedContent = renderClaudeAgent(sa)
			ext = ".md"
		case "gemini":
			expectedContent = renderGeminiAgent(sa)
			ext = ".md"
		case "codex":
			expectedContent = renderCodexAgentToml(sa)
			ext = ".toml"
		default:
			continue
		}

		dest := filepath.Join(agentsDir, sa.Slug+ext)
		onDisk, err := os.ReadFile(dest)
		if err != nil {
			missing = append(missing, sa.Slug)
			continue
		}
		if !bytes.Equal(onDisk, []byte(expectedContent)) {
			changed = append(changed, sa.Slug)
		}
	}
	return
}
