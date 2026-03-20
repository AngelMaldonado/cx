---
name: doctor
type: delta-spec
area: doctor
change: memory-architecture
---

## ADDED Requirements

### Memory DB Health Checks (new check group or additions to existing "memory files" group)

| Check | Severity | Auto-fixable |
|-------|----------|-------------|
| `.cx/memory.db` exists | Warning | Yes (`cx index rebuild`) |
| `.cx/memory.db` schema version matches current expected version | Warning | Yes (`cx index rebuild`) |
| Memory sync conflict: same ID exists in both `.cx/memory.db` and `docs/memory/` with different content | Warning | No |

**Memory sync conflict check behavior:**
- Compares IDs in local `.cx/memory.db` against files in `docs/memory/{observations,decisions}/`
- For each ID present in both with differing `content` field: emit warning "memory sync conflict — local and shared versions of memory `<id>` differ"
- Does not block — prints warnings only, exits 0
- Mirrors the conflict detection in `cx memory pull`

### `~/.cx/index.db` Health Check

| Check | Severity | Auto-fixable |
|-------|----------|-------------|
| `~/.cx/index.db` exists | Warning | Yes (bootstrapped on next `cx init` or `cx index rebuild`) |

## MODIFIED Requirements

### `docs/ Structure` Check — `cx.yaml` Validation

- Previous: `cx doctor` checked for `DIRECTION.md` at `docs/memories/DIRECTION.md` as a required file (Error, auto-fixable)
- Modified: `cx doctor` validates `.cx/cx.yaml` when present (Warning if it fails to parse); no longer checks for `DIRECTION.md` as a required file
- The `docs/memory/DIRECTION.md` file (if it exists) is checked as part of memory health, not as a required structural file

### Memory File Health — Directory Paths

- Previous spec referenced `docs/memories/{observations,decisions,sessions}/` as the validated directories
- Modified: validated directories are `docs/memory/{observations,decisions,sessions}/` (singular, matching codebase)

## REMOVED Requirements

### `docs/memories/DIRECTION.md` Required File Check

- Previous: `cx doctor` emitted an Error if `docs/memories/DIRECTION.md` was missing or empty
- Removed: this check is no longer present; the memory policy file is optional, not required
- Replaced by: `.cx/memory.db` existence check (Warning) and memory sync conflict check (Warning)
