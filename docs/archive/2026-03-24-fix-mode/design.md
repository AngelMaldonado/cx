---
name: fix-mode
type: design
---

## Architecture

FIX mode is a skill-based session mode that sits alongside BUILD, CONTINUE, and PLAN in the Master's dispatch table. It introduces no new Go components — it is entirely a documentation and configuration change.

### Component map

```
CLAUDE.md (dispatch table)
    └── cx-fix skill (trigger classification + workflow steps)
            ├── Scout agent (always dispatched — maps affected code)
            ├── Executor agent (applies the fix)
            └── Reviewer agent (optional — Master asks developer)

internal/skills/data/cx-fix.md   ← embedded source of truth
.claude/skills/cx-fix/SKILL.md   ← on-disk copy (cx sync overwrites from embedded)
```

### 3-step flow

```
1. Scout     → always first; receives the developer's fix description;
               returns a focused map of affected files
2. Executor  → receives fix description + Scout's map;
               applies the change; no change docs required
3. Reviewer  → optional; Master uses AskUserQuestion("Fix applied. Want a review?");
               if yes: Reviewer receives executor's changes + fix description
```

### What FIX skips (by design)

FIX intentionally omits every lifecycle artifact that makes BUILD heavyweight:

| Skipped step | Reason |
|---|---|
| Primer dispatch | No project context needed for localized changes |
| Requirements gathering rounds | Fix description IS the requirement |
| Planner dispatch | No design needed for trivial changes |
| `cx decompose`, `cx change new` | No change scaffold created |
| proposal.md, design.md, tasks.md | Git log is sufficient traceability |
| Archive flow and spec merging | No docs/ artifacts to archive |
| `cx memory save` / `cx memory session` | No project knowledge generated |

### What FIX keeps

| Retained step | Reason |
|---|---|
| Scout | Executor needs file locations; always dispatched |
| Executor | The actual change |
| Reviewer (optional) | Developer may want a quality check even on small fixes |
| `cx agent-run log` | Traceability in agent-run log even without docs/ |

### Scope guard

If the fix description grows in scope (multiple unrelated files, architectural implications), the Master stops FIX mode and redirects to BUILD. The skill's Rules section enforces this explicitly.

## Technical Decisions

### Frontmatter convention

All existing embedded skill files (`cx-build.md`, `cx-continue.md`, etc.) include YAML frontmatter despite the skills spec saying "No YAML frontmatter". The embedded files are the canonical source; the spec has a documentation lag. The new `cx-fix.md` must match the actual frontmatter pattern:

```
---
name: cx-fix
description: FIX mode workflow. Activate when the developer wants a quick, localized code change that bypasses the full change lifecycle.
---
```

### No Go changes required

`internal/skills/` uses a `go:embed data/*.md` glob. Adding `cx-fix.md` to `data/` is auto-discovered at build time — no Go source edits needed.

### On-disk skill must match embedded exactly

`cx sync` overwrites `.claude/skills/<name>/SKILL.md` from the embedded source. The on-disk file is therefore always derived; the executor must write it as an exact copy of the embedded file.

### CLAUDE.md edit strategy

CLAUDE.md has uncommitted local edits at change-start (per git status). The executor must use a targeted Edit (not a full Write) to avoid clobbering those changes. Two edits required:
1. Append FIX row to the dispatch table (after the PLAN row)
2. Append FIX mode reference to the "Quick tasks" line

### Skills spec delta and active change interaction

The `spec-system-evolution` active change has a delta spec for the `skills` spec area. That delta only modifies requirements for the `cx-change` skill content — it does not touch the skill registry table or CLAUDE.md. Adding `cx-fix` as a new registry row is additive and will not conflict.

## Implementation Notes

### Skill file content (complete)

The executor writes `internal/skills/data/cx-fix.md` with this content:

```markdown
---
name: cx-fix
description: FIX mode workflow. Activate when the developer wants a quick, localized code change that bypasses the full change lifecycle.
---

# Skill: cx-fix

## Description

Lightweight path for small, localized code changes. Skips the full change lifecycle (no change docs, no Planner, no archive). Use when the developer wants speed over documentation.

## Triggers

- Developer says "fix", "patch", "tweak", "quick fix", "one-liner"
- Developer requests a small, localized change without architectural implications
- Developer explicitly wants to skip the change lifecycle ("just change X to Y", "rename this", "update this value")

## Steps

### 1. Map the affected area

- Dispatch **Scout** to locate the relevant files and understand the code surrounding the fix
- Pass the developer's fix description to Scout so it knows where to look
- Wait for Scout to return a focused map of the affected area

### 2. Apply the fix

- Dispatch **executor agent** with:
  1. The developer's fix description (exact wording)
  2. Scout's map of affected files
- No change docs required — the executor works directly from the description and Scout context
- After executor returns, log the run: `cx agent-run log --type executor --session <session_id> --status <status> --summary "..."`

### 3. Offer review (optional)

- Use `AskUserQuestion` to ask the developer: "Fix applied. Want a review?"
- If yes: dispatch **Reviewer** with the executor's changes and the original fix description as acceptance criteria
- If no: session is complete

## Rules

- Never create change docs (no proposal.md, design.md, tasks.md, no cx decompose)
- Never dispatch Primer — FIX mode loads no project context intentionally
- Never dispatch Planner — FIX mode skips planning entirely
- Never save memory (no cx memory save, no cx memory session, no cx memory decide)
- Always dispatch Scout before the executor — the executor needs file locations
- If the fix description grows in scope (multiple files, architectural changes), stop and redirect to BUILD mode
- FIX mode is for localized changes only — one area, one concern
```

### CLAUDE.md dispatch table (target state)

```
| Mode | Skill | When |
|------|-------|------|
| **BUILD** | `cx-build` | Developer wants to create something new |
| **CONTINUE** | `cx-continue` | Developer is resuming existing work |
| **PLAN** | `cx-plan` | Developer wants to brainstorm or design |
| **FIX** | `cx-fix` | Developer wants a quick, localized code change |
```

### CLAUDE.md Quick tasks line (target state)

```
**Quick tasks** (no skill needed): code question → Scout; health check → `cx doctor`; simple answer → respond directly. For small code changes, use **FIX** mode (`cx-fix`).
```

### Validation checklist

After implementation, the executor must verify:
1. `cx doctor` passes with no warnings about skill files or missing sections
2. `cx sync` equivalent — `.claude/skills/cx-fix/SKILL.md` content matches `internal/skills/data/cx-fix.md`
3. CLAUDE.md dispatch table has 4 rows including FIX
4. Skill file has all four required `##` sections (Description, Triggers, Steps, Rules) and is under 200 lines
