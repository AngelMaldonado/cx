---
name: fix-mode
type: proposal
---

## Problem

Every code change in cx today must go through the full BUILD lifecycle: Primer dispatch, multiple rounds of requirements gathering, Planner, decompose, proposal.md, design.md, tasks.md, executor, Reviewer, and archive. For a one-line rename, a typo fix, or a quick config tweak, this overhead is disproportionate. In practice, developers bypass cx entirely for small changes — they tell the executor directly — which means cx loses traceability of those changes and the framework looks bureaucratic for trivial work.

## Approach

Add a 4th session mode, **FIX**, as a lightweight bypass-the-lifecycle path for small, localized code changes. FIX mode runs a 3-step flow: Scout maps the affected area, executor applies the fix, and the developer optionally requests a Reviewer pass. No change scaffold, no masterfile, no Planner, no archive. Traceability is provided solely by git log.

## Scope

**In scope:**
- New embedded skill file: `internal/skills/data/cx-fix.md`
- New on-disk skill file: `.claude/skills/cx-fix/SKILL.md` (content identical to embedded)
- CLAUDE.md changes: add FIX row to the dispatch table; update Quick tasks note to reference FIX for small code changes
- Delta specs for `session-modes` and `skills` spec areas

**Out of scope:**
- No Go code changes — the `go:embed data/*.md` glob auto-discovers `cx-fix.md`
- No changes to `cmd/`, `internal/`, or any Go source files
- No new `cx` CLI commands
- No memory persistence (FIX sessions intentionally generate no docs/ artifacts)

## Affected Specs

- **session-modes** — adds FIX as the 4th session mode with its own classification rules and 3-step flow
- **skills** — adds `cx-fix` to the skill registry and documents the new skill's format
