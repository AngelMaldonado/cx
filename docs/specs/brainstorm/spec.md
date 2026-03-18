# Spec: Brainstorm & Decompose

The brainstorm phase is for structured ideation before committing to a change. It produces a **masterfile** — a single document that captures the idea, constraints, and direction — which is then **decomposed** into a proper change structure.

---

## When to Use

Brainstorming is the natural path from **PLAN** mode. The developer is thinking, not building. The Primer detects PLAN mode, and the Master uses the brainstorm skill.

BUILD mode can also enter brainstorm if the agent judges the scope is large enough to warrant structured thinking first. The skill guides this decision:
- Quick fix or small feature → skip brainstorm, `cx change new` directly
- New system, large feature, architectural change → brainstorm first

---

## Masterfile

### Location

Masterfiles live in `docs/masterfiles/` during the brainstorm phase:

```
docs/masterfiles/
├── gas-leak-alerting.md
└── v2-architecture.md
```

Committed to git so the team can see what's being ideated on.

### Format

`cx brainstorm` creates a masterfile from a template:

```markdown
# Masterfile: <name>

## Problem
<what pain point or opportunity is being addressed>

## Context
<relevant background — what exists today, what constraints are known>

## Direction
<the emerging solution direction — updated as the brainstorm evolves>

## Open Questions
- <question 1>
- <question 2>

## Observations
<discoveries, resolved questions, and notes gathered during brainstorming>

## Files to Modify
<specific files and what changes in each>

## Risks
<what could go wrong and how to mitigate>

## Testing
<how to verify the implementation>

## References
<links to specs, external docs, prior art>
```

### Template generation

```bash
cx brainstorm <name>
```

Creates `docs/masterfiles/<name>.md` with the template above. The `<name>` is a kebab-case slug (e.g., `gas-leak-alerting`).

If a masterfile with that name already exists, the command exits with an error.

---

## Refinement — A Skill, Not a Command

There is no `cx refine` binary command. Refinement is handled entirely by the agent, guided by the **cx-refine skill**.

The skill teaches the agent:

1. **How to update the masterfile** — which sections to modify based on new information
2. **When to move observations to their own section** — if a discovery during brainstorming is significant enough
3. **How to evolve the Direction section** — from vague to specific as understanding grows
4. **When the masterfile is ready for decompose** — all open questions resolved, direction clear, scope defined

### Refinement triggers

The agent refines the masterfile when:
- The developer provides new input ("also consider firmware side", "what about offline mode")
- The agent discovers something during research (web search, spec reading, code exploration)
- An open question gets answered

### Refinement rules

- **Never delete content** — move resolved questions to the Context or Observations section
- **Direction should narrow over time** — each refinement should make it more specific
- **Open Questions should shrink** — if they grow, the scope may be too large
- **The developer always reviews** — the agent proposes changes, the developer approves

---

## Decompose

When the masterfile is ready, the agent runs:

```bash
cx decompose <name>
```

### What it does

1. Reads `docs/masterfiles/<name>.md`
2. Creates `docs/changes/<name>/` with templates:
   - `proposal.md` — pre-filled from masterfile's Problem + Direction sections
   - `design.md` — empty template (agent fills during BUILD)
   - `tasks.md` — empty template (agent fills during BUILD)
3. If the masterfile's Direction section mentions specific spec areas, creates delta placeholders:
   - `docs/changes/<name>/specs/<area>/delta.md`
4. Archives the masterfile (see below)

### Pre-filling proposal.md

The binary maps masterfile sections to proposal sections:

| Masterfile section | Downstream document |
|-------------------|---------------------|
| `## Problem` | `proposal.md` → `## Problem` (copied verbatim) |
| `## Direction` | `proposal.md` → `## Approach` (copied as starting point) |
| `## Context` | `proposal.md` → `## Scope` (added as context) |
| `## Observations` | `proposal.md` → appended as an appendix |
| `## Files to Modify` | `design.md` → `## Files to Modify` |
| `## Risks` | `design.md` → `## Risks` |
| `## Testing` | `tasks.md` → `## Testing` |

The agent is expected to refine the proposal further — the pre-fill is a starting point, not the final version.

---

## Post-Decompose Archive

After decompose, the masterfile is moved to the archive:

```
docs/masterfiles/gas-leak-alerting.md
    → docs/archive/<date>-masterfile-gas-leak-alerting.md
```

This preserves the brainstorm history. The masterfile is no longer active — the change in `docs/changes/` is the source of truth from this point forward.

---

## Commands

| Command | Purpose |
|---------|---------|
| `cx brainstorm <name>` | Create masterfile template in `docs/masterfiles/` |
| `cx decompose <name>` | Transform masterfile into a change structure, archive the masterfile |

---

## Lifecycle

```
Developer (PLAN mode): "Let's think about gas leak alerting"
    │
    ▼
Agent runs: cx brainstorm gas-leak-alerting
    → Creates docs/masterfiles/gas-leak-alerting.md
    │
    ▼
Agent + developer iterate (skill-guided refinement)
    → Agent updates masterfile sections
    → Open questions get resolved
    → Direction narrows
    │
    ▼
Developer: "Looks good, let's build it"
    │
    ▼
Agent runs: cx decompose gas-leak-alerting
    → Creates docs/changes/gas-leak-alerting/ (proposal, design, tasks)
    → Archives masterfile to docs/archive/
    │
    ▼
Session switches to BUILD mode
    → Agent fills design.md, tasks.md
    → Agent creates Linear issues
    → Implementation begins
```
