# Delta Spec: skills

Change: fix-mode

## ADDED Requirements

### cx-fix skill entry in the Skill Registry

The Skill Registry table gains a new row:

| Skill | Type | Used by | Purpose |
|-------|------|---------|---------|
| [cx-fix](catalog/cx-fix.md) | Master | Master | Lightweight FIX mode for quick, localized code changes |

The row is inserted after `cx-plan` (or at the end of the Master skill group) in the registry table.

### cx-fix skill file

A new embedded skill file is created at `internal/skills/data/cx-fix.md` and mirrored to `.claude/skills/cx-fix/SKILL.md`.

The skill follows the established embedded file convention: YAML frontmatter present (matching `cx-build.md` pattern), four required sections, under 200 lines.

**Frontmatter:**
```yaml
---
name: cx-fix
description: FIX mode workflow. Activate when the developer wants a quick, localized code change that bypasses the full change lifecycle.
---
```

**Required sections:** Description, Triggers, Steps, Rules — all present.

**Key rules the skill enforces:**
- Never create change docs (no proposal.md, design.md, tasks.md, no `cx decompose`)
- Never dispatch Primer
- Never dispatch Planner
- Never save memory (`cx memory save`, `cx memory session`, `cx memory decide`)
- Always dispatch Scout before the executor
- If fix description grows in scope, redirect to BUILD mode

### cx-fix passes cx doctor validation

`cx doctor` must find:
- `cx-fix` skill file exists at `.claude/skills/cx-fix/SKILL.md`
- All four required sections present (## Description, ## Triggers, ## Steps, ## Rules)
- File matches the binary's embedded version

## MODIFIED Requirements

### Skill Registry — count reference

Any documentation or spec text referring to the number of skills in the registry must be updated to include `cx-fix`.

### Frontmatter convention documentation

The Format rules section states "Skill files are pure markdown. No YAML frontmatter, no metadata." This is a documentation lag — all existing embedded skill files (`cx-build.md`, `cx-continue.md`, etc.) include YAML frontmatter. The `cx-fix.md` skill follows the actual convention (frontmatter present). The spec text should be corrected to reflect reality:

**Previous:** "Skill files are pure markdown. No YAML frontmatter, no metadata"

**New:** "Embedded skill files include YAML frontmatter (`name`, `description` keys). On-disk skill files (`.claude/skills/`) mirror the embedded content exactly, including frontmatter."

## REMOVED Requirements

None.
