package doctor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/amald/cx/internal/agents"
	"github.com/amald/cx/internal/config"
	"github.com/amald/cx/internal/project"
)

type Severity int

const (
	Pass Severity = iota
	Warning
	Error
)

type CheckResult struct {
	Name     string
	Severity Severity
	Message  string
	Fixable  bool
	FixLabel string
	FixFunc  func() error
}

type CheckGroup struct {
	Name    string
	Results []CheckResult
}

func RunAllChecks(rootDir string) []CheckGroup {
	return []CheckGroup{
		CheckDocsStructure(rootDir),
		CheckMemoryHealth(rootDir),
		CheckIndexHealth(rootDir),
		CheckGitHooks(rootDir),
		CheckMCPConfig(rootDir),
		CheckSkillFiles(rootDir),
		CheckSubagentFiles(rootDir),
	}
}

func CheckDocsStructure(rootDir string) CheckGroup {
	group := CheckGroup{Name: "docs/ structure"}

	requiredDirs := []string{
		"docs",
		"docs/architecture",
		"docs/specs",
		"docs/memory",
		"docs/memory/observations",
		"docs/memory/decisions",
		"docs/memory/sessions",
		"docs/changes",
	}

	for _, dir := range requiredDirs {
		full := filepath.Join(rootDir, dir)
		if _, err := os.Stat(full); os.IsNotExist(err) {
			d := dir
			group.Results = append(group.Results, CheckResult{
				Name:     dir,
				Severity: Error,
				Message:  fmt.Sprintf("missing directory: %s", dir),
				Fixable:  true,
				FixLabel: fmt.Sprintf("create %s/", dir),
				FixFunc: func() error {
					return os.MkdirAll(filepath.Join(rootDir, d), 0o755)
				},
			})
		} else {
			group.Results = append(group.Results, CheckResult{
				Name:     dir,
				Severity: Pass,
				Message:  fmt.Sprintf("%s exists", dir),
			})
		}
	}

	// Check overview.md exists and has H1
	overviewPath := filepath.Join(rootDir, "docs/overview.md")
	if data, err := os.ReadFile(overviewPath); err != nil {
		group.Results = append(group.Results, CheckResult{
			Name:     "docs/overview.md",
			Severity: Warning,
			Message:  "docs/overview.md not found",
			Fixable:  true,
			FixLabel: "create docs/overview.md from template",
			FixFunc: func() error {
				return os.WriteFile(overviewPath, []byte(project.OverviewTemplate()), 0o644)
			},
		})
	} else if !strings.HasPrefix(string(data), "# ") {
		group.Results = append(group.Results, CheckResult{
			Name:     "docs/overview.md",
			Severity: Warning,
			Message:  "docs/overview.md missing H1 heading",
		})
	} else {
		group.Results = append(group.Results, CheckResult{
			Name:     "docs/overview.md",
			Severity: Pass,
			Message:  "docs/overview.md valid",
		})
	}

	// Check DIRECTION.md
	directionPath := filepath.Join(rootDir, "docs", "memory", "DIRECTION.md")
	if _, err := os.Stat(directionPath); os.IsNotExist(err) {
		group.Results = append(group.Results, CheckResult{
			Name:     "DIRECTION.md",
			Severity: Warning,
			Message:  "docs/memory/DIRECTION.md not found — run cx init to generate",
			Fixable:  true,
			FixLabel: "create docs/memory/DIRECTION.md from template",
			FixFunc: func() error {
				tmpl := "# DIRECTION\n\n<!-- Run cx init to generate project-specific guidance -->\n"
				return os.WriteFile(directionPath, []byte(tmpl), 0o644)
			},
		})
	} else {
		group.Results = append(group.Results, CheckResult{
			Name:     "DIRECTION.md",
			Severity: Pass,
			Message:  "docs/memory/DIRECTION.md exists",
		})
	}

	// Check cx.yaml if present (optional)
	cxYamlPath := filepath.Join(rootDir, ".cx", "cx.yaml")
	if _, statErr := os.Stat(cxYamlPath); statErr == nil {
		_, loadErr := config.Load(rootDir)
		if loadErr != nil {
			group.Results = append(group.Results, CheckResult{
				Name:     "cx.yaml",
				Severity: Warning,
				Message:  fmt.Sprintf("cx.yaml: %v", loadErr),
				Fixable:  false,
			})
		} else {
			group.Results = append(group.Results, CheckResult{
				Name:     "cx.yaml",
				Severity: Pass,
				Message:  "cx.yaml valid structure",
			})
		}
	}

	return group
}

func CheckMemoryHealth(rootDir string) CheckGroup {
	group := CheckGroup{Name: "memory health"}

	memDirs := map[string]string{
		"observations": "docs/memory/observations",
		"decisions":    "docs/memory/decisions",
		"sessions":     "docs/memory/sessions",
	}

	for label, dir := range memDirs {
		full := filepath.Join(rootDir, dir)
		entries, err := os.ReadDir(full)
		if err != nil {
			group.Results = append(group.Results, CheckResult{
				Name:     label,
				Severity: Warning,
				Message:  fmt.Sprintf("%s directory not readable", dir),
			})
			continue
		}
		count := 0
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".md") {
				count++
			}
		}
		group.Results = append(group.Results, CheckResult{
			Name:     label,
			Severity: Pass,
			Message:  fmt.Sprintf("%d %s files", count, label),
		})
	}

	return group
}

func CheckIndexHealth(rootDir string) CheckGroup {
	group := CheckGroup{Name: "index health"}

	indexPath := filepath.Join(rootDir, ".cx", ".index.db")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		group.Results = append(group.Results, CheckResult{
			Name:     "FTS5 index",
			Severity: Warning,
			Message:  "search index not found (.cx/.index.db) — will be created on first search",
		})
	} else {
		group.Results = append(group.Results, CheckResult{
			Name:     "FTS5 index",
			Severity: Pass,
			Message:  "search index exists",
		})
	}

	return group
}

func CheckGitHooks(rootDir string) CheckGroup {
	group := CheckGroup{Name: "git hooks"}

	hooks := []string{"post-merge", "post-checkout"}
	for _, hookType := range hooks {
		hookPath := filepath.Join(project.HooksDir(rootDir), hookType)
		if _, err := os.Stat(hookPath); os.IsNotExist(err) {
			ht := hookType
			group.Results = append(group.Results, CheckResult{
				Name:     hookType,
				Severity: Warning,
				Message:  fmt.Sprintf("%s hook not installed", hookType),
				Fixable:  true,
				FixLabel: fmt.Sprintf("install %s hook", hookType),
				FixFunc: func() error {
					_, err := project.InstallHook(rootDir, ht, true)
					return err
				},
			})
		} else {
			hasCX, _ := project.HookContainsCX(rootDir, hookType)
			if hasCX {
				group.Results = append(group.Results, CheckResult{
					Name:     hookType,
					Severity: Pass,
					Message:  fmt.Sprintf("%s hook installed with CX marker", hookType),
				})
			} else {
				group.Results = append(group.Results, CheckResult{
					Name:     hookType,
					Severity: Warning,
					Message:  fmt.Sprintf("%s hook exists but missing CX marker", hookType),
				})
			}
		}
	}

	return group
}

func CheckMCPConfig(rootDir string) CheckGroup {
	group := CheckGroup{Name: "MCP config"}

	hasMCP, missing, err := project.CheckMCP(rootDir)
	if err != nil {
		group.Results = append(group.Results, CheckResult{
			Name:     ".mcp.json",
			Severity: Error,
			Message:  fmt.Sprintf("error reading .mcp.json: %v", err),
		})
		return group
	}

	if !hasMCP {
		group.Results = append(group.Results, CheckResult{
			Name:     ".mcp.json",
			Severity: Warning,
			Message:  ".mcp.json not found — run cx init to configure",
			Fixable:  true,
			FixLabel: "create .mcp.json with context7 + linear",
			FixFunc: func() error {
				return project.WriteMCPConfigs(rootDir, []string{"claude"})
			},
		})
	} else {
		group.Results = append(group.Results, CheckResult{
			Name:     ".mcp.json",
			Severity: Pass,
			Message:  ".mcp.json exists",
		})
	}

	for _, server := range missing {
		group.Results = append(group.Results, CheckResult{
			Name:     server + " MCP",
			Severity: Warning,
			Message:  fmt.Sprintf("%s MCP server not configured in .mcp.json", server),
		})
	}

	return group
}

func CheckSkillFiles(rootDir string) CheckGroup {
	group := CheckGroup{Name: "skill files"}

	installed := agents.DetectInstalled(rootDir)
	if len(installed) == 0 {
		group.Results = append(group.Results, CheckResult{
			Name:     "agents",
			Severity: Warning,
			Message:  "no agent directories found — run cx init",
		})
		return group
	}

	for _, agent := range installed {
		skillsDir := filepath.Join(rootDir, agent.SkillsDir)
		entries, err := os.ReadDir(skillsDir)
		if err != nil {
			group.Results = append(group.Results, CheckResult{
				Name:     agent.Name + " skills",
				Severity: Warning,
				Message:  fmt.Sprintf("%s skills directory not readable", agent.Name),
			})
			continue
		}

		skillCount := 0
		driftCount := 0
		sectionIssues := 0

		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			slug := e.Name()
			skillFile := filepath.Join(skillsDir, slug, "SKILL.md")
			data, err := os.ReadFile(skillFile)
			if err != nil {
				continue
			}
			skillCount++

			// Check sections
			missing := agents.ValidateSkillSections(data)
			if len(missing) > 0 {
				sectionIssues++
			}

			// Check drift from embedded
			if !agents.SkillMatchesEmbedded(data, slug) {
				driftCount++
			}
		}

		if sectionIssues > 0 {
			group.Results = append(group.Results, CheckResult{
				Name:     agent.Name + " skill sections",
				Severity: Warning,
				Message:  fmt.Sprintf("%s: %d skill(s) missing required sections", agent.Name, sectionIssues),
			})
		}

		if driftCount > 0 {
			a := agent
			group.Results = append(group.Results, CheckResult{
				Name:     agent.Name + " skill sync",
				Severity: Warning,
				Message:  fmt.Sprintf("%s: %d skill(s) differ from embedded defaults", agent.Name, driftCount),
				Fixable:  true,
				FixLabel: fmt.Sprintf("sync %s skills to embedded defaults", agent.Name),
				FixFunc: func() error {
					_, err := agents.WriteSkills(rootDir, a)
					return err
				},
			})
		} else if skillCount > 0 {
			group.Results = append(group.Results, CheckResult{
				Name:     agent.Name + " skills",
				Severity: Pass,
				Message:  fmt.Sprintf("%s: %d skills, all in sync", agent.Name, skillCount),
			})
		}
	}

	return group
}

func CheckSubagentFiles(rootDir string) CheckGroup {
	group := CheckGroup{Name: "subagent files"}

	installed := agents.DetectInstalled(rootDir)
	if len(installed) == 0 {
		return group
	}

	expectedSlugs := agents.SubagentSlugs()

	for _, agent := range installed {
		if agent.AgentsDir == "" {
			continue
		}

		agentsDir := filepath.Join(rootDir, agent.AgentsDir)
		missingCount := 0
		presentCount := 0

		ext := ".md"
		if agent.Slug == "codex" {
			ext = ".toml"
		}

		for _, slug := range expectedSlugs {
			agentFile := filepath.Join(agentsDir, slug+ext)
			if _, err := os.Stat(agentFile); os.IsNotExist(err) {
				missingCount++
			} else {
				presentCount++
			}
		}

		if missingCount > 0 {
			a := agent
			group.Results = append(group.Results, CheckResult{
				Name:     agent.Name + " subagents",
				Severity: Warning,
				Message:  fmt.Sprintf("%s: %d/%d subagent(s) missing", agent.Name, missingCount, len(expectedSlugs)),
				Fixable:  true,
				FixLabel: fmt.Sprintf("sync %s subagents", agent.Name),
				FixFunc: func() error {
					_, err := agents.WriteSubagents(rootDir, a)
					return err
				},
			})
		} else if presentCount > 0 {
			group.Results = append(group.Results, CheckResult{
				Name:     agent.Name + " subagents",
				Severity: Pass,
				Message:  fmt.Sprintf("%s: %d subagents present", agent.Name, presentCount),
			})
		}
	}

	return group
}
