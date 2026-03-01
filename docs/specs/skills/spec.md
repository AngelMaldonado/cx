# Spec: Skill Protocol

Skills are markdown files that teach coding agents how to use the `cx` binary. They are the only interface between agents and the system — developers never call `cx` commands directly (except `cx init` and `cx dashboard`).

See [catalog/](catalog/) for the full content of each skill.

---

## Skill Format

Every skill follows this exact structure:

```markdown
# Skill: cx-<name>

## Description
One paragraph: what this skill does and when the agent should use it.

## Triggers
Bullet list of natural language patterns or conditions that activate this skill.
The agent matches these against the developer's message or session context.

## Steps
Numbered list of exact actions the agent should take.
Each step specifies:
- What cx command to run (with flags and arguments)
- How to interpret the output
- What to do next based on the output

## Rules
Bullet list of constraints, guardrails, and things the agent must never do.
These override the agent's general behavior when this skill is active.
```

### Format rules

- **H1 title** must be `# Skill: cx-<name>` — the binary uses this to validate skill files
- **All four sections** are required. `cx doctor` warns on any skill file missing a section
- **## Steps** must contain numbered steps with concrete commands — no vague instructions
- **## Rules** must contain at least one rule
- Skill files are pure markdown. No YAML frontmatter, no metadata
- Max recommended length: 200 lines (agents process shorter skills more reliably)

---

## Skill Types

| Type | Used by | Spawned how |
|------|---------|-------------|
| **Master skill** | The Master agent (pure orchestrator) | Master reads the skill from its skills directory |
| **Standalone agent skill** | Primer, Conflict-resolver, Reviewer (disposable context) | Master or Primer spawns the agent, passes the skill |
| **Team agent skill** | Scout, Contractor, Workers within a team | Supervisor or Contractor spawns the agent, passes the skill |

Standalone agent skills (cx-prime, cx-conflict-resolve, cx-review) include explicit instructions for what the agent should output back to its spawner. Team agent skills (cx-scout, cx-supervise, cx-contract) include coordination instructions for working within the team hierarchy.

See [orchestration spec](../orchestration/spec.md) for the full agent hierarchy.

---

## Skill Generation

`cx init` and `cx sync` generate skills tailored to each detected agent:

| Agent | Skill location | Config file |
|-------|---------------|-------------|
| Claude Code | `.claude/skills/cx-*.md` | `.claude/CLAUDE.md` |
| Gemini | `.gemini/skills/cx-*.md` | `.gemini/GEMINI.md` |
| Codex | `.codex/skills/cx-*.md` | `.codex/AGENTS.md` |

The skill **content** is identical across agents. Only the file path differs. `cx sync` always overwrites skill files — they are not user-editable. If a developer needs to customize agent behavior, they edit the agent's config file (CLAUDE.md, etc.), not the skills.

### Compilation

Skill files are embedded in the Go binary at build time using `go:embed`. This means:

- Skills ship as part of the binary — no external files to distribute or lose
- `cx sync` copies skills from the embedded filesystem to the agent's skills directory
- Updating skills requires a new binary release (`cx upgrade` → `brew upgrade cx` → `cx sync` in each project)
- The binary's embedded versions are the canonical source; on-disk skill files are always overwritable copies

---

## Skill Registry

| Skill | Type | Used by | Purpose |
|-------|------|---------|---------|
| [cx-prime](catalog/cx-prime.md) | Standalone agent | Primer | Context priming at session start |
| [cx-memory](catalog/cx-memory.md) | Master / Team agent | Master, Contractor | Save observations, decisions, sessions |
| [cx-brainstorm](catalog/cx-brainstorm.md) | Master | Master | Create and decompose masterfiles |
| [cx-refine](catalog/cx-refine.md) | Master | Master | Iteratively improve masterfiles |
| [cx-change](catalog/cx-change.md) | Master / Team agent | Master, Contractor | Manage change lifecycle |
| [cx-linear](catalog/cx-linear.md) | Master / Team agent | Master, Contractor | Linear integration via MCP |
| [cx-scout](catalog/cx-scout.md) | Team agent | Scout | Read-only codebase exploration |
| [cx-supervise](catalog/cx-supervise.md) | Team agent | Supervisor | Team coordination for complex tasks |
| [cx-contract](catalog/cx-contract.md) | Team agent | Contractor | Implementation coordination and Worker management |
| [cx-doctor](catalog/cx-doctor.md) | Master | Master | Project health checks |
| [cx-review](catalog/cx-review.md) | Standalone agent | Reviewer | Post-implementation review against specs |
| [cx-conflict-resolve](catalog/cx-conflict-resolve.md) | Standalone agent | Conflict-resolver | Resolve semantic conflicts after git pull |

---

## Validation

`cx doctor` checks:
- All expected skill files exist for each detected agent
- Each skill file has the four required sections (## Description, ## Triggers, ## Steps, ## Rules)
- Skill files match the binary's built-in versions (warns if they've been manually edited)
- Agent config files reference the skills directory
