# Spec: Search & Index

CX provides a unified search command that queries an FTS5 index covering all of `docs/`. The index is a local cache that rebuilds from the markdown files on demand.

---

## cx search

```bash
cx search "query"                    # search everything in docs/
cx search "query" --memory           # only docs/memories/
cx search "query" --specs            # only docs/specs/
cx search "query" --changes          # only docs/changes/
cx search "query" --personal         # include personal notes from ~/.cx/memory.db
cx search "query" --include-deprecated  # include deprecated observations/decisions
```

### Default behavior

- Searches all of `docs/` (equivalent to `--all`)
- Excludes deprecated observations and superseded/cancelled decisions
- Returns results ranked by FTS5 relevance score
- Output includes: file path, entity type (if memory), title/heading, and a content snippet

### Output format

```
cx search "mqtt"

  docs/memories/observations/2026-02-21T10-00-00-angel-mqtt-drops.md
    [observation:discovery] MQTT broker silently drops messages over 256KB
    "...must chunk large payloads. Discovered when testing with 500 devices..."

  docs/specs/device-communication/spec.md
    [spec] § Device Communication
    "...MQTT used for all device-to-server telemetry. Messages published to..."

  docs/changes/fix-gas-threshold/design.md
    [change] § Design: fix-gas-threshold
    "...MQTT retry logic needs exponential backoff to avoid broker overload..."

  3 results (1 deprecated result hidden, use --include-deprecated to show)
```

### Filters

| Flag | Scope |
|------|-------|
| `--memory` | `docs/memories/observations/`, `docs/memories/decisions/`, `docs/memories/sessions/` |
| `--specs` | `docs/specs/` |
| `--changes` | `docs/changes/` |
| `--all` | Everything in `docs/` (default) |
| `--personal` | Also search `~/.cx/memory.db` (personal notes — local only, never in docs/) |
| `--include-deprecated` | Include deprecated/superseded/cancelled entities |
| `--type <t>` | Memory entities only: `observation`, `decision`, `session` |
| `--author <a>` | Memory entities only: filter by author |

Filters can be combined: `cx search "mqtt" --memory --type observation --author angel`

### Personal notes

`--personal` queries `~/.cx/memory.db` (the local personal notes SQLite database) in addition to the docs/ index. Personal note results are shown with a `[PERSONAL]` badge:

```
cx search "hono" --personal

  docs/memories/observations/2026-03-01T14-00-00-angel-hono-middleware.md
    [observation:pattern] Hono middleware structured as separate files per concern
    "...each middleware gets its own file under src/middleware/..."

  ~/.cx/memory.db
    [PERSONAL:preference] I prefer Hono middleware as separate files
    "...always split middleware into individual files rather than one big..."

  2 results
```

Personal notes are never included in default search — `--personal` must be explicitly specified. They are also surfaced through context priming (BUILD and CONTINUE modes) without needing `--personal`.

---

## cx index rebuild

```bash
cx index rebuild
```

Rebuilds the FTS5 index from scratch. Reads every `.md` file in `docs/` and indexes it into `.cx/.index.db`.

### What gets indexed

| Source | How it's indexed |
|--------|-----------------|
| `docs/memories/observations/*.md` | Frontmatter parsed, H1 as title, full body for FTS5 |
| `docs/memories/decisions/*.md` | Frontmatter parsed, H1 as title, sections extracted, full body for FTS5 |
| `docs/memories/sessions/*.md` | Frontmatter parsed, H1 as title, sections extracted, full body for FTS5 |
| `docs/specs/**/*.md` | H1 as title, full body for FTS5, path as spec area identifier |
| `docs/architecture/**/*.md` | H1 as title, full body for FTS5 |
| `docs/changes/**/*.md` | H1 as title, full body for FTS5, parent directory as change name |
| `docs/overview.md` | H1 as title, full body for FTS5 |
| `docs/memories/DIRECTION.md` | Indexed but not returned in search by default (policy doc, not knowledge) |

Files in `docs/archive/` and `docs/masterfiles/` are NOT indexed — they're historical.

### Index schema

```sql
CREATE TABLE indexed_docs (
    id          TEXT PRIMARY KEY,     -- relative path from project root
    doc_type    TEXT NOT NULL,        -- observation | decision | session | spec | architecture | change | overview
    title       TEXT NOT NULL,        -- H1 heading
    content     TEXT NOT NULL,        -- full markdown body
    author      TEXT,                 -- frontmatter (memory entities only)
    created_at  TEXT,                 -- frontmatter (memory entities only)
    change_id   TEXT,                 -- frontmatter or parent directory
    tags        TEXT,                 -- frontmatter (memory entities only)
    deprecated  INTEGER DEFAULT 0,   -- 1 if deprecated by another entity
    status      TEXT,                 -- active | superseded | cancelled (decisions only)
    -- memory-specific fields
    entity_subtype TEXT,             -- bugfix | discovery | pattern | context (observations only)
    spec_refs   TEXT,                -- JSON array (memory entities only)
    file_refs   TEXT,                -- JSON array (observations only)
    deprecates  TEXT                 -- slug this entity deprecates (unified — no separate supersedes)
);

CREATE VIRTUAL TABLE docs_fts USING fts5(
    title, content, tags, doc_type,
    content=indexed_docs, content_rowid=rowid
);
```

### Rebuild triggers

1. **Proactive** — `post-merge` git hook calls `cx index rebuild`
2. **Lazy** — any `cx` command compares file mtimes against `last_indexed_at`
3. **Explicit** — developer or agent calls `cx index rebuild`

---

## cx context --load

Returns structured output for a specific resource. Used by the primer subagent during context priming.

```bash
cx context --load spec <area>
cx context --load scenarios <area>
cx context --load change <name>
cx context --load architecture
cx context --load overview
cx context --load direction
cx context --load decision <slug>
cx context --load observation <slug>
cx context --load session <slug>
```

### Structured output format

Every `--load` response includes a metadata header:

```markdown
<!-- cx:load | type: spec | path: docs/specs/gas-monitoring/spec.md | modified: 2026-02-20T14:30:00Z -->
<!-- cx:load | active_deltas: fix-gas-threshold -->

# Gas Monitoring

<file content follows>
```

The metadata header tells the primer:
- What type of resource this is
- Where it lives on disk
- When it was last modified
- Whether any active changes have deltas against it

For changes, `--load change <name>` returns all files in the change directory concatenated with separators:

```markdown
<!-- cx:load | type: change | path: docs/changes/fix-gas-threshold/ -->

<!-- file: proposal.md -->
<proposal content>

<!-- file: design.md -->
<design content>

<!-- file: tasks.md -->
<tasks content>

<!-- file: specs/gas-monitoring/delta.md -->
<delta content>
```

---

## Migration: cx memory search

`cx memory search` is replaced by `cx search --memory`. For backwards compatibility during the transition:

```bash
cx memory search "query"           →  cx search "query" --memory
cx memory search --type decision   →  cx search --memory --type decision
cx memory search --author angel    →  cx search --memory --author angel
cx memory decisions                →  cx search --memory --type decision
cx memory decisions --all          →  cx search --memory --type decision --include-deprecated
```

The old commands remain as aliases until removed in a future version.
