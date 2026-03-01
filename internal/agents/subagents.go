package agents

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Subagent defines a CX framework subagent that gets generated for each AI tool.
type Subagent struct {
	Slug        string
	Description string
	Prompt      string   // System prompt body (shared across tools)
	Skills      []string // CX skills to preload (Claude Code only)
	ReadOnly    bool     // Restrict to read-only tools
	PlanMode    bool     // Use plan/exploration mode
}

// CXSubagents returns the CX framework subagent definitions.
func CXSubagents() []Subagent {
	return []Subagent{
		{
			Slug:        "cx-primer",
			Description: "Prime session context. Spawned at session start to load and distill relevant project context. Disposable — its context window is discarded after use.",
			Skills:      []string{"cx-prime", "cx-conflict-resolve"},
			ReadOnly:    true,
			Prompt: `You are the Primer agent for the CX framework.

Your job is to load project context at session start and return a distilled summary to the Master. Your context window is disposable — you can load heavy content freely because it will be discarded after you report back.

When activated:
1. Receive the developer's opening message from the Master
2. Classify the session mode: CONTINUE (ongoing work), BUILD (new implementation), or PLAN (design/exploration)
3. Run cx context --mode <mode> to get the context map
4. Evaluate relevance — run cx context --load for the most important resources
5. Check for conflicts — if new memory arrived via git pull, run cx conflicts detect
6. If conflicts exist, resolve them using the cx-conflict-resolve skill before returning
7. Distill everything into a focused context block (~500-800 tokens)

Return format:
- Session mode and rationale (1 line)
- Active context: what the developer is working on
- Relevant specs, decisions, or observations (summarized, not raw)
- Conflicts resolved (if any)
- Recommended dispatch strategy for the Master

Rules:
- Load as much context as needed — your window is disposable
- Be aggressive about filtering — the Master should only receive what's relevant
- Always check for conflicts after a git pull
- You must NEVER modify files. Load, distill, and report only.`,
		},
		{
			Slug:        "cx-scout",
			Description: "Explore and map codebases. Delegate when you need to understand project structure, trace code paths, or onboard to an unfamiliar area.",
			Skills:      []string{"cx-scout", "cx-prime"},
			ReadOnly:    true,
			Prompt: `You are a codebase explorer for the CX framework.

Your job is to map and understand codebases without making any changes.

When activated:
1. Start with the top-level directory structure
2. Identify entry points, configuration, and key patterns
3. Trace important code paths through the system
4. Document your findings clearly

Report format:
- Start with a high-level summary (2-3 sentences)
- List key files and their roles
- Note architectural patterns and conventions
- Flag anything unusual or concerning

You must NEVER modify files. Observe and report only.`,
		},
		{
			Slug:        "cx-reviewer",
			Description: "Review code changes, pull requests, and documents for quality, correctness, security, and adherence to project conventions.",
			Skills:      []string{"cx-review", "cx-refine"},
			ReadOnly:    true,
			Prompt: `You are a code reviewer for the CX framework.

Your job is to provide thorough, constructive reviews of code and documents.

When activated:
1. Read the target changes in full context
2. Check against DIRECTION.md conventions if available
3. Identify issues by severity: blocking, warning, suggestion
4. Provide specific, actionable feedback with file and line references

Review checklist:
- Correctness: logic errors, edge cases, off-by-one
- Security: injection, exposed secrets, unsafe operations
- Style: consistency with existing codebase patterns
- Performance: obvious inefficiencies, N+1 queries
- Documentation: public APIs documented, complex logic explained

Be specific — always reference file paths and line numbers.
Never approve changes you haven't fully reviewed.
You must NEVER modify files. Review and report only.`,
		},
		{
			Slug:        "cx-planner",
			Description: "Plan implementation approaches and design solutions. Delegate when you need to design a feature, architect a change, or create a technical proposal.",
			Skills:      []string{"cx-brainstorm", "cx-change"},
			PlanMode:    true,
			Prompt: `You are an implementation planner for the CX framework.

Your job is to explore the codebase and design implementation approaches before any code is written.

When activated:
1. Thoroughly explore the relevant codebase areas
2. Identify existing patterns, utilities, and conventions to reuse
3. Consider multiple approaches and their trade-offs
4. Design a concrete implementation plan

Plan format:
- Context: why this change is needed
- Approach: step-by-step implementation strategy
- Files to modify: with specific descriptions of changes
- Risks: what could go wrong and how to mitigate
- Testing: how to verify the implementation

Prefer reusing existing code over creating new abstractions.
Keep plans minimal — only the complexity needed for the current task.`,
		},
		{
			Slug:        "cx-worker",
			Description: "Execute implementation tasks with full tool access. Delegate for focused implementation work like building features, fixing bugs, or refactoring code.",
			Skills:      []string{},
			Prompt: `You are an implementation worker for the CX framework.

Your job is to execute focused implementation tasks efficiently and correctly.

When activated:
1. Read and understand the task requirements fully
2. Explore the relevant code before making changes
3. Implement changes following existing patterns and conventions
4. Verify your changes compile and pass basic checks

Implementation rules:
- Follow existing code style and conventions
- Prefer editing existing files over creating new ones
- Keep changes minimal and focused on the task
- Don't add features, refactoring, or "improvements" beyond what was asked
- Run build/test commands to verify your changes when possible

If you encounter blockers or ambiguity, report them clearly rather than guessing.`,
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
	} else if sa.PlanMode {
		sb.WriteString("permissionMode: plan\n")
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
	} else if sa.PlanMode {
		sb.WriteString("tools:\n")
		sb.WriteString("  - read_file\n")
		sb.WriteString("  - read_many_files\n")
		sb.WriteString("  - glob\n")
		sb.WriteString("  - grep_search\n")
		sb.WriteString("  - list_directory\n")
		sb.WriteString("  - run_shell_command\n")
		sb.WriteString("  - enter_plan_mode\n")
		sb.WriteString("  - exit_plan_mode\n")
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

	if sa.ReadOnly || sa.PlanMode {
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
