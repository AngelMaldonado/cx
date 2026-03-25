---
name: fix-mode
type: masterfile
---

## Problem

Every code change in cx today must go through the full BUILD lifecycle: Primer dispatch, requirements gathering (3-5 rounds), Planner, decompose, proposal.md, design.md, tasks.md, executor, Reviewer, archive. For a one-line rename, a typo fix, or a quick config tweak, this overhead is disproportionate. Developers end up bypassing cx entirely for small changes — they just tell the executor directly — which means cx loses traceability of those changes and the framework looks bureaucratic for trivial work.

## Context

Today the Master's dispatch table has three session modes:
- BUILD — full lifecycle with change docs, Primer, Planner, decompose, archive
- CONTINUE — resume an existing change (Primer + executor + Reviewer)
- PLAN — brainstorm only (Planner, masterfile, no code)

The skills spec (`docs/specs/skills/spec.md`) defines the skill format: H1 title `# Skill: cx-<name>`, four required sections (## Description, ## Triggers, ## Steps, ## Rules), max 200 lines. The skills spec also notes: "Skill files are pure markdown. No YAML frontmatter, no metadata." However, all current embedded skill files (`internal/skills/data/cx-build.md`, `cx-plan.md`, `cx-continue.md`) and their on-disk copies in `.claude/skills/` DO include YAML frontmatter. The embedded files are the canonical source; the spec has a documentation lag. The executor must match the actual pattern (frontmatter present).

The `go:embed data/*.md` glob in `internal/skills/` (or equivalent) auto-discovers all files matching `*.md`, so adding `cx-fix.md` requires no Go code changes.

`cx sync` overwrites `.claude/skills/<name>/SKILL.md` from the embedded source. The on-disk file must therefore match the embedded file exactly.

Two active changes touch the skills spec area:
- `spec-system-evolution` — TASK-16 updates `cx-change.md` skill; does not add new skills
- `memory-tui-dashboard` — touches memory spec, not skills

Neither active change conflicts with adding a new `cx-fix` skill. The delta spec for `spec-system-evolution/specs/skills/spec.md` only adds/modifies requirements for `cx-change` skill content, not for the skill registry list or CLAUDE.md dispatch table. An executor writing `cx-fix.md` will not conflict with those tasks.

CLAUDE.md has uncommitted local edits (per git status). The executor must read CLAUDE.md before editing it to avoid clobbering existing changes.

Agent-run logging via `cx agent-run log` is retained — the Master still tracks which agents were dispatched in a FIX session even though no docs/ artifacts are created.

## Direction

Add a 4th session mode: **FIX**. It is a lightweight, bypass-the-lifecycle path for small code changes.

### Trigger classification

The Master classifies a session as FIX when the developer's message matches:
- Trigger words: "fix", "patch", "tweak", "quick fix", "one-liner"
- Pattern: small, localized code change with no architectural implications
- Explicitly requests speed over process ("just change X to Y", "rename this")

### 3-step flow

```
1. Scout     → always dispatched; maps the affected code area
2. Executor  → dispatched with fix description + Scout's map
3. Reviewer  → optional; Master asks "Want a review?" after executor returns
```

### What FIX skips (by design)

- Primer dispatch (no project context load needed for trivial changes)
- Requirements gathering rounds
- Planner dispatch
- `cx decompose`, `cx change new`, proposal.md, design.md, tasks.md
- Archive flow and spec merging
- `cx memory save`, `cx memory decide` (no persistent project knowledge generated)
- Session summary (`cx memory session`)

### What FIX keeps

- Scout (always — executor needs to know where to look)
- Executor (the actual change)
- Reviewer (optional, developer's choice via AskUserQuestion)
- `cx agent-run log` (traceability in the agent-run log even without docs/)

### Traceability model

Git log is the only artifact. No docs/ files are created. If the developer later needs to understand a fix, `git log` shows the commit. This is a deliberate trade-off: FIX prioritizes speed over documentation.

### Skill file design

The new `cx-fix.md` skill follows the established embedded skill pattern exactly:

```
---
name: cx-fix
description: FIX mode workflow. Activate when the developer wants a quick, localized code change that bypasses the full change lifecycle.
---

# Skill: cx-fix

## Description
...

## Triggers
...

## Steps
1. Dispatch Scout ...
2. Dispatch executor ...
3. Ask "Want a review?" ...

## Rules
...
```

## Open Questions

None. All design decisions are resolved.

## Observations

- The frontmatter-present pattern (observed in cx-build.md, cx-plan.md) is the real convention, not the stale "no frontmatter" claim in the skills spec. All embedded skill files have frontmatter. The executor must follow the frontmatter pattern.
- `cx doctor` validates the four required sections (## Description, ## Triggers, ## Steps, ## Rules). The new skill must pass this check.
- The skills spec Skill Registry table (`docs/specs/skills/spec.md`) will need a `cx-fix` row added. Since `spec-system-evolution` has a delta spec for this area, the executor should note the conflict risk and either: (a) leave the registry table update for after spec-system-evolution archives, or (b) coordinate with the developer. Adding a row to the registry is additive and unlikely to conflict with spec-system-evolution's MODIFIED requirements (which only affect cx-change skill content). Safe to add.
- CLAUDE.md's dispatch table and "Quick tasks" note must be updated. Since CLAUDE.md has uncommitted local edits, the executor must read the file first.

## Files to Modify

### 1. `internal/skills/data/cx-fix.md` (NEW)

Create the embedded skill file. Content:

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

### 2. `.claude/skills/cx-fix/SKILL.md` (NEW)

Create the on-disk skill directory and file. Content must exactly match `internal/skills/data/cx-fix.md` (identical — `cx sync` would overwrite from embedded source anyway).

### 3. `CLAUDE.md` (MODIFY)

Two changes:

**a. Add FIX row to the dispatch table:**

```
| **FIX** | `cx-fix` | Developer wants a quick, localized code change |
```

Insert after the PLAN row so the table reads:
```
| Mode | Skill | When |
|------|-------|------|
| **BUILD** | `cx-build` | Developer wants to create something new |
| **CONTINUE** | `cx-continue` | Developer is resuming existing work |
| **PLAN** | `cx-plan` | Developer wants to brainstorm or design |
| **FIX** | `cx-fix` | Developer wants a quick, localized code change |
```

**b. Update "Quick tasks" line to reference FIX for small fixes:**

Change:
```
**Quick tasks** (no skill needed): code question → Scout; health check → `cx doctor`; simple answer → respond directly.
```

To:
```
**Quick tasks** (no skill needed): code question → Scout; health check → `cx doctor`; simple answer → respond directly. For small code changes, use **FIX** mode (`cx-fix`).
```

### 4. No Go code changes

The `go:embed data/*.md` glob auto-discovers `cx-fix.md`. No changes to `internal/skills/` Go files, no changes to `cmd/`, no changes to `internal/`.

## Risks

**Risk 1: Frontmatter vs. skills spec discrepancy**
The `docs/specs/skills/spec.md` says "No YAML frontmatter" but all existing embedded skills have frontmatter. The executor might follow the spec text instead of the actual pattern.
Mitigation: The masterfile explicitly calls this out. The executor must use the frontmatter pattern matching `cx-build.md`.

**Risk 2: CLAUDE.md uncommitted local edits clobbered**
CLAUDE.md has local modifications in git status. A naive write could overwrite them.
Mitigation: Executor must read CLAUDE.md before editing. Use targeted Edit (not full Write).

**Risk 3: spec-system-evolution skills delta conflict**
spec-system-evolution has a delta spec for the skills area. Adding a new skill and a registry row is additive and doesn't touch any of the MODIFIED/ADDED requirements in that delta (which are about cx-change skill content only). Low conflict risk.
Mitigation: Adding cx-fix to the registry table is a new row (additive), not a modification to existing content. Safe to proceed.

**Risk 4: cx doctor validation failure**
If the new skill file is missing one of the four required sections, `cx doctor` will warn.
Mitigation: The skill content in this plan includes all four sections. Executor should run `cx doctor` after implementation to verify.

**Risk 5: FIX misused for large changes**
Developers might classify non-trivial changes as FIX to skip the lifecycle.
Mitigation: The skill's Rules section explicitly states: "If the fix description grows in scope (multiple files, architectural changes), stop and redirect to BUILD mode." This is enforced by the Master reading the skill.

## Testing

1. `cx doctor` — must pass with no warnings about skill files or missing sections
2. `cx sync` equivalent test — verify `.claude/skills/cx-fix/SKILL.md` content matches `internal/skills/data/cx-fix.md`
3. CLAUDE.md manual review — dispatch table has 4 rows including FIX; "Quick tasks" line references FIX mode
4. Skill format check — `# Skill: cx-fix` H1 present, all four `##` sections present, file under 200 lines
5. End-to-end smoke test — developer says "fix the typo in error message X"; Master classifies as FIX, dispatches Scout, dispatches executor, asks about review

## References

- `docs/specs/skills/spec.md` — skill format spec (note frontmatter discrepancy)
- `docs/specs/session-modes/spec.md` — current three modes (FIX becomes the 4th)
- `docs/specs/orchestration/spec.md` — agent hierarchy
- `internal/skills/data/cx-build.md` — canonical reference for skill format with frontmatter
- `internal/skills/data/cx-continue.md` — closest analogue (lightweight mode)
- `docs/changes/spec-system-evolution/specs/skills/spec.md` — active delta for skills area (additive only, no conflict)
