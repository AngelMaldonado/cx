---
name: search
type: delta-spec
area: search
change: memory-architecture
---

## ADDED Requirements

### `--all-projects` Flag

- `cx search "query" --memory --all-projects` — federates search across all registered projects
- Opens `~/.cx/index.db` to get project paths; opens each project's `.cx/memory.db`; merges and ranks results with project attribution
- Results include a project identifier in the output badge: `[observation:discovery | project: my-project]`

### Memory Search Now Hits `.cx/memory.db`

- `cx search "query" --memory` queries `.cx/memory.db` FTS5 (`memories_fts`) instead of `.cx/.index.db`
- `.cx/.index.db` continues to serve non-memory doc search (specs, changes, architecture, overview)
- The two indexes are queried independently and results merged when `--memory` is combined with other scope flags

## MODIFIED Requirements

### `cx memory search` Alias Source Change

- Previous: `cx memory search` was an alias that hit `.cx/.index.db` for memory entity results
- Modified: `cx memory search` now hits `.cx/memory.db` FTS5 directly for memory queries; the alias behavior is preserved but the underlying query target changes

### `cx index rebuild` — Also Populates `.cx/memory.db`

- Previous: `cx index rebuild` only rebuilt `.cx/.index.db` from all `docs/` markdown
- Modified: `cx index rebuild` also populates `.cx/memory.db` from `docs/memory/{observations,decisions,sessions}/` markdown files (full re-ingest via `RebuildFromMarkdown`)
- The two indexes (`.index.db` for docs, `memory.db` for memory entities) are both rebuilt by `cx index rebuild`

### `--memory` Filter Scope

- Previous: `--memory` scoped search to `docs/memories/observations/`, `docs/memories/decisions/`, `docs/memories/sessions/` in `.index.db`
- Modified: `--memory` scopes search to `.cx/memory.db`; the directory paths for the underlying markdown files remain `docs/memory/` (singular) matching the codebase

### `cx index rebuild` What Gets Indexed — Memory Path

- Previous spec listed `docs/memories/observations/*.md`, `docs/memories/decisions/*.md`, `docs/memories/sessions/*.md` as indexed sources
- Modified: source paths are `docs/memory/observations/*.md`, `docs/memory/decisions/*.md`, `docs/memory/sessions/*.md` (singular)

## REMOVED Requirements

None. All existing search filters, output format, and index rebuild triggers remain in force.
