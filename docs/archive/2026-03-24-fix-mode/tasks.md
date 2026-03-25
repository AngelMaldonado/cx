---
name: fix-mode
type: tasks
---

## Tasks

### Task 1: Create embedded skill file [DONE]

**Agent:** cx-worker (general-purpose executor)

**Files to create:**
- `/Users/amald/dev/cx/internal/skills/data/cx-fix.md`

**What to do:**

Create the new embedded skill file with YAML frontmatter followed by the four required `##` sections. The content must be written exactly as specified below — the Go embed glob (`go:embed data/*.md`) will auto-discover it at build time; no Go source edits are needed.

Write this exact content:

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

**Validation after creation:**
- File must have exactly four `##` sections: Description, Triggers, Steps, Rules
- File must be under 200 lines
- Run `cx doctor` and confirm no warnings about the skill

**Dependencies:** None — this task can be done first independently.

---

### Task 2: Create on-disk skill file [DONE]

**Agent:** cx-worker (general-purpose executor)

**Files to create:**
- `/Users/amald/dev/cx/.claude/skills/cx-fix/SKILL.md`

**What to do:**

Create the directory `.claude/skills/cx-fix/` and write `SKILL.md` with content identical to the embedded file from Task 1 (including frontmatter). The on-disk copy is always derived from the embedded source — `cx sync` overwrites it from the embedded version, so they must match exactly.

Copy the full content from `internal/skills/data/cx-fix.md` verbatim.

**Validation after creation:**
- Content must be byte-for-byte identical to `internal/skills/data/cx-fix.md`
- The four required `##` sections must be present

**Dependencies:** Task 1 must be complete so the on-disk file can be copied from the embedded source.

---

### Task 3: Update CLAUDE.md dispatch table and quick tasks line [DONE]

**Agent:** cx-worker (general-purpose executor)

**Files to modify:**
- `/Users/amald/dev/cx/CLAUDE.md`

**What to do:**

CLAUDE.md has uncommitted local edits — you MUST read the file first before making any edits. Use targeted Edit operations (not a full Write) to avoid clobbering existing changes.

Make exactly two edits:

**Edit 1 — Append FIX row to the dispatch table.**

Locate this exact block:

```
| Mode | Skill | When |
|------|-------|------|
| **BUILD** | `cx-build` | Developer wants to create something new |
| **CONTINUE** | `cx-continue` | Developer is resuming existing work |
| **PLAN** | `cx-plan` | Developer wants to brainstorm or design |
```

Replace it with:

```
| Mode | Skill | When |
|------|-------|------|
| **BUILD** | `cx-build` | Developer wants to create something new |
| **CONTINUE** | `cx-continue` | Developer is resuming existing work |
| **PLAN** | `cx-plan` | Developer wants to brainstorm or design |
| **FIX** | `cx-fix` | Developer wants a quick, localized code change |
```

**Edit 2 — Update the Quick tasks line.**

Locate this exact line:

```
**Quick tasks** (no skill needed): code question → Scout; health check → `cx doctor`; simple answer → respond directly.
```

Replace it with:

```
**Quick tasks** (no skill needed): code question → Scout; health check → `cx doctor`; simple answer → respond directly. For small code changes, use **FIX** mode (`cx-fix`).
```

**Validation after edits:**
- Dispatch table has exactly 4 rows (BUILD, CONTINUE, PLAN, FIX)
- Quick tasks line ends with the FIX mode reference
- No other content in CLAUDE.md was changed

**Dependencies:** None — this task can be done in parallel with Tasks 1 and 2, but read the file first to confirm the current state of uncommitted edits.

---

## Implementation Notes

**Execution order:** Tasks 1 and 3 are fully independent and can be dispatched in parallel. Task 2 depends on Task 1 (copy from embedded source), so it should run after Task 1 completes.

**No Go code changes.** The `go:embed data/*.md` glob in `internal/skills/` auto-discovers `cx-fix.md` at build time. Do not touch any `.go` files.

**Validation gate.** After all three tasks complete, run `cx doctor` to confirm:
1. No warnings about missing skill sections
2. The FIX skill is recognized
3. Four dispatch-table rows appear in CLAUDE.md

**Skills table in CLAUDE.md.** The Skills table at the bottom of CLAUDE.md (listing all skill paths) does not need updating — it references `.claude/skills/` on-disk files, and the new `cx-fix` skill will appear there once Task 2 is done. If the executor notices it is absent and the table should list it, add `| cx-fix | [SKILL.md](skills/cx-fix/SKILL.md) |` in alphabetical order after `cx-continue`. This is a minor addition and does not block the other tasks.
