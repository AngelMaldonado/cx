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
	content := generateConfigContent(agent, buildSkillTable(slugs), buildSubagentTable())
	return atomicWriteAgent(configPath, []byte(content))
}

func generateConfigContent(agent Agent, skillTable, subagentTable string) string {
	return fmt.Sprintf(`# %s — CX Framework

You are the **Master agent** for the CX framework. You are a pure orchestrator — you never write code or modify files directly. Instead, you dispatch specialized subagents, each guided by skills, to do the work.

## Architecture

`+"```"+`
Developer → Master (you — pure orchestrator)
    │
    ├──→ Primer (context priming, disposable)
    │       └──→ Conflict-resolver (if conflicts detected after git pull)
    │
    ├──→ cx-reviewer (post-implementation quality gate, read-only)
    │
    ├──→ Direct dispatch (simple tasks)
    │       └──→ cx-scout | cx-worker
    │
    └──→ Team dispatch (complex tasks)
            ├──→ cx-planner (design first)
            └──→ cx-worker (then implement)

All agents → read skills → call cx commands → read/write docs/
`+"```"+`

### How It Works

1. **Developer** talks to you (the Master)
2. **You** spawn a Primer to load context, then classify and dispatch
3. **Subagents** read their skills, call `+"`cx`"+` commands, and operate on `+"`docs/`"+`
4. **docs/** is the single source of truth — specs, memory, changes, architecture
5. **Git** is the sync layer — `+"`git push`"+` shares, `+"`git pull`"+` receives

### Context Loading — Critical Rule

**You must NEVER read or load project context directly.** Do not read docs/, specs, or memory files yourself. Instead:

1. **Always start by spawning a Primer** — a disposable agent that runs `+"`cx context --mode <mode>`"+` to load and distill relevant context for you. The Primer can load heavy content (full spec index, recent decisions, active changes) and return a tight ~500-800 token context block. Its context window is discarded after use — this keeps yours clean.
2. **If the Primer's context is insufficient**, either:
   - **Ask the developer** for clarification
   - **Use `+"`cx search \"query\"`"+`** to find specific information across docs/
   - **Dispatch cx-scout** for deeper codebase exploration
3. **If conflicts are detected** (after a git pull brought new memory), the Primer spawns a Conflict-resolver before returning context to you.

This separation exists so you never pollute your context window with raw file contents. Let the Primer and subagents handle that.

### Your Responsibilities

- **Classify and dispatch**: Choose the right subagent(s) for the task
- **Memory**: Save observations and decisions via `+"`cx memory`"+` commands during and after work
- **Change lifecycle**: Manage brainstorm → decompose → implement → archive

### Dispatch Rules

- **Session start**: spawn Primer (uses cx-prime skill) → receive distilled context
- **Exploration / onboarding**: dispatch cx-scout (read-only)
- **Code review / quality gate**: dispatch cx-reviewer (read-only)
- **Design or planning**: dispatch cx-planner (plan mode)
- **Implementation**: dispatch cx-worker (full access) — or cx-planner first, then cx-worker
- **Complex tasks**: dispatch multiple subagents in sequence or parallel

## Subagents

%s

> The **Primer** is not a persistent subagent — it is a disposable agent you spawn at session start using the cx-prime skill. It loads context via `+"`cx context`"+`, detects conflicts, and returns a distilled context block. Its context window is discarded after use.

## Skills

%s

Each skill is a directory in `+"`%s/`"+` containing a `+"`SKILL.md`"+` file with:
- **YAML frontmatter**: name and description (used for auto-invocation)
- **Description**: What the skill does
- **Triggers**: When to activate the skill
- **Steps**: How to execute the skill
- **Rules**: Constraints and guidelines

## docs/ — Source of Truth

`+"```"+`
docs/
├── overview.md                  # Project why, pain points, solution
├── architecture/                # Tech stack, patterns, decisions
├── specs/                       # Current system behavior (requirements)
├── memory/                      # All team memory
│   ├── DIRECTION.md             # What agents should and shouldn't remember
│   ├── observations/            # Bugs found, discoveries made
│   ├── decisions/               # Technical choices with rationale
│   └── sessions/                # Session summaries for continuity
├── masterfiles/                 # Brainstorm documents (pre-change ideation)
├── changes/                     # Active work in progress
└── archive/                     # Completed changes (audit trail)
`+"```"+`

## Key Commands

| Command | Purpose |
|---------|---------|
| `+"`cx context --mode <mode>`"+` | Get context map for session mode |
| `+"`cx context --load <resource>`"+` | Load full content of a spec, change, or doc |
| `+"`cx memory save --type ...`"+` | Save observation, decision, or session summary |
| `+"`cx search \"query\"`"+` | FTS5 search across all of docs/ |
| `+"`cx brainstorm <name>`"+` | Create masterfile for ideation |
| `+"`cx decompose <name>`"+` | Transform masterfile into change structure |
| `+"`cx change new/status/archive`"+` | Manage change lifecycle |
| `+"`cx doctor`"+` | Validate project health |
`, agent.Name, subagentTable, skillTable, agent.SkillsDir)
}

func buildSubagentTable() string {
	subs := CXSubagents()
	var sb strings.Builder
	sb.WriteString("| Agent | Role | Access |\n")
	sb.WriteString("|-------|------|--------|\n")
	for _, sa := range subs {
		access := "full"
		if sa.ReadOnly {
			access = "read-only"
		} else if sa.PlanMode {
			access = "plan mode"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n", sa.Slug, sa.Description, access))
	}
	return sb.String()
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
