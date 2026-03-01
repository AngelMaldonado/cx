# Spec: Memory

The CX memory system stores four distinct entity types, each with different lifecycles, audiences, and query patterns. Project memory (observations, decisions, sessions) lives in `docs/memories/` and is committed to git. Personal notes are local-only SQLite.

See diagrams: [entity relationships](diagrams/01-entity-relationships.mermaid) · [storage architecture](diagrams/02-memory-storage.mermaid) · [entity lifecycles](diagrams/03-entity-lifecycles.mermaid) · [context composition](diagrams/04-context-composition.mermaid)

---

## Memory Direction — docs/memories/DIRECTION.md

Every project ships a `DIRECTION.md` inside `docs/memories/`. This file is written by the team and read by the agent at save-time to decide whether something is worth recording. It answers one question: **what counts as signal vs. noise in this project?**

`cx init` generates a default `DIRECTION.md` template. Teams are expected to refine it as they learn what kinds of observations actually prove useful.

### What DIRECTION.md contains

**1. What to always save**
Explicit categories of things this project considers high-value. Examples:
- Firmware behaviors that contradict the datasheet
- Any constraint discovered about an external system (MQTT broker, third-party API, OS)
- The rationale behind any interface contract (why this field is an INT not a FLOAT)
- Performance characteristics measured empirically

**2. What to never save**
Noise filters specific to this project. Examples:
- Anything already captured in Linear (don't duplicate issue descriptions)
- Standard library behavior that's in official docs
- Implementation details visible in the code itself (redundant with reading the file)
- Status updates ("started working on X") — use session summaries for that

**3. Threshold heuristics**
Rules of thumb agents use when it's not obvious. The default template ships with:
- *Would a new team member need to know this to avoid hitting the same wall?*
- *Is this a constraint that can't be inferred from the code or docs?*
- *Would I be frustrated to rediscover this in 3 months?*

If the answer to any of these is yes, save it.

**4. Project-specific type guidance**
Which observation types apply to this project and what each means in context. A firmware project may care heavily about `discovery` (undocumented hardware behaviors). A web API project may care more about `bugfix` and `pattern`.

### File format

```markdown
# Memory Direction — <project name>

## Always Save
- <category>: <why it matters here>
- ...

## Never Save
- <category>: <why it's noise here>
- ...

## Threshold Test
When unsure, ask:
- <heuristic 1>
- <heuristic 2>

## Type Guidance
- **bugfix**: <what counts as a bugfix worth recording in this project>
- **discovery**: <what counts as a discovery here>
- **pattern**: <what counts as a reusable pattern here>
- **context**: <what background context is worth preserving here>
```

### How agents use DIRECTION.md

Before calling `cx memory save`, the agent reads `docs/memories/DIRECTION.md` and applies its rules to the candidate observation. If it clearly falls into a "never save" category, the agent skips it. If it matches an "always save" category, the agent saves without deliberation. For everything in between, the threshold heuristics decide.

`cx doctor` warns if `docs/memories/DIRECTION.md` is missing or empty — it is considered a required project file alongside `overview.md`.

---

## Entity 1: Observation

**What it is**: Something that happened or was learned during development work.

**Examples**:
- "Fixed N+1 query in /devices endpoint — added JOIN for sensor readings, reduced response from 2s to 50ms"
- "MQTT broker silently drops messages over 256KB — must chunk large payloads"
- "iOS CoreBluetooth requires Background Modes capability for BLE scanning to work in background"

**Lifecycle**: Created once, never modified, never deleted. An observation is a historical fact. If the information becomes outdated, a new observation can **deprecate** it — the old file stays untouched but is excluded from context output.

**Audience**: The whole team working on this project.

**When it's created**: Agent runs `cx memory save` after completing significant work — fixing a bug, discovering a constraint, hitting a wall.

### Fields

```
id          : string     — unique slug (auto from timestamp + title)
type        : enum       — bugfix | discovery | pattern | context
title       : string     — one-line summary (used in compact listings)
content     : string     — full description (markdown)
author      : string     — from git config user.name
created_at  : datetime   — when it was recorded

session_id  : string?    — which session produced it
change_id   : string?    — which change it relates to (e.g., "add-ble-pairing")
deprecates  : string?    — slug of an older observation this one replaces
file_refs   : string[]?  — codebase files referenced
spec_refs   : string[]?  — spec areas affected
tags        : string[]?  — free-form tags for search
```

### Type Definitions

| Type | Meaning | Example |
|------|---------|---------|
| `bugfix` | A bug was found and resolved | "N+1 query in /devices endpoint" |
| `discovery` | A constraint or behavior was learned | "MQTT drops messages over 256KB" |
| `pattern` | A reusable approach was identified | "Use Zephyr work queues for async BLE events" |
| `context` | Background information worth recording | "Sensor readings arrive every 60s, not 30s as documented" |

### File Format

Stored as `docs/memories/observations/<date>-<author>-<slug>.md`:

```markdown
---
type: bugfix
author: angel
created: 2026-02-20T14:30:00Z
change: add-ble-pairing
files: [src/services/devices.ts, src/db/queries.ts]
specs: [device-communication]
tags: [performance, database]
---

# Fixed N+1 query in /devices endpoint

<full description>
```

### Deprecation

When an observation becomes outdated, write a new observation with `deprecates` in its frontmatter:

```markdown
---
type: discovery
author: angel
created: 2026-08-15T10:00:00Z
deprecates: 2026-02-21T10-00-00-angel-mqtt-drops
tags: [mqtt]
---

# MQTT broker upgraded — 256KB limit no longer applies

Upgraded to EMQX 5.x which handles messages up to 100MB.
The previous chunking workaround is no longer needed.
```

The old file is never touched. The binary sees `deprecates: <slug>` and marks the referenced observation as deprecated in the index:
- **Excluded** from `cx context` output (won't pollute the primer's map)
- **De-prioritized** in `cx memory search` results (shown at the bottom, marked as deprecated)
- **Still findable** with `cx memory search --include-deprecated "mqtt"`
- **Chain is traceable** — the new observation explains *why* the old one no longer applies

---

## Entity 2: Decision

**What it is**: A deliberate choice that was made, with rationale and alternatives considered.

**Examples**:
- "Chose TimescaleDB over InfluxDB for sensor telemetry storage"
- "BLE pairing will use Just Works mode, not Passkey, for consumer simplicity"
- "Alert thresholds stored in device config, not server-side, for offline operation"

**Lifecycle**: Created once. Can be **deprecated** by a new decision (the old decision's status becomes `superseded` in the index), or **cancelled** when a decision is retired without a replacement. Never modified directly — the history of decisions and why they changed is valuable.

**Audience**: The whole team. Decisions prevent teams from re-debating settled questions and help new members understand *why* things are the way they are.

**When it's created**: Agent runs `cx memory decide` when a meaningful technical choice is made — architecture, library selection, approach, convention.

### Fields

```
id              : string     — unique slug
title           : string     — what was decided (one line)
context         : string     — why this decision was needed
outcome         : string     — what was chosen
alternatives    : string[]   — what else was considered
rationale       : string     — why this option won
tradeoffs       : string[]   — known downsides accepted
author          : string     — who made the decision
created_at      : datetime

deprecates      : string?    — slug of an older entity this one replaces (any type)
status          : enum       — active | superseded | cancelled
change_id       : string?    — which change prompted this
spec_refs       : string[]?  — spec areas affected
tags            : string[]?  — free-form tags
```

### Deprecation (Unified)

All entity linking uses a single field: `deprecates`. There is no separate `supersedes` field. When a decision replaces another decision, it uses `deprecates: <old-decision-slug>`. When a decision invalidates an observation, it uses the same field.

The `status` field in the index tracks the outcome:

| Status | Meaning | How it's set |
|--------|---------|-------------|
| `active` | This is the current decision — the team follows it | Written in the file's frontmatter |
| `superseded` | A newer decision replaces this one | Set **in the index only** when another decision deprecates this one |
| `cancelled` | Retired without replacement — the constraint no longer applies | Written in the file's frontmatter |

**Decision replacing a decision:**

```markdown
---
author: angel
created: 2026-08-01T10:00:00Z
status: active
deprecates: 2026-02-22T10-00-00-angel-rest-for-all
change: add-mqtt-layer
specs: [device-communication]
tags: [architecture, api]
---

# Device communication uses MQTT, user-facing APIs stay REST

## Context
The original "REST for everything" decision doesn't account for real-time device telemetry...

## Outcome
Device-to-server communication uses MQTT. User-facing APIs stay REST.

## Alternatives
...

## Rationale
...

## Tradeoffs
...
```

Index behavior:
- Old decision (`rest-for-all`): `deprecated=1`, `status='superseded'` (in index only — file is never touched)
- New decision: `deprecated=0`, `status='active'`

**Decision deprecating an observation:**

```markdown
---
author: angel
created: 2026-08-01T10:00:00Z
status: active
deprecates: 2026-02-21T10-00-00-angel-mqtt-drops
change: upgrade-mqtt-broker
specs: [device-communication]
tags: [mqtt, architecture]
---

# Switch to EMQX 5.x removes payload size constraints

## Context
The original MQTT 256KB limit observation drove our chunking architecture decision...

## Outcome
With EMQX 5.x, we remove the chunking layer entirely...

## Alternatives
...

## Rationale
...

## Tradeoffs
...
```

Index behavior:
- Old observation (`mqtt-drops`): `deprecated=1`, `status` stays null (observations don't have status)
- New decision: `deprecated=0`, `status='active'`

**Decision cancelled (no replacement):**

```markdown
---
author: angel
created: 2026-09-15T10:00:00Z
status: cancelled
change: remove-offline-mode
specs: [device-communication]
tags: [offline]
---

# Cancel: Alert thresholds in device config for offline operation

## Context
We decided to store alert thresholds in device config for offline operation, but offline mode has been removed from the product scope.

## Outcome
This decision is cancelled — thresholds move to server-side config.

## Alternatives
...

## Rationale
...

## Tradeoffs
...
```

A cancelled decision doesn't need to `deprecates` anything — the `status: cancelled` alone causes it to be excluded from default context output. If it does deprecate another entity, both behaviors apply.

`cx doctor` validates that `status` is one of `active`, `superseded`, or `cancelled`.

### File Format

Stored as `docs/memories/decisions/<date>-<author>-<slug>.md`.

All five body sections are **required**. `cx doctor` will warn on any decision file missing one. `cx memory decide` will refuse to write a file with missing sections.

```markdown
---
author: angel
created: 2026-02-22T10:00:00Z
status: active
deprecates: null
change: add-ble-pairing
specs: [device-communication]
tags: [architecture, ble, pairing]
---

# BLE pairing will use Just Works mode

## Context
<why this decision was needed — what problem prompted it>

## Outcome
<what was chosen — one clear statement>

## Alternatives
- **Option A**: <what it was and why it lost>
- **Option B**: <what it was and why it lost>

## Rationale
<why the chosen option won over the alternatives>

## Tradeoffs
- <known downside or risk accepted with this choice>
```

---

## Entity 3: Session

**What it is**: A summary of a single coding session — what was attempted, accomplished, and what to do next.

**Lifecycle**: Created when a session ends or when context is about to compact. Never modified after creation.

**Audience**: Primarily the same developer (session continuity), visible to team (so others know what's being worked on).

**When it's created**: Agent runs `cx memory session` when ending work or when context compaction is detected.

### Fields

```
id              : string     — auto-generated
author          : string     — who ran the session
started_at      : datetime   — session start
ended_at        : datetime   — session end

change_id       : string?    — which change was active
goal            : string     — what the session aimed to do
discoveries     : string[]   — what was learned (can become observations)
accomplished    : string[]   — what was completed
blockers        : string[]?  — what got in the way
files_touched   : string[]   — files modified during session
next_steps      : string     — what to do next (critical for session recovery)
```

### Session Recovery Flow

```
New session starts
    → cx memory context
        → Find latest session for this author + project
        → Return: goal, accomplished, blockers, next_steps
        → Agent has full continuity without reading chat history
```

### File Format

Stored as `docs/memories/sessions/<date>-<author>.md`:

```markdown
---
author: angel
change: add-ble-pairing
started: 2026-02-22T09:00:00Z
ended: 2026-02-22T12:30:00Z
---

# Session: BLE pairing state machine

## Goal
<what the session aimed to do>

## Accomplished
- <completed item>

## Discoveries
- <what was learned>

## Blockers
- <what stopped progress>

## Files Touched
- src/ble/service.c

## Next Steps
<exact pickup point for next session>
```

---

## Entity 4: Personal Note (LOCAL ONLY)

**What it is**: Personal knowledge that spans projects. Things about *you* and *how you work*, not about any specific codebase.

**Examples**:
- "I prefer Hono middleware structured as separate files per concern"
- "Zephyr's k_sleep takes milliseconds not seconds — I always forget this"
- "When debugging BLE on macOS, use PacketLogger not Wireshark"

**Lifecycle**: Can be created, updated (upsert via `topic_key`), or deleted. These evolve as you learn.

**Audience**: You only. Never committed, never synced.

**When it's created**: Agent runs `cx memory note` when it notices a personal pattern, preference, or recurring mistake.

### Fields

```
id              : string     — auto-generated
type            : enum       — pattern | preference | tool_tip | reminder
title           : string     — short description
content         : string     — full content
author          : string     — from git config user.name
created_at      : datetime
updated_at      : datetime   — for upserts

topic_key       : string?    — for upsert dedup (same key = update, not append)
projects        : string[]?  — which projects this is relevant to (optional)
tags            : string[]?  — searchable tags
```

### Topic Key Upsert

Personal notes can evolve. If you save a note with `topic_key: "hono-middleware-pattern"`, and later save another with the same topic_key, the old one gets replaced. This prevents accumulating outdated personal preferences.

Project-scoped entities (observations, decisions, sessions) are NEVER upserted — they are append-only history.

### Storage

Personal notes live in `~/.cx/memory.db` (SQLite). This is the one place where SQLite is the source of truth, not a cache.

```sql
CREATE TABLE personal_notes (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    type        TEXT NOT NULL CHECK(type IN ('pattern', 'preference', 'tool_tip', 'reminder')),
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,
    topic_key   TEXT UNIQUE,
    projects    TEXT,                  -- JSON array
    tags        TEXT,                  -- comma-separated
    created_at  TEXT DEFAULT (datetime('now')),
    updated_at  TEXT DEFAULT (datetime('now'))
);

CREATE VIRTUAL TABLE personal_notes_fts USING fts5(
    title, content, tags,
    content=personal_notes, content_rowid=id
);
```

---

## Storage Model

### Project Memory — Git-Native Markdown

```
docs/memories/
├── DIRECTION.md        ← committed (memory policy for this project)
├── observations/       ← committed (team knowledge)
├── decisions/          ← committed (team knowledge)
└── sessions/           ← committed (team knowledge)

.cx/                    ← fully gitignored, local cache only
└── .index.db           ← FTS5 index, rebuilt from all docs/ on demand
```

**Team sync is just git.** `git push` shares your memories. `git pull` gets the team's. No S3, no remote server, no custom sync protocol. Memory lives in `docs/` — the single source of truth.

When teammates' memory files arrive via `git pull`, a `post-merge` git hook triggers `cx index rebuild` to update the local FTS5 cache. If hooks aren't installed, the binary still rebuilds lazily on the next command (mtime check). See [architecture — git hooks](../../architecture/index.md#git-hooks) for hook details.

### Index Cache

`.cx/.index.db` is a SQLite database with FTS5 that indexes all markdown in `docs/` (not just memories — see [search spec](../search/spec.md) for the full index schema). Memory entities are stored in the `indexed_docs` table alongside specs, changes, and architecture docs. It is a **read cache**, never the source of truth. `.cx/` is fully gitignored — nothing in it is ever committed.

The schema below shows the memory-specific columns. For the full unified index schema covering all doc types, see the [search spec](../search/spec.md).

```sql
CREATE TABLE indexed_entities (
    id          TEXT PRIMARY KEY,     -- filename slug
    entity_type TEXT NOT NULL,        -- observation | decision | session
    title       TEXT NOT NULL,
    content     TEXT NOT NULL,        -- full markdown body
    author      TEXT NOT NULL,
    created_at  TEXT NOT NULL,
    change_id   TEXT,
    file_refs   TEXT,                 -- JSON array
    spec_refs   TEXT,                 -- JSON array
    tags        TEXT,                 -- comma-separated
    deprecated  INTEGER DEFAULT 0,   -- 1 if another entity deprecates this one
    deprecates  TEXT,                 -- entity slug this one replaces (any entity type)
    status      TEXT                  -- active | superseded | cancelled (decisions only; superseded set by index, not file)
);

CREATE VIRTUAL TABLE entities_fts USING fts5(
    title, content, tags, entity_type,
    content=indexed_entities, content_rowid=rowid
);
```

During index rebuild, after all files are inserted, the binary runs a second pass (see [Deprecation indexing](#deprecation-indexing) for full rules). Default queries filter with `WHERE deprecated = 0`.

Rebuild triggers:
1. **Proactive** — `post-merge` git hook calls `cx index rebuild` when memory files arrive via `git pull`
2. **Lazy** — on any `cx` command, compare file mtimes in `docs/memories/{observations,decisions,sessions}/` against `last_indexed_at`; rebuild if any file is newer

For a team of 3–5 with hundreds of files, full reindex takes <100ms.

---

## Parsing & Reconstruction

This defines how the binary reads memory files and rebuilds the FTS5 index. Every rule here is a contract — if the file format deviates from it, parsing fails.

### Which files are parsed

The binary only indexes files inside exactly three directories:

```
docs/memories/observations/
docs/memories/decisions/
docs/memories/sessions/
```

No other file in `docs/` is ever read as a memory entity — not `DIRECTION.md`, not specs, not architecture docs. The directory name determines the entity type; no type field is needed in decisions or sessions.

### Filename convention

```
<ISO-date>T<time>-<author>-<slug>.md

2026-02-22T10-00-00-angel-ble-just-works-pairing.md
```

- Date/time: ISO 8601, colons replaced with hyphens (filesystem-safe)
- Slug: derived from H1 title — lowercase, hyphens, max 40 chars
- Generated by `cx memory save/decide/session` — never written by hand
- The filename IS the entity ID; no separate ID field exists

### Parsing rules

```
1. Frontmatter  — YAML between the opening and closing ---
                  Must appear at the very top of the file
                  Malformed YAML → file skipped, cx doctor warns

2. Title        — First H1 (# ...) after the frontmatter
                  Required in all entity types
                  Used as the display title and compact listing text

3. Sections     — ## level headings only
                  Content is everything from the heading to the next ## or EOF
                  Section order in the file does not matter
                  The binary finds sections by exact heading name (case-insensitive)

4. Body (FTS5)  — Everything after the frontmatter block, including all sections
                  Indexed verbatim for full-text search
```

### What the binary extracts per type

| Field | Observation | Decision | Session |
|-------|-------------|----------|---------|
| ID | filename slug | filename slug | filename slug |
| Type | `type` in frontmatter | inferred from directory | inferred from directory |
| Title | H1 | H1 | H1 |
| Author | `author` frontmatter | `author` frontmatter | `author` frontmatter |
| Date | `created` frontmatter | `created` frontmatter | `started` frontmatter |
| Tags | `tags` frontmatter | `tags` frontmatter | — |
| Change | `change` frontmatter | `change` frontmatter | `change` frontmatter |
| Spec refs | `specs` frontmatter | `specs` frontmatter | — |
| File refs | `files` frontmatter | — | — |
| Deprecates | `deprecates` frontmatter | `deprecates` frontmatter | — |
| Status | — | `status` frontmatter | — |
| FTS body | full body | full body | full body |
| Compact extract | title only | `## Outcome` section | `## Goal` + `## Next Steps` + `## Blockers` |

The **compact extract** is what appears in `cx context` output. The binary reads only these specific fields for context assembly — not the full file — keeping context output tight and token-efficient.

### Deprecation indexing

When the binary indexes an entity with `deprecates: <slug>`, it marks the referenced entity as `deprecated = 1` in the index. The `deprecates` field works across entity types — a decision can deprecate an observation, an observation can deprecate another observation, etc.

During the second pass of index rebuild:
1. For every row with a non-null `deprecates`, set `deprecated = 1` on the referenced entity
2. **If the deprecating entity is a decision AND the referenced entity is also a decision**, additionally set the old decision's `status = 'superseded'` in the index (the file is never modified)
3. **If the deprecating entity is an observation AND the referenced entity is a decision**, set `deprecated = 1` but leave `status` as-is in the index
4. Decisions with `status: cancelled` (written in the file) are also treated as deprecated — excluded from default context

The indexing rules:
- The deprecated entity's file is never modified — flags live only in the index
- If the deprecating entity is itself deprecated later, both it and the original stay deprecated — chains don't "un-deprecate"
- `cx context` and `cx search` exclude deprecated entities by default
- `cx search --include-deprecated` includes them (shown at the bottom, marked)
- `cx doctor` warns if a `deprecates` slug doesn't match any existing file

### Validation on write

`cx memory save/decide/session` validates before writing:

- All required frontmatter fields are present
- For decisions: all five body sections (`## Context`, `## Outcome`, `## Alternatives`, `## Rationale`, `## Tradeoffs`) are present and non-empty
- For sessions: `## Goal` and `## Next Steps` are present and non-empty
- For entities with `deprecates`: the referenced slug must exist in `docs/memories/{observations,decisions,sessions}/`
- For decisions: `status` must be one of `active`, `superseded`, `cancelled`
- H1 title is present

On failure, the command prints the missing fields and exits non-zero without writing the file.

### Validation on read (`cx doctor`)

- Warns on any file in the three directories that fails to parse
- Warns on any decision missing a required section
- Warns on any file whose frontmatter `author` field doesn't match a known team member
- Does not block on warnings — just reports

---

## Context Priming

Memory is one input to context priming, not the whole picture. The `cx context` command (not `cx memory context`) assembles memory alongside specs, changes, and other docs for session start. This is handled by the **Primer**, not the Master.

See [context-priming spec](../context-priming/spec.md) for the full priming architecture.

### How memory feeds into cx context

| Mode | What `cx context` reads from memory |
|------|-------------------------------------|
| BUILD | Active decisions (compact), observations from last 7 days (compact), personal notes |
| CONTINUE | Session recovery (full), change-scoped observations + decisions (compact), personal notes |
| PLAN | Personal preference notes only |

The Primer receives these as part of the context map, evaluates relevance against the developer's intent, and distills them into the primed context output.

---

## Command Reference

### Project Memory (stored in docs/memories/, committed to git)

| Command | Entity | Action |
|---------|--------|--------|
| `cx memory save` | Observation | Create `docs/memories/observations/*.md` |
| `cx memory decide` | Decision | Create `docs/memories/decisions/*.md` |
| `cx memory session` | Session | Create `docs/memories/sessions/*.md` |

### Local Memory (stored in ~/.cx/memory.db, never synced)

| Command | Entity | Action |
|---------|--------|--------|
| `cx memory note` | Personal Note | Insert/upsert into SQLite |
| `cx memory forget` | Personal Note | Delete from SQLite |

### Query (reads both project + local)

| Command | Action |
|---------|--------|
| `cx search "query" --memory` | FTS5 search over memories (excludes deprecated by default) |
| `cx search "query" --memory --include-deprecated` | Include deprecated/cancelled entities in results |
| `cx search "query" --memory --type <t>` | Filter by entity type |
| `cx search "query" --memory --author <a>` | Filter by author |

See [search spec](../search/spec.md) for the full unified search interface. `cx memory search` remains as an alias during migration.

### Context Priming (called by Primer, not Master)

| Command | Action |
|---------|--------|
| `cx context --mode <mode>` | Return the context map for a session mode |
| `cx context --load <resource> [name]` | Load full content of a specific resource |
