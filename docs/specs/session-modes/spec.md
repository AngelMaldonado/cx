# Spec: Session Entry Modes

When a developer starts a CX session, they're in one of three modes. Each has fundamentally different context needs. The **Primer** handles mode classification, context loading, and distillation — the Master never interacts with `cx context` directly.

> Full priming architecture: [context-priming spec](../context-priming/spec.md)

See diagram: [session entry modes flowchart](diagrams/05-session-entry-modes.mermaid)

---

## The Three Modes

```
Developer starts session
        │
        ▼
Master spawns Primer
        │
        ▼
Primer checks .cx/conflicts.json
        │ (if conflicts exist → spawn Conflict-resolver first)
        ▼
Primer classifies developer's opening message
        │
        ├── "Let's continue on the BLE pairing"
        │         ▼
        │    CONTINUE — Heavy: session recovery + change files +
        │               change-scoped memory + relevant specs
        │
        ├── "I want to add gas leak SMS alerts"
        │         ▼
        │    BUILD — Medium: spec index + relevant specs +
        │            active decisions + recent observations
        │
        └── "We need to plan the v2 architecture"
                  ▼
             PLAN — Light: overview + architecture summary +
                    personal preferences only
```

---

## Mode: CONTINUE

**When**: Developer is picking up where they left off on an existing change.

**Signal phrases**: "let's continue", "where were we", "back to [feature]", "pick up on [change]", "resume", references an existing change name, mentions previous work.

### Memory touchpoints

| Step | Action | Command |
|------|--------|---------|
| Session start | Primer loads last session (critical bridge) | `cx memory list --type session --change <name>` (sorted by `started_at` desc) |
| Session start | Primer loads change-scoped memory | `cx memory search --change <name>` |
| Implementation | Executor saves per-task discoveries | `cx memory save --type observation --change <name>` |
| Session end | Master saves session summary with populated `next_steps` | `cx memory session --next "..."` |

The `next_steps` field of the previous session summary IS the session recovery mechanism. Primer always includes it in the primed context for CONTINUE mode. Executors and Master are responsible for populating `next_steps` meaningfully at session end.

### What the Primer loads

**From `cx context --mode continue --change <name>` (the map):**

| Section | Content | Priority |
|---------|---------|----------|
| `[CHANGE SUMMARY]` | proposal.md, design.md, tasks.md summaries | Critical |
| `[SESSION RECOVERY]` | Last session from `.cx/memory.db`: goal, accomplished, blockers, next steps, files | Critical |
| `[CHANGE MEMORY]` | Observations + decisions scoped to this change (via `cx memory search --change <name>`) | High |
| `[DIRECTION]` | Full DIRECTION.md (if it exists) | Medium |
| `[PERSONAL NOTES]` | Relevant to files/specs being touched | Low |

**From `cx context --load` (selective drill-in):**

The Primer loads the canonical spec for each area the change's delta touches, so the Master knows the current state being modified.

### What gets excluded

- Observations unrelated to this change
- Decisions about other spec areas
- Other team members' session summaries
- General project overview
- Spec areas not affected by this change

---

## Mode: BUILD

**When**: Developer is starting a new feature or fix from scratch.

**Signal phrases**: "I want to add...", "let's build...", "new feature:", "we need to create...", "implement", describes something that doesn't exist yet.

### Memory touchpoints

Primer is dispatched at the start of every BUILD session before requirements gathering.

| Step | Action | Command |
|------|--------|---------|
| Session start | Primer loads recent observations + active decisions | `cx memory list --type observation --recent 7d`, `cx memory list --type decision` |
| Requirements | Master saves significant constraints as decisions | `cx memory decide --change <name>` |
| Implementation | Executor saves per-task discoveries | `cx memory save --type observation --change <name>` |
| Each agent dispatch | Master logs agent run | `cx agent-run log --type <t> --session <id> ...` |
| Session end | Master saves session summary | `cx memory session --goal "..." --accomplished "..." --next "..."` |

### What the Primer loads

**From `cx context --mode build` (the map):**

| Section | Content | Priority |
|---------|---------|----------|
| `[SPEC INDEX]` | Full docs/specs/index.md — what already exists | Critical |
| `[ACTIVE CHANGES]` | Names + summaries of in-progress work | High |
| `[DECISIONS]` | All active decisions from `.cx/memory.db` (title + outcome, compact) | High |
| `[OBSERVATIONS]` | Last 7 days from `cx memory list --type observation --recent 7d` (compact) | Medium |
| `[DIRECTION]` | Full DIRECTION.md (if it exists) | Medium |
| `[PERSONAL NOTES]` | Preferences and patterns | Low |

**From `cx context --load` (selective drill-in):**

The Primer reads the spec index, identifies which spec areas relate to the developer's intent, and loads those. Typically 1-2 areas, rarely more. If an active change overlaps with the developer's intent, the Primer includes a collision warning.

### What gets excluded

- Session history (there's no previous session for this)
- Change-specific observations (no change exists yet)
- Full decision rationale (just outcomes)
- Observations older than 7 days
- Spec areas unrelated to the developer's intent

---

## Mode: PLAN

**When**: Developer is doing high-level thinking — planning a big feature, rethinking architecture, starting a new project.

**Signal phrases**: "let's plan...", "we need to think about...", "brainstorm...", "what if we...", "how should we approach...", "architecture", "v2", "roadmap", "redesign".

### Memory touchpoints

| Step | Action | Command |
|------|--------|---------|
| Session start | Minimal — personal notes only (clean-slate intent) | Personal notes via `cx context --mode plan` |
| Transition to BUILD | Master saves planning session summary | `cx memory session --goal "..." --accomplished "..."` |

### What the Primer loads

**From `cx context --mode plan` (the map — no step 2 needed):**

| Section | Content | Priority |
|---------|---------|----------|
| `[OVERVIEW]` | Full docs/overview.md | High |
| `[ARCHITECTURE SUMMARY]` | Design principles + tech stack from architecture doc | Medium |
| `[PERSONAL NOTES]` | Preferences and working-style notes | Medium |

### What gets excluded

- ALL session history
- ALL observations and discoveries
- ALL decisions (planning should be unconstrained)
- Spec details (too granular for high-level thinking)
- Team member activity
- DIRECTION.md (not relevant to planning)

### Why so little?

Planning needs a **clean slate**. Loading the agent with implementation details, past decisions, and team observations anchors thinking to the current state. The whole point of planning is to step back and think freely. The developer will reference specific things when needed — the agent can then do targeted searches with `cx memory search`.

---

## Mode Classification

Classification is done by the **Primer**, not the binary. The Primer reads the developer's opening message and applies these rules:

### Decision tree

```
Developer's opening message
    │
    ├── References an existing change by name?
    │   └── YES → CONTINUE (use that change name)
    │
    ├── Mentions "continue", "resume", "pick up", "where were we"?
    │   └── YES → CONTINUE (Primer checks active changes to find which one)
    │
    ├── Mentions "plan", "brainstorm", "think about", "redesign", "architecture"?
    │   └── YES → PLAN
    │
    ├── Describes something new to build, add, create, or implement?
    │   └── YES → BUILD
    │
    └── Unclear → BUILD (safest default)
```

### Disambiguation

If CONTINUE is detected but the change name is ambiguous (e.g., "let's continue" with multiple active changes), the Primer returns a disambiguation request instead of context. The Master presents this to the developer as a question.

---

## Token Budgets

### What the Primer consumes (disposable)

| Mode | Map (step 1) | Loaded content (step 2) | Primer total |
|------|-------------|------------------------|-------------|
| CONTINUE | ~600 tok | ~1-2K tok (canonical specs) | ~2-3K tok |
| BUILD | ~800 tok | ~2-4K tok (1-2 spec areas) | ~3-5K tok |
| PLAN | ~400 tok | 0 tok | ~400 tok |

### What the Master receives

| Mode | Primed context |
|------|---------------|
| CONTINUE | 500-800 tok |
| BUILD | 500-800 tok |
| PLAN | 300-500 tok |

The Primer absorbs the full load and distills it. The Master starts with only what's relevant, leaving maximum room for the actual work.

> For how the Master dispatches work after receiving primed context, see [orchestration spec](../orchestration/spec.md).
