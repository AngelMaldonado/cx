# Spec: Context Priming

The context priming system is the core of CX. It ensures every agent session starts with exactly the right context — no more, no less — by using a dedicated **Primer** that loads, evaluates, and distills project knowledge before the Master begins work.

---

## Architecture

```
Developer writes opening message
        │
        ▼
Master reads message, spawns Primer
        │ passes: developer's message (verbatim)
        ▼
Primer (disposable context window)
        │
        ├── 0. Check .cx/conflicts.json — if conflicts exist,
        │       spawn Conflict-resolver first (see conflict-resolution spec)
        │       Wait for resolution before proceeding
        ├── 1. Classify → CONTINUE | BUILD | PLAN
        ├── 2. Call cx context --mode <mode> [flags]
        │       → receives the "map": compact listings of everything available
        ├── 3. Evaluate: what from the map is relevant to this session?
        ├── 4. Call cx context --load <resource> for each relevant item
        │       → receives full content of selected specs, changes, etc.
        ├── 5. Distill: compose a focused context block (~500-800 tokens)
        │       stripping noise, keeping only what the Master needs
        └── 6. Return structured context to Master
                │
                ▼
Master receives primed context and decides dispatch strategy
```

> Step 0 only runs if `.cx/conflicts.json` exists (written by `cx conflicts detect` during a prior `git pull`). See [conflict-resolution spec](../conflict-resolution/spec.md).

The Primer's context window is disposable. It can load 10K+ tokens of specs, memory, and docs, reason about relevance, and output a tight context block. The Master never sees the noise.

After receiving context, the Master classifies the task and either dispatches a single agent directly or spawns a Supervisor-led team. See [orchestration spec](../orchestration/spec.md) for dispatch strategies.

---

## Why a dedicated Primer?

The binary can't understand intent — it doesn't know that "gas leak SMS alerts" relates to the `gas-monitoring` spec area. The Master could figure it out, but would waste tokens loading everything and filtering it down.

The Primer solves both:
- It understands natural language (it's an LLM)
- Its context is thrown away after priming, so token waste doesn't matter
- The Master starts clean with only relevant context

---

## The `cx context` command

### Step 1 — The map

Returns compact listings of available context. The Primer reads this to decide what to drill into.

```bash
cx context --mode build
cx context --mode continue --change <name>
cx context --mode plan
```

### Step 2 — Load specific content

Returns full content of a specific resource. The Primer calls these selectively based on what the map revealed.

```bash
cx context --load spec <area>         # docs/specs/<area>/spec.md
cx context --load scenarios <area>    # docs/specs/<area>/scenarios.md
cx context --load change <name>       # docs/changes/<name>/ (all files)
cx context --load architecture        # docs/architecture/index.md
cx context --load overview            # docs/overview.md
cx context --load direction           # docs/memories/DIRECTION.md
cx context --load decision <slug>     # single decision, full content
cx context --load observation <slug>  # single observation, full content
cx context --load session <slug>      # single session, full content
```

---

## What the map returns per mode

### BUILD

The developer wants to create something new. The map gives broad awareness.

```markdown
<!-- cx context | mode: build | 2026-02-23T10:00:00Z -->

## [SPEC INDEX]
Full content of docs/specs/index.md — the tree of all spec areas with descriptions.

## [ACTIVE CHANGES]
One line per active change (from docs/changes/*/proposal.md first H1 + first paragraph):
- add-ble-pairing: Add BLE pairing support for iOS and Android (angel, 3 days)
- fix-gas-threshold: Fix false positives in gas threshold alerting (carlos, 1 day)

## [DECISIONS]
All active decisions, compact (title → outcome):
- BLE pairing uses Just Works mode → no user input required during pairing
- TimescaleDB for telemetry → already running PostgreSQL, hypertable extension

## [OBSERVATIONS]
Last 7 days, compact (author, age, title):
- [angel, 2h] Fixed N+1 in /devices — JOIN for sensor readings
- [carlos, 5h] MQTT drops messages >256KB — must chunk
- [angel, 3d] Gas threshold callback has 100ms debounce

## [DIRECTION]
Full content of docs/memories/DIRECTION.md.

## [PERSONAL NOTES]
Matching personal notes from ~/.cx/memory.db:
- Zephyr k_sleep takes milliseconds not seconds
- Hono middleware: separate files per concern
```

### CONTINUE

The developer is picking up an existing change. The map gives change-focused context.

```markdown
<!-- cx context | mode: continue | change: add-ble-pairing | 2026-02-23T10:00:00Z -->

## [CHANGE SUMMARY]
From docs/changes/add-ble-pairing/:
  proposal.md: <first H1 + first paragraph>
  design.md: <first H1 + first paragraph>
  tasks.md: <Linear issue refs + completion state>
  delta specs: device-communication

## [SESSION RECOVERY]
Latest session by this author for this change:
  Goal: Implement basic BLE pairing flow on nRF52840
  Accomplished: basic pairing works, tested with nRF Connect
  Blockers: iOS CoreBluetooth not discovering service
  Next steps: Debug iOS discovery — check Background Modes
  Files touched: src/ble/service.c, src/ble/pairing.c, prj.conf

## [CHANGE MEMORY]
Observations + decisions where change = add-ble-pairing (compact):
- [obs] iOS CoreBluetooth requires Background Modes for BLE scanning
- [obs] Zephyr BLE stack requires CONFIG_BT_SMP for pairing
- [dec] BLE pairing uses Just Works mode → no user input required

## [DIRECTION]
Full content of docs/memories/DIRECTION.md.

## [PERSONAL NOTES]
Matching personal notes:
- Zephyr k_sleep takes milliseconds not seconds
```

### PLAN

The developer is doing high-level thinking. Minimal context, clean slate.

```markdown
<!-- cx context | mode: plan | 2026-02-23T10:00:00Z -->

## [OVERVIEW]
Full content of docs/overview.md.

## [ARCHITECTURE SUMMARY]
From docs/architecture/index.md: Design Principles + Tech Stack sections only.

## [PERSONAL NOTES]
Preference and working-style notes:
- I prefer Hono middleware as separate files per concern
- My preferred test structure: arrange-act-assert
```

---

## Mode classification

The Primer classifies the developer's opening message into one of three modes. Classification is pure LLM reasoning — no binary involved.

### CONTINUE

**Signal phrases**: "continue", "pick up", "where were we", "back to [feature]", "resume", references an existing change name, mentions previous work on something specific.

**Inference rule**: Developer references work that's already in progress.

**Required flag**: `--change <name>` — the Primer must identify which change from the active changes list. If the developer says "continue on BLE" and there's only one change matching BLE, use that. If ambiguous, the Primer returns a disambiguation question to the Master.

### BUILD

**Signal phrases**: "I want to add", "let's build", "new feature", "create a", "implement", describes something that doesn't exist yet.

**Inference rule**: Developer describes something new. No existing change matches.

**Default**: If the Primer can't confidently classify, default to BUILD.

### PLAN

**Signal phrases**: "let's plan", "brainstorm", "think about", "how should we approach", "what if", "architecture", "v2", "roadmap", "redesign".

**Inference rule**: Developer is thinking, not doing. High-level, open-ended.

---

## The Primer's relevance evaluation

After receiving the map from step 1, the Primer must decide what to load in step 2. This is the core value the Primer provides — it's the relevance filter the binary can't be.

### For BUILD mode

1. Read the `[SPEC INDEX]` — identify which spec areas relate to what the developer wants to build
2. Call `cx context --load spec <area>` for each relevant area (usually 1-2, rarely more than 3)
3. Read `[ACTIVE CHANGES]` — warn the Master if any active change overlaps with what they want to build (collision risk)
4. Read `[DECISIONS]` — include any decision whose outcome constrains the new work
5. Read `[OBSERVATIONS]` — include any observation the developer would hit if they didn't know about it
6. Skip anything that's clearly unrelated

### For CONTINUE mode

1. Read `[CHANGE SUMMARY]` — always included in full
2. Read `[SESSION RECOVERY]` — always included in full (this is the pickup point)
3. Read `[CHANGE MEMORY]` — include all (these are already scoped to this change)
4. If the delta touches specs, call `cx context --load spec <area>` for the canonical spec (so the agent knows the current state, not just the delta)
5. Read `[PERSONAL NOTES]` — include if relevant to the files/specs being touched

### For PLAN mode

1. `[OVERVIEW]` and `[ARCHITECTURE SUMMARY]` are already loaded in the map — include both
2. `[PERSONAL NOTES]` — include all preference notes
3. Do NOT load specs, observations, decisions, or changes — planning needs a clean slate
4. If the developer's message mentions a specific area ("let's plan v2 of the alerting system"), the Primer MAY load that one spec area as reference, but should flag it clearly as "current state for reference" not as a constraint

---

## Output format

The Primer returns a single structured markdown block to the Master. This is the contract between the Primer and the Master.

```markdown
<!-- PRIMED CONTEXT | mode: BUILD | 2026-02-23T10:00:00Z -->

## Session Mode
BUILD — new feature: gas leak SMS alerts

## Relevant Specs
### gas-monitoring (docs/specs/gas-monitoring/spec.md)
<loaded spec content, possibly trimmed to relevant sections>

## Active Decisions
- Alert thresholds stored in device config for offline operation
- TimescaleDB for sensor telemetry

## Warnings
- Active change "fix-gas-threshold" touches gas-monitoring — coordinate to avoid conflicts

## Recent Observations
- [carlos, 5h] MQTT drops messages >256KB — must chunk payloads
- [angel, 3d] Gas threshold firmware callback has 100ms debounce

## Memory Direction
<DIRECTION.md content — so agent knows what to save during this session>
```

### Output rules

- **Token budget**: The Primer should target 500-800 tokens of output. Enough to be useful, not enough to crowd the Master's context.
- **No raw dumps**: The Primer never passes through a full file verbatim unless it's short (<200 tokens). Long files get summarized or trimmed to relevant sections.
- **Warnings are first-class**: If the Primer detects a collision risk (active change on same spec area), ambiguity (multiple changes could match), or missing context (no spec exists for what the developer wants), it includes a `## Warnings` section.
- **Sections can be omitted**: If there are no relevant observations, skip `## Recent Observations`. Don't include empty sections.

---

## Error cases

### Ambiguous CONTINUE
Developer says "let's continue" but there are multiple active changes, or no recent session.
- Primer returns a disambiguation block instead of context:
```markdown
<!-- PRIMED CONTEXT | mode: CONTINUE | needs disambiguation -->

## Disambiguation Required
Multiple active changes found. Which one?
- add-ble-pairing (angel, 3 days, last session: 2h ago)
- fix-gas-threshold (carlos, 1 day, last session: 5h ago)
```
The Master presents this as a question to the developer.

### No matching spec for BUILD
Developer wants to build something that doesn't map to any existing spec area.
- This is fine — it means a new spec area will be created. The Primer notes this:
```markdown
## Warnings
- No existing spec area matches "SMS alerting". A new spec area will likely be needed.
```

### Empty project (first session)
No memories, no specs, no changes exist yet.
- The Primer detects this and returns minimal context:
```markdown
<!-- PRIMED CONTEXT | mode: BUILD | 2026-02-23T10:00:00Z -->

## Session Mode
BUILD — first session on this project

## Project Overview
<docs/overview.md content>

## Memory Direction
<DIRECTION.md content>

## Notes
No specs, memories, or changes exist yet. This is a fresh project.
```

---

## Token budget by mode

| Mode | Map (step 1) | Loaded content (step 2) | Primer output | Master receives |
|------|-------------|------------------------|---------------|-------------------|
| BUILD | ~800 tok | ~2-4K tok (1-2 specs) | 500-800 tok | 500-800 tok |
| CONTINUE | ~600 tok | ~1-2K tok (canonical spec) | 500-800 tok | 500-800 tok |
| PLAN | ~400 tok | 0 tok (map has everything) | 300-500 tok | 300-500 tok |

The Primer's own context window absorbs the map + loaded content. The Master only ever sees the distilled output.
