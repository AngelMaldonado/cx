---
name: context-priming
type: delta-spec
area: context-priming
change: memory-architecture
---

## ADDED Requirements

### DB-Backed Memory Loading

- Primer loads memory by querying `.cx/memory.db` via `cx memory search` and `cx memory list` commands
- These commands return structured, ranked results from the FTS5 index — not raw file scans
- Primer passes `--change <name>` to scope memory results for CONTINUE mode
- Primer passes `--type decision` or `--type observation` to filter by entity type

### Mode-Specific Memory Loading Table (formalized)

| Mode | What Primer reads from memory |
|------|-------------------------------|
| BUILD | Active decisions (`cx memory list --type decision`), recent observations (`cx memory list --type observation --recent 7d`), personal notes |
| CONTINUE | Last session (`cx memory list --type session`), change-scoped observations + decisions (`cx memory search --change <name>`) |
| PLAN | Personal preference notes only — no project memory |

### `cx context --mode` as Primary Entry Point

- `cx context --mode <build|continue|plan> [--change <name>]` is the primary Primer entry point; it assembles the full map from all memory + doc sources
- Primer calls `cx context --load` selectively for specific resources after evaluating the map

## MODIFIED Requirements

### Memory Input to `cx context` Map — Source Changed

- Previous: `[DIRECTION]` section in the BUILD and CONTINUE maps loaded `docs/memories/DIRECTION.md`
- Modified: The path for memory direction is `docs/memory/DIRECTION.md` (singular, matching codebase convention)
- Previous: Memory sections in the map were assembled from markdown file scans
- Modified: Memory sections are assembled from `.cx/memory.db` FTS5 queries

### BUILD Map — `[OBSERVATIONS]` Source

- Previous: Last 7 days sourced by scanning `docs/memories/observations/` by mtime
- Modified: Last 7 days sourced by `cx memory list --type observation --recent 7d` against `.cx/memory.db`

### CONTINUE Map — `[SESSION RECOVERY]` Source

- Previous: Latest session found by scanning `docs/memories/sessions/` filenames for most recent mtime
- Modified: Latest session found by `cx memory list --type session --change <name>` sorted by `started_at` descending

### CONTINUE Map — `[CHANGE MEMORY]` Source

- Previous: Scanned `docs/memories/observations/` and `docs/memories/decisions/` for `change: <name>` in frontmatter
- Modified: Queried via `cx memory search --change <name>` against `.cx/memory.db`

## REMOVED Requirements

### Direct File Reading for Memory Priming

- Previous: Primer could read `docs/memories/observations/`, `docs/memories/decisions/`, `docs/memories/sessions/` directly as files
- Removed: Primer must use `cx memory search` and `cx memory list` commands; direct file access for memory priming is no longer the expected pattern
