# Spec: Search & Index

CX provides a unified search command that queries an FTS5 index covering all of `docs/`. The index is a local cache that rebuilds from the markdown files on demand.

---

## cx search

```bash
cx search "query"                    # search everything in docs/
cx search "query" --memory           # only memory entities (from .cx/memory.db)
cx search "query" --specs            # only docs/specs/
cx search "query" --changes          # only docs/changes/
cx search "query" --personal         # include personal notes from ~/.cx/memory.db
cx search "query" --include-deprecated  # include deprecated observations/decisions
cx search "query" --memory --all-projects  # federated cross-project memory search
```

### Default behavior

- Searches all of `docs/` (equivalent to `--all`)
- Excludes deprecated observations and superseded/cancelled decisions
- Returns results ranked by FTS5 relevance score
- Output includes: file path, entity type (if memory), title/heading, and a content snippet

### Output format

```
cx search "mqtt"

  docs/memory/observations/2026-02-21T10-00-00-angel-mqtt-drops.md
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
| `--memory` | Memory entities from `.cx/memory.db` FTS5 (`memories_fts`) |
| `--specs` | `docs/specs/` |
| `--changes` | `docs/changes/` |
| `--all` | Everything in `docs/` via `.cx/.index.db` + memory entities from `.cx/memory.db` (default) |
| `--personal` | Also search `~/.cx/memory.db` (personal notes — local only, never in docs/) |
| `--include-deprecated` | Include deprecated/superseded/cancelled entities |
| `--type <t>` | Memory entities only: `observation`, `decision`, `session` |
| `--author <a>` | Memory entities only: filter by author |
| `--all-projects` | Federate search across all registered projects (requires `--memory`); opens `~/.cx/index.db` for project paths, merges results with project attribution |

Filters can be combined: `cx search "mqtt" --memory --type observation --author angel`

### Personal notes

`--personal` queries `~/.cx/memory.db` (the local personal notes SQLite database) in addition to the docs/ index. Personal note results are shown with a `[PERSONAL]` badge:

```
cx search "hono" --personal

  docs/memory/observations/2026-03-01T14-00-00-angel-hono-middleware.md
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

Rebuilds both `.cx/.index.db` (docs FTS5 cache) and `.cx/memory.db` (memory entities) from scratch.

### What gets indexed

**`.cx/.index.db`** (docs FTS5 cache — non-memory docs):

| Source | How it's indexed |
|--------|-----------------|
| `docs/specs/**/*.md` | H1 as title, full body for FTS5, path as spec area identifier |
| `docs/architecture/**/*.md` | H1 as title, full body for FTS5 |
| `docs/changes/**/*.md` | H1 as title, full body for FTS5, parent directory as change name |
| `docs/overview.md` | H1 as title, full body for FTS5 |
| `docs/memory/DIRECTION.md` | Indexed but not returned in search by default (policy doc, not knowledge) |

**`.cx/memory.db`** (memory entities — full re-ingest via `RebuildFromMarkdown`):

| Source | How it's indexed |
|--------|-----------------|
| `docs/memory/observations/*.md` | Frontmatter parsed, H1 as title, full body for FTS5 |
| `docs/memory/decisions/*.md` | Frontmatter parsed, H1 as title, sections extracted, full body for FTS5 |
| `docs/memory/sessions/*.md` | Frontmatter parsed, H1 as title, sections extracted, full body for FTS5 |

Files in `docs/archive/` and `docs/masterfiles/` are NOT indexed — they're historical.

### Index schemas

**`.cx/.index.db`** — docs FTS5 cache (specs, changes, architecture, overview):

```sql
CREATE TABLE indexed_docs (
    id          TEXT PRIMARY KEY,     -- relative path from project root
    doc_type    TEXT NOT NULL,        -- spec | architecture | change | overview
    title       TEXT NOT NULL,        -- H1 heading
    content     TEXT NOT NULL,        -- full markdown body
    author      TEXT,
    created_at  TEXT,
    change_id   TEXT,                 -- parent directory (changes only)
    tags        TEXT,
    deprecated  INTEGER DEFAULT 0,
    status      TEXT
);

CREATE VIRTUAL TABLE docs_fts USING fts5(
    title, content, tags, doc_type,
    content=indexed_docs, content_rowid=rowid
);
```

**`.cx/memory.db`** — memory entities (observations, decisions, sessions):

```sql
CREATE TABLE memories (
    id          TEXT PRIMARY KEY,
    entity_type TEXT NOT NULL,        -- observation | decision
    visibility  TEXT NOT NULL,        -- personal | project
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,
    author      TEXT NOT NULL,
    source      TEXT,
    change_id   TEXT,
    file_refs   TEXT,                 -- JSON array
    spec_refs   TEXT,                 -- JSON array
    tags        TEXT,
    deprecated  INTEGER DEFAULT 0,
    deprecates  TEXT,
    status      TEXT,                 -- active | superseded | cancelled (decisions only)
    created_at  TEXT NOT NULL,
    shared_at   TEXT
);

CREATE VIRTUAL TABLE memories_fts USING fts5(
    title, content, tags, entity_type,
    content=memories, content_rowid=rowid
);
```

`cx search "query" --memory` queries `memories_fts` in `.cx/memory.db`. `cx search "query" --specs` (and other doc-scope flags) query `docs_fts` in `.cx/.index.db`. Both indexes are queried independently and results merged when `--memory` is combined with other scope flags.

### Rebuild triggers

1. **Proactive** — `post-merge` git hook calls `cx index rebuild`
2. **Lazy** — any `cx` command compares file mtimes against `last_indexed_at`
3. **Explicit** — developer or agent calls `cx index rebuild`

---

## cx context --load

Returns structured output for a specific resource. Used by the Primer during context priming.

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

## Cross-Project Search

`cx search "query" --memory --all-projects` federates search across all registered projects:
- Opens `~/.cx/index.db` to get project paths
- Opens each project's `.cx/memory.db`
- Merges and ranks results with project attribution

```
cx search "mqtt" --memory --all-projects

  [observation:discovery | project: my-project] MQTT broker drops messages over 256KB
    "...must chunk large payloads..."

  [observation:pattern | project: other-project] MQTT retry with exponential backoff
    "...pattern proven effective under high load..."

  2 results (2 projects searched)
```

---

## cx memory search

`cx memory search "query"` queries `.cx/memory.db` FTS5 directly (`memories_fts` virtual table). This is the recommended way for agents to search memory entities. Supports `--change <name>`, `--type <t>`, `--include-deprecated`, and `--all-projects` flags.

`cx search "query" --memory` is the equivalent unified search command.

```bash
cx memory search "query"                        # search memory entities
cx memory search "query" --change <name>        # scoped to a change
cx memory search "query" --type observation     # filter by type
cx memory search "query" --all-projects         # cross-project federation
cx memory list --type decision                  # list all active decisions
cx memory list --type session --change <name>   # list sessions for a change
cx memory list --type observation --recent 7d  # list recent observations
```
