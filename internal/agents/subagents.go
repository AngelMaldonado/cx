package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AngelMaldonado/cx/internal/templates"
)

// Subagent defines a CX framework subagent that gets generated for each AI tool.
type Subagent struct {
	Slug        string
	Description string
	Prompt      string   // System prompt body (shared across tools)
	Skills      []string // CX skills to preload (Claude Code only)
	ReadOnly    bool     // Restrict to read-only tools
}

// CXSubagents returns the CX framework subagent definitions.
func CXSubagents() []Subagent {
	return []Subagent{
		{
			Slug:        "cx-primer",
			Description: "Prime session context. Spawned at session start to load and distill relevant project context. Disposable — its context window is discarded after use.",
			Skills:      []string{"cx-prime", "cx-conflict-resolve"},
			ReadOnly:    true,
			Prompt:      templates.MustContent("subagents/cx-primer.md"),
		},
		{
			Slug:        "cx-scout",
			Description: "Explore and map codebases. Delegate when you need to understand project structure, trace code paths, or onboard to an unfamiliar area.",
			Skills:      []string{"cx-scout", "cx-prime"},
			ReadOnly:    true,
			Prompt:      templates.MustContent("subagents/cx-scout.md"),
		},
		{
			Slug:        "cx-reviewer",
			Description: "Review code changes, pull requests, and documents for quality, correctness, security, and adherence to project conventions.",
			Skills:      []string{"cx-review", "cx-refine"},
			ReadOnly:    true,
			Prompt:      templates.MustContent("subagents/cx-reviewer.md"),
		},
		{
			Slug:        "cx-planner",
			Description: "Plan implementation approaches and design solutions. Delegate when you need to design a feature, architect a change, or create a technical proposal.",
			Skills:      []string{"cx-brainstorm", "cx-change"},
			Prompt:      templates.MustContent("subagents/cx-planner.md"),
		},
		{
			Slug:        "cx-executor",
			Description: "Implement tasks from change docs. Delegate when you need to write code, run tests, or apply a specific task from tasks.md.",
			Prompt:      templates.MustContent("subagents/cx-executor.md"),
		},
		{
			Slug:        "cx-merger",
			Description: "Integrate multiple task branches into a single change branch after parallel executor work. Delegate after parallel executors complete to merge their worktrees and resolve conflicts.",
			Prompt:      templates.MustContent("subagents/cx-merger.md"),
		},
	}
}

// WriteSubagents writes CX subagent definitions for the given agent tool.
// For Claude/Gemini this writes markdown files to the agents directory.
// For Codex this writes per-agent .toml files plus a config.toml with [agents.*] declarations.
// Returns the number of subagents written.
func WriteSubagents(rootDir string, agent Agent) (int, error) {
	if agent.AgentsDir == "" {
		return 0, nil
	}

	agentsDir := filepath.Join(rootDir, agent.AgentsDir)
	if err := os.MkdirAll(agentsDir, 0o755); err != nil {
		return 0, err
	}

	subagents := CXSubagents()
	written := 0

	for _, sa := range subagents {
		var content string
		var ext string
		switch agent.Slug {
		case "claude":
			content = renderClaudeAgent(sa)
			ext = ".md"
		case "gemini":
			content = renderGeminiAgent(sa)
			ext = ".md"
		case "codex":
			content = renderCodexAgentToml(sa)
			ext = ".toml"
		default:
			continue
		}
		dest := filepath.Join(agentsDir, sa.Slug+ext)
		if err := atomicWriteAgent(dest, []byte(content)); err != nil {
			return written, err
		}
		written++
	}

	// Codex also needs a config.toml with [features] and [agents.*] declarations
	if agent.Slug == "codex" {
		configContent := renderCodexConfigToml(subagents)
		dest := filepath.Join(rootDir, agent.Dir, "config.toml")
		if err := atomicWriteAgent(dest, []byte(configContent)); err != nil {
			return written, err
		}
	}

	return written, nil
}

// SubagentSlugs returns the slugs of all CX subagents.
func SubagentSlugs() []string {
	subs := CXSubagents()
	slugs := make([]string, len(subs))
	for i, s := range subs {
		slugs[i] = s.Slug
	}
	return slugs
}

func renderClaudeAgent(sa Subagent) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", sa.Slug))
	sb.WriteString(fmt.Sprintf("description: %s\n", sa.Description))

	if sa.ReadOnly {
		sb.WriteString("tools: Read, Glob, Grep, Bash\n")
		sb.WriteString("disallowedTools: Write, Edit, MultiEdit, NotebookEdit\n")
	}

	sb.WriteString("model: sonnet\n")

	if len(sa.Skills) > 0 {
		sb.WriteString("skills:\n")
		for _, skill := range sa.Skills {
			sb.WriteString(fmt.Sprintf("  - %s\n", skill))
		}
	}

	sb.WriteString("---\n\n")
	sb.WriteString(sa.Prompt)
	sb.WriteString("\n")
	return sb.String()
}

func renderGeminiAgent(sa Subagent) string {
	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", sa.Slug))
	sb.WriteString(fmt.Sprintf("description: %s\n", sa.Description))

	if sa.ReadOnly {
		sb.WriteString("tools:\n")
		sb.WriteString("  - read_file\n")
		sb.WriteString("  - read_many_files\n")
		sb.WriteString("  - glob\n")
		sb.WriteString("  - grep_search\n")
		sb.WriteString("  - list_directory\n")
		sb.WriteString("  - run_shell_command\n")
	}
	// Full-access agents omit tools field to inherit all defaults

	sb.WriteString("model: inherit\n")
	sb.WriteString("max_turns: 25\n")
	sb.WriteString("timeout_mins: 10\n")
	sb.WriteString("---\n\n")
	sb.WriteString(sa.Prompt)
	sb.WriteString("\n")
	return sb.String()
}

// renderCodexAgentToml renders a per-agent .toml file for Codex CLI.
func renderCodexAgentToml(sa Subagent) string {
	var sb strings.Builder

	if sa.ReadOnly {
		sb.WriteString("sandbox_mode = \"read-only\"\n")
	} else {
		sb.WriteString("sandbox_mode = \"workspace-write\"\n")
	}

	if sa.ReadOnly {
		sb.WriteString("model_reasoning_effort = \"medium\"\n")
	} else {
		sb.WriteString("model_reasoning_effort = \"high\"\n")
	}

	sb.WriteString(fmt.Sprintf("developer_instructions = \"\"\"\n%s\n\"\"\"\n", sa.Prompt))
	return sb.String()
}

// renderCodexConfigToml renders the main .codex/config.toml with [features]
// and [agents.*] declarations pointing to the per-agent .toml files.
func renderCodexConfigToml(subagents []Subagent) string {
	var sb strings.Builder

	sb.WriteString("# CX Framework — Codex CLI configuration\n")
	sb.WriteString("# Generated by cx init\n\n")

	sb.WriteString("[features]\n")
	sb.WriteString("multi_agent = true\n\n")

	sb.WriteString("[agents]\n")
	sb.WriteString(fmt.Sprintf("max_threads = %d\n", len(subagents)))
	sb.WriteString("max_depth = 1\n\n")

	for _, sa := range subagents {
		// TOML keys can't contain hyphens unquoted, so use the slug as-is in quotes
		sb.WriteString(fmt.Sprintf("[agents.\"%s\"]\n", sa.Slug))
		sb.WriteString(fmt.Sprintf("description = %q\n", sa.Description))
		sb.WriteString(fmt.Sprintf("config_file = \"agents/%s.toml\"\n\n", sa.Slug))
	}

	return sb.String()
}
