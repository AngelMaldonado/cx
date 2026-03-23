---
name: memory-tui-dashboard
type: masterfile
---

## Problem

The `cx` memory system accumulates valuable institutional knowledge — observations, decisions, session histories, agent run logs, and cross-project connections — but today this knowledge is only accessible through CLI commands that return flat text output. Developers must mentally assemble a picture of their project's memory landscape across multiple invocations of `cx memory search`, `cx memory list`, `cx agent-run list`, and separate commands for push/pull sync status.

Specific pain points:

1. **No ambient visibility.** A developer starting work has no quick way to see "what's been happening here?" — the recent sessions, what agents ran, what was learned, whether sync is stale.

2. **Search is the only entry point.** Without a browse mode, developers who don't know what to search for can't discover relevant memories. FTS5 requires you to have a query in mind.

3. **Relationship opacity.** Memory links (`related-to`, `caused-by`, `resolved-by`, `see-also`) are stored in the DB but invisible in CLI output. A developer cannot see which decisions caused which observations, or how a bug fix chain resolves back to a root cause.

4. **Sync state is opaque.** There is no single view of "what has been pushed, what is pending, what conflicts exist." You must remember to run `cx doctor` and interpret its output manually.

5. **Agent run history is buried.** `cx agent-run list` dumps rows. There is no way to browse agent runs grouped by session, see which runs produced which artifacts, or spot patterns like "the go-expert always blocks on database tasks."

6. **Cross-project discovery is awkward.** `cx memory search --all-projects "query"` works but requires knowing what to ask. A developer managing five projects cannot quickly orient to the cross-project memory landscape.

The `docs/overview.md` design already calls out `cx dashboard` as one of three developer entry points to the system: "cx dashboard — TUI for visibility into specs, changes, memory, team sync." This feature delivers that entry point.

---

## Context

### What exists today

- **Memory system**: Three SQLite DBs — `.cx/memory.db` (per-project), `~/.cx/index.db` (global project registry), `~/.cx/memory.db` (personal notes). Fully implemented as of the memory-architecture change.
- **Go API** (`internal/memory/`): `OpenProjectDB`, `OpenGlobalIndexDB`, `OpenPersonalDB`, `SaveMemory`, `GetMemory`, `ListMemories`, `SearchMemories`, `SearchAllProjects`, `SaveSession`, `GetLatestSession`, `SaveAgentRun`, `ListAgentRuns`, `SaveMemoryLink`, `Push`, `Pull`. Types: `Memory`, `Session`, `AgentRun`, `MemoryLink`, `ListOpts`, `SearchOpts`, `MemoryResult`, `ProjectMemoryResult`, `PushResult`.
- **CLI commands**: `cx memory save/decide/session/note/search/list/link/push/pull/forget`, `cx agent-run log/list`. All implemented.
- **WAL mode**: All DBs opened with `PRAGMA journal_mode=WAL`, which allows concurrent readers — a TUI can open the same DB file while the agent is writing to it without corruption.
- **`modernc.org/sqlite`**: Pure-Go driver in use (no cgo). Available to the TUI without any driver change.
- **`cx dashboard` placeholder**: Referenced in `docs/overview.md` and `docs/specs/skills/spec.md` ("developers never call cx commands directly except cx init and cx dashboard") as an expected command. Not yet implemented.

### Constraints

- **Read-only with two write exceptions**: The dashboard is a viewer. It MUST NOT write memories, create sessions, or modify the DB. The two exceptions are: (1) triggering `cx memory push`/`cx memory pull` as a convenience action; (2) marking a memory as deprecated (soft, non-destructive).
- **No network.** The binary is fully local. Dashboard reads only local SQLite files.
- **No separate process.** The dashboard runs as `cx dashboard` within the same Go binary. It does not spawn a server.
- **Go binary constraint.** The TUI must be implemented in Go using a library compatible with the existing build. `modernc.org/sqlite` is already vendored; no new CGO dependency can be introduced.
- **Terminal only.** No web UI, no electron, no separate binary. The TUI must work in any terminal that supports ANSI codes (iTerm2, Terminal.app, VS Code terminal, tmux).
- **`docs/overview.md` intent**: The dashboard is a "read-only view of docs/ + memory for visibility into specs, changes, memory, and team sync." This scopes the feature.

### Tech stack recommendation: Bubble Tea

[Bubble Tea](https://github.com/charmbracelet/bubbletea) by Charmbracelet is the standard Go TUI framework. It follows the Elm architecture: `Model`, `Update(msg) (Model, Cmd)`, `View() string`. Companion libraries:
- `github.com/charmbracelet/bubbles` — pre-built components: table, textinput, spinner, viewport, list, paginator
- `github.com/charmbracelet/lipgloss` — terminal styling, borders, colors, layout
- `github.com/charmbracelet/glamour` — markdown rendering (for memory content preview)

These three libraries are the idiomatic Bubble Tea stack and are well-maintained. Adding them as dependencies is the right call.

---

## Direction

Implement `cx dashboard` as a fullscreen Bubble Tea TUI with eight views navigated by tab and keyboard. The TUI opens all three SQLite databases read-only (via the existing `internal/memory` Go API), polls for updates every 5 seconds, and renders structured views for memories, sessions, agent runs, memory links, cross-project data, and sync status.

The entry point is a new `cmd/dashboard.go` cobra command that initializes the TUI application. The TUI lives in a new `internal/tui/` package. All DB reads go through the existing `internal/memory` Go API — no new SQL is written in the TUI layer.

### Design principles

1. **Browse-first**: The default view is a navigable list, not a search prompt. Search is always available via `/` but is not required to start.
2. **Context everywhere**: Every selected item shows a preview pane. You never need to exit the TUI to see full content.
3. **Progressive disclosure**: List view shows the compact extract; pressing Enter or right-arrow opens the detail view with full markdown-rendered content.
4. **Keyboard-native**: No mouse required. All navigation is via familiar vim-style keys (`j/k`, `g/G`) plus tab, enter, escape, and `/` for search.
5. **Live-ish**: Auto-refresh every 5 seconds via a `time.Ticker` Bubble Tea command. No fsnotify dependency — polling is sufficient for this use case.
6. **Graceful degradation**: If a database does not exist (no `.cx/memory.db` — project not initialized), show a helpful empty state rather than crashing.

---

## Screen Layout and Navigation

### Global key bindings (all views)

| Key | Action |
|-----|--------|
| `1-8` or `Tab`/`Shift+Tab` | Switch between views |
| `/` | Focus search bar |
| `Esc` | Clear search / close detail / return to list |
| `q` or `Ctrl+C` | Quit |
| `r` | Force refresh |
| `?` | Toggle key binding help overlay |

### View 1: Home / Overview

The landing screen. Provides an at-a-glance status of the project's memory health.

```
╔══════════════════════════════════════════════════════════════════════╗
║  cx dashboard                         project: cx  branch: main     ║
╠══════════════╦═══════════════════════╦══════════════════════════════╣
║  [1] Home    ║  [2] Memories  [3] Sessions  [4] Runs  [5] Graph     ║
║  [6] Cross   ║  [7] Sync      [8] Notes                             ║
╠══════════════╩═══════════════════════╩══════════════════════════════╣
║                                                                      ║
║  MEMORY SUMMARY                          RECENT ACTIVITY            ║
║  ─────────────────────────────           ──────────────────────     ║
║  Observations  42  (3 deprecated)        2h ago  session ended      ║
║  Decisions     18  (1 superseded)          angel / build / cx       ║
║  Sessions      31                        3h ago  go-expert blocked  ║
║  Agent Runs   124                          on database schema       ║
║  Links         17                        5h ago  decision saved     ║
║                                            use bubbletea for TUI    ║
║  SYNC STATUS                                                         ║
║  ─────────────────────────────           LATEST SESSION             ║
║  Last push    2h ago                     ──────────────────────     ║
║  Pending      0 memories                 Goal: implement TUI        ║
║  Conflicts    0                          dashboard for cx           ║
║  Last pull    just now                   Next: write Phase 1        ║
║                                          tests and wire up the      ║
║  PERSONAL NOTES                          cobra command              ║
║  ─────────────────────────────                                       ║
║  12 notes in ~/.cx/memory.db                                        ║
║  Last note: 3 days ago                                              ║
║                                                                      ║
╠══════════════════════════════════════════════════════════════════════╣
║  tab: next view  r: refresh  /: search  ?: help  q: quit            ║
╚══════════════════════════════════════════════════════════════════════╝
```

**Data sources:**
- `ListMemories(projectDB, ListOpts{})` — count by entity_type, count deprecated
- `GetLatestSession(projectDB)` — latest session summary
- `ListAgentRuns(projectDB, "")` — total count, recent runs (limit 3)
- `Push(...)` status query: `SELECT COUNT(*) FROM memories WHERE visibility='project' AND shared_at IS NULL`
- `db.QueryRow("SELECT COUNT(*) FROM memory_links")` — link count

### View 2: Memories Browser

The primary browsing surface. Two-pane layout: filterable list on the left, preview on the right.

```
╔══════════════════════════════════════════════════════════════════════╗
║  [2] Memories                          42 observations / 18 decisions ║
╠══════════════════════════╦═════════════════════════════════════════╣
║ Filter: [all▼] /search   ║  2026-03-20  angel  observation          ║
║ ─────────────────────────║  ─────────────────────────────────────  ║
║ > SQLite WAL mode allows  ║  SQLite WAL mode allows concurrent        ║
║   concurrent TUI readers  ║  readers without locking                 ║
║   bugfix · memory · 2h   ║                                          ║
║                           ║  SQLite WAL mode supports concurrent     ║
║   Use bubbletea for TUI   ║  reads with serialized writes. Opening   ║
║   decision · active       ║  the TUI's read connection while the     ║
║   architecture · 3h       ║  agent writes does not cause corruption  ║
║                           ║  because WAL mode provides snapshot      ║
║   MQTT drops >256KB       ║  isolation for readers.                  ║
║   discovery · deprecated  ║                                          ║
║   ~~crossed out~~ · 1d   ║  Tags: sqlite, wal, tui, concurrency     ║
║                           ║  Change: memory-tui-dashboard            ║
║   CoreBluetooth needs     ║  Source: planner                         ║
║   Background Modes cap.   ║  Shared: not yet pushed                  ║
║   discovery · 2d          ║                                          ║
║                           ║  Links:                                  ║
║   ...                     ║  ↳ caused-by: ble-background-modes       ║
║ ─────────────────────────║  ↳ see-also: wal-mode-decision           ║
║ 42 shown  j/k: nav        ║                                          ║
╠══════════════════════════╩═════════════════════════════════════════╣
║  enter: detail  f: filter  d: deprecate  /: search  tab: next view  ║
╚══════════════════════════════════════════════════════════════════════╝
```

**Filter panel** (activated by `f`):
- Entity type: all / observation / decision
- Observation subtype: all / bugfix / discovery / pattern / context
- Decision status: all / active / superseded / cancelled
- Author: all / [author names from DB]
- Change: all / [change IDs from DB]
- Show deprecated: toggle

**Key bindings:**
| Key | Action |
|-----|--------|
| `j/k` or `↑/↓` | Navigate list |
| `Enter` or `→` | Open full detail view |
| `←` | Return to list from detail |
| `f` | Toggle filter panel |
| `d` | Mark selected as deprecated (confirmation prompt required) |
| `g/G` | Jump to top/bottom |
| `n/N` | Next/previous search match |
| `p` | Preview toggle (hide right pane for more list space) |

**Data sources:**
- `ListMemories(projectDB, opts)` — with filter state mapped to `ListOpts`
- `SearchMemories(projectDB, query, opts)` — when search is active
- `db.Query("SELECT from_id, to_id, relation_type FROM memory_links WHERE from_id = ? OR to_id = ?", id, id)` — for link display in preview

### View 3: Sessions Timeline

Chronological view of all sessions with quick access to the key fields that matter for session continuity.

```
╔══════════════════════════════════════════════════════════════════════╗
║  [3] Sessions Timeline                              31 sessions      ║
╠══════════════════════════╦═════════════════════════════════════════╣
║  2026-03-20  build       ║  Session: 2026-03-20 / build             ║
║  > cx / 3h 12m           ║  ─────────────────────────────────────  ║
║    implement TUI layout  ║  Change: memory-tui-dashboard            ║
║                          ║  Duration: 3h 12m                        ║
║  2026-03-19  continue    ║                                          ║
║    cx / 1h 45m           ║  Goal:                                   ║
║    fix export tests      ║  Implement TUI dashboard layout and       ║
║                          ║  wire up the Bubble Tea component tree.  ║
║  2026-03-18  build       ║                                          ║
║    cx / 5h 01m           ║  Accomplished:                           ║
║    memory architecture   ║  - Scaffolded internal/tui/ package      ║
║                          ║  - Implemented home view and memories     ║
║  2026-03-17  plan        ║    browser with filter panel             ║
║    cx / 0h 47m           ║  - Added lipgloss color scheme           ║
║    tui masterfile        ║                                          ║
║                          ║  Next Steps:                             ║
║  2026-03-15  build       ║  Implement sessions timeline (View 3)    ║
║    cx / 2h 30m           ║  and agent runs view (View 4), then      ║
║    memory CRUD           ║  write integration tests.                ║
╠══════════════════════════╩═════════════════════════════════════════╣
║  enter: detail  /: search by goal  tab: next view  q: quit         ║
╚══════════════════════════════════════════════════════════════════════╝
```

**Key bindings:** Same navigation as Memories Browser. `/` searches session goals and summaries.

**Data sources:**
- `db.Query("SELECT id, mode, change_name, goal, started_at, ended_at, summary FROM sessions ORDER BY started_at DESC")` — direct query since `ListMemories` does not cover sessions
- Duration computed from `started_at` and `ended_at` (handle `ended_at = ""` as "in progress")

### View 4: Agent Runs

Session-grouped list of all agent dispatches. Surfaces what ran, how long, what status, and what was produced.

```
╔══════════════════════════════════════════════════════════════════════╗
║  [4] Agent Runs                                    124 total runs   ║
╠══════════════════════════╦═════════════════════════════════════════╣
║  Session 2026-03-20 ▼    ║  Agent Run Detail                        ║
║  ── 8 runs ──────────    ║  ─────────────────────────────────────  ║
║  > planner  success  3m  ║  Agent Type: planner                     ║
║    memory-tui masterfile ║  Session: 2026-03-20 build               ║
║                          ║  Status: success                         ║
║    scout    success  1m  ║  Duration: 3m 14s                        ║
║    explore internal/mem  ║                                          ║
║                          ║  Prompt Summary:                         ║
║    go-expert blocked 8m  ║  Write masterfile for TUI dashboard      ║
║    implement migration   ║  feature. Explore codebase areas,        ║
║                          ║  identify Bubble Tea patterns...         ║
║  Session 2026-03-19 ▶    ║                                          ║
║  ── 5 runs (collapsed)   ║  Result Summary:                         ║
║                          ║  Wrote comprehensive masterfile at        ║
║  Session 2026-03-18 ▶    ║  docs/masterfiles/memory-tui-           ║
║  ── 12 runs (collapsed)  ║  dashboard.md covering 8 views,          ║
║                          ║  3 phases, Bubble Tea architecture.      ║
║                          ║                                          ║
║                          ║  Artifacts:                              ║
║                          ║  docs/masterfiles/memory-tui-dashboard.md ║
╠══════════════════════════╩═════════════════════════════════════════╣
║  enter: detail  space: expand/collapse session  tab: next  q: quit  ║
╚══════════════════════════════════════════════════════════════════════╝
```

**Key bindings:**
| Key | Action |
|-----|--------|
| `Space` | Expand/collapse session group |
| `a` | Expand all sessions |
| `c` | Collapse all sessions |
| `/` | Filter by agent type |

**Status color coding** (via lipgloss):
- `success` → green
- `blocked` → yellow
- `needs-input` → orange

**Data sources:**
- `db.Query("SELECT id, mode, change_name, started_at FROM sessions ORDER BY started_at DESC")` — session headers
- `ListAgentRuns(projectDB, sessionID)` — runs per session

### View 5: Memory Graph

Visual representation of memory links. Given the constraints of terminal rendering, this uses a text-based adjacency format rather than a force-directed graph — more readable than ASCII art boxes.

```
╔══════════════════════════════════════════════════════════════════════╗
║  [5] Memory Graph                                  17 links total   ║
╠══════════════════════════╦═════════════════════════════════════════╣
║  Filter: [all▼]          ║  Link Detail                             ║
║  ──────────────────────  ║  ─────────────────────────────────────  ║
║  > ble-background-modes  ║  FROM:                                   ║
║    ╠═ caused-by          ║  ble-background-modes (discovery)        ║
║    ║   wal-sqlite-tui    ║  iOS CoreBluetooth requires Background   ║
║    ╚═ see-also           ║  Modes capability...                     ║
║        mqtt-256kb        ║                                          ║
║                          ║  TO:                                     ║
║    use-bubbletea (dec.)  ║  wal-sqlite-tui (observation)            ║
║    ╚═ resolved-by        ║  SQLite WAL mode allows concurrent TUI   ║
║        tui-arch-done     ║  readers without locking...              ║
║                          ║                                          ║
║    memory-architecture   ║  Relation: caused-by                     ║
║    ╠═ related-to                                                    ║
║    ║   ble-pairing-just  ║  Other links from ble-background-modes:  ║
║    ╚═ related-to         ║  ↳ see-also: mqtt-256kb                  ║
║        search-spec       ║                                          ║
║                          ║  Inbound links to ble-background-modes:  ║
║                          ║  ← use-bubbletea resolved-by this        ║
╠══════════════════════════╩═════════════════════════════════════════╣
║  enter: open memory  /: filter by relation type  tab: next  q: quit ║
╚══════════════════════════════════════════════════════════════════════╝
```

**Key bindings:**
- `Enter` on a memory ID: navigate to that memory in View 2
- `/`: filter by relation type

**Data sources:**
- `db.Query("SELECT from_id, to_id, relation_type, created_at FROM memory_links ORDER BY from_id")` — all links
- Group by `from_id` for the tree display
- `GetMemory(projectDB, id)` — for preview pane of selected node

### View 6: Cross-Project View

Federation across all registered projects in `~/.cx/index.db`.

```
╔══════════════════════════════════════════════════════════════════════╗
║  [6] Cross-Project                                  4 projects      ║
╠══════════════════════════╦═════════════════════════════════════════╣
║ /search all projects     ║  Result Detail                           ║
║ ──────────────────────   ║  ─────────────────────────────────────  ║
║  Projects:               ║  Project: firmware-ble                   ║
║  [x] cx          (here)  ║  Path: ~/dev/firmware-ble                ║
║  [x] firmware-ble        ║                                          ║
║  [x] api-service         ║  SQLite WAL mode supports concurrent     ║
║  [ ] old-project         ║  reads with serialized writers. We       ║
║                          ║  confirmed this on Zephyr + SD card      ║
║  ── Results ─────────    ║  logging scenario — opened reader from   ║
║  > SQLite WAL mode       ║  two threads simultaneously.             ║
║    observation · cx      ║                                          ║
║    2026-03-20            ║  Tags: sqlite, wal, embedded             ║
║                          ║  Author: angel                           ║
║    WAL concurrent reads  ║  Change: firmware-ble / ble-v2           ║
║    observation           ║                                          ║
║    firmware-ble          ║  Note: this memory is from a different   ║
║    2025-11-14            ║  project. Press Enter to open that       ║
║                          ║  project's dashboard in a new context.   ║
╠══════════════════════════╩═════════════════════════════════════════╣
║  /: search  space: toggle project  enter: view detail  q: quit      ║
╚══════════════════════════════════════════════════════════════════════╝
```

**Key bindings:**
- `Space` on project: toggle include/exclude from search
- `/`: run federated search across selected projects
- `Enter`: open detail for selected result

**Data sources:**
- `OpenGlobalIndexDB()` → `db.Query("SELECT id, name, path FROM projects")` — project list
- `SearchAllProjects(globalDB, query, opts)` — federated search
- Each result carries `ProjectName` and `ProjectPath` from `ProjectMemoryResult`

### View 7: Sync Status

Visibility into push/pull state and conflict warnings.

```
╔══════════════════════════════════════════════════════════════════════╗
║  [7] Sync Status                                                     ║
╠══════════════════════════════════════════════════════════════════════╣
║                                                                      ║
║  PUSH STATUS                                                         ║
║  ─────────────────────────────────────────────────────────────────  ║
║  Last push:   2026-03-20 14:05:00 UTC (2 hours ago)                 ║
║  Pending:     3 memories not yet pushed                             ║
║                                                                      ║
║  Pending memories:                                                   ║
║  > SQLite WAL mode allows concurrent readers    observation  2h ago  ║
║    Use bubbletea for TUI dashboard              decision     3h ago  ║
║    TUI dashboard masterfile planning session    observation  3h ago  ║
║                                                                      ║
║  [Push now]  Press 'p' to run cx memory push                        ║
║                                                                      ║
║  PULL / CONFLICTS                                                    ║
║  ─────────────────────────────────────────────────────────────────  ║
║  Last pull:  just now (git post-merge hook triggered cx index rebuild)║
║  Conflicts:  0                                                       ║
║  Status:     clean                                                   ║
║                                                                      ║
║  DOCTOR CHECK                                                        ║
║  ─────────────────────────────────────────────────────────────────  ║
║  DB health:  ok (schema version 1)                                  ║
║  FTS index:  ok                                                      ║
║  Missing files: 0                                                    ║
║                                                                      ║
╠══════════════════════════════════════════════════════════════════════╣
║  p: push now  u: pull now  r: refresh  tab: next view  q: quit      ║
╚══════════════════════════════════════════════════════════════════════╝
```

**Write actions (only two in the dashboard):**
- `p`: run `cx memory push` as an `exec.Command`. Show spinner while running, display result.
- `u`: run `cx memory pull` as an `exec.Command`. Show spinner, display result.

These are the only two write operations. Everything else is read-only.

**Data sources:**
- `db.Query("SELECT COUNT(*), MAX(shared_at) FROM memories WHERE visibility='project' AND shared_at IS NULL AND entity_type != 'session'")` — pending push count + last push time
- `db.Query("SELECT id, title, entity_type, created_at FROM memories WHERE visibility='project' AND shared_at IS NULL AND entity_type != 'session' ORDER BY created_at DESC")` — pending list
- Conflict detection: query for same-ID rows where local content differs from markdown file (re-use doctor check logic)

### View 8: Personal Notes

Browse personal notes from `~/.cx/memory.db`.

```
╔══════════════════════════════════════════════════════════════════════╗
║  [8] Personal Notes                        12 notes  ~/.cx/memory.db ║
╠══════════════════════════╦═════════════════════════════════════════╣
║  /search notes           ║  Note Detail                             ║
║  ──────────────────────  ║  ─────────────────────────────────────  ║
║  > prefer Hono middleware  ║  Type: preference                       ║
║    preference            ║  Topic Key: hono-middleware-pattern      ║
║    Updated 3 days ago    ║                                          ║
║                          ║  I prefer Hono middleware structured as  ║
║    Zephyr k_sleep takes  ║  separate files per concern — one file   ║
║    milliseconds not secs ║  per middleware function. Avoids mega-   ║
║    reminder              ║  files and makes testing easier.         ║
║    Updated 1 week ago    ║                                          ║
║                          ║  Projects: [api-service, firmware-ble]   ║
║    BLE debug on macOS    ║  Tags: hono, middleware, structure       ║
║    use PacketLogger      ║  Created: 2026-01-15                     ║
║    tool_tip              ║  Updated: 2026-03-17                     ║
║    Updated 2 weeks ago   ║                                          ║
║                          ║  Note: personal notes are local-only.    ║
║    ...                   ║  They are never committed or pushed.     ║
╠══════════════════════════╩═════════════════════════════════════════╣
║  enter: detail  /: search  tab: next view  q: quit                  ║
╚══════════════════════════════════════════════════════════════════════╝
```

**Data sources:**
- `OpenPersonalDB()` → `db.Query("SELECT id, type, title, content, topic_key, projects, tags, created_at, updated_at FROM personal_notes ORDER BY updated_at DESC")`
- Note: personal notes use a different table schema (`personal_notes`, not `memories`)

---

## Architecture and Integration

### Entry point

`cmd/dashboard.go` — a new cobra command registered with the root command:

```go
// cmd/dashboard.go
package cmd

import (
    "github.com/anthropics/cx/internal/tui"
    "github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
    Use:   "dashboard",
    Short: "Open TUI dashboard for memory, sessions, and sync",
    Long:  "Read-only TUI dashboard providing visibility into specs, changes, memory, and team sync status.",
    RunE: func(cmd *cobra.Command, args []string) error {
        projectPath, err := findProjectRoot()
        if err != nil {
            return err
        }
        return tui.Run(projectPath)
    },
}

func init() {
    rootCmd.AddCommand(dashboardCmd)
}
```

### Package structure

```
internal/tui/
├── app.go          ← top-level App model, tab routing, global key handling
├── home.go         ← View 1: overview stats and recent activity
├── memories.go     ← View 2: memories browser with filter/search
├── sessions.go     ← View 3: sessions timeline
├── runs.go         ← View 4: agent runs grouped by session
├── graph.go        ← View 5: memory link graph
├── crossproject.go ← View 6: cross-project federated search
├── sync.go         ← View 7: push/pull sync status
├── notes.go        ← View 8: personal notes browser
├── detail.go       ← shared full-detail view (rendered markdown)
├── styles.go       ← lipgloss color scheme, shared styles
├── components/
│   ├── table.go    ← reusable list/table component (wraps bubbles/list)
│   ├── preview.go  ← right-pane preview with markdown rendering
│   ├── search.go   ← search bar (wraps bubbles/textinput)
│   ├── filter.go   ← filter panel popup
│   ├── statusbar.go← bottom status bar
│   └── spinner.go  ← loading spinner for async operations
└── data/
    ├── loader.go   ← DB connections + data loading, cached
    └── poller.go   ← 5-second tick-based refresh
```

### App model (Bubble Tea top level)

```go
// internal/tui/app.go
type View int

const (
    ViewHome View = iota
    ViewMemories
    ViewSessions
    ViewRuns
    ViewGraph
    ViewCrossProject
    ViewSync
    ViewNotes
)

type AppModel struct {
    activeView   View
    width        int
    height       int
    projectPath  string
    projectDB    *sql.DB
    globalDB     *sql.DB
    personalDB   *sql.DB
    home         HomeModel
    memories     MemoriesModel
    sessions     SessionsModel
    runs         RunsModel
    graph        GraphModel
    crossProject CrossProjectModel
    sync         SyncModel
    notes        NotesModel
    statusBar    components.StatusBar
    helpOverlay  bool
    lastRefresh  time.Time
    loading      bool
    err          error
}

type TickMsg time.Time
type RefreshMsg struct{}
type DataLoadedMsg struct { /* loaded data */ }
type SyncResultMsg struct {
    Action  string // push | pull
    Result  string
    Success bool
}

func (m AppModel) Init() tea.Cmd {
    return tea.Batch(
        tickCmd(),          // start 5-second refresh tick
        loadAllDataCmd(m),  // initial data load
    )
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        // propagate size to all sub-models
    case tea.KeyMsg:
        // handle global keys: tab, 1-8, q, ?, r
        // delegate view-specific keys to active sub-model
    case TickMsg:
        return m, tea.Batch(tickCmd(), loadAllDataCmd(m))
    case DataLoadedMsg:
        // update sub-models with fresh data
    case SyncResultMsg:
        // update sync view with result
    }
    // delegate to active sub-model's Update
}
```

### DB connection strategy

The dashboard opens all three databases once at startup and holds them open for the session:

```go
// internal/tui/data/loader.go
type Loader struct {
    projectDB  *sql.DB
    globalDB   *sql.DB
    personalDB *sql.DB
}

func NewLoader(projectPath string) (*Loader, error) {
    pdb, err := memory.OpenProjectDB(projectPath)
    if err != nil {
        // project not initialized — return nil pdb, not an error
        // dashboard shows empty state for project-specific views
    }
    gdb, err := memory.OpenGlobalIndexDB()
    // personalDB always opens (personal notes)
    perdb, err := memory.OpenPersonalDB()
    return &Loader{projectDB: pdb, globalDB: gdb, personalDB: perdb}, nil
}
```

All three DBs are opened with WAL mode (handled by the existing `openDB` function). WAL mode allows concurrent reads while the agent is writing — the TUI will never block the agent and the agent will never corrupt the TUI's reads.

The loader opens connections in read-only spirit but does not use `PRAGMA query_only=ON` — this is deferred because push/pull write actions need a writable connection for View 7 (they shell out via `exec.Command` instead, which opens its own connection).

### Polling strategy

```go
// internal/tui/data/poller.go
func tickCmd() tea.Cmd {
    return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
        return TickMsg(t)
    })
}

func loadAllDataCmd(loader *Loader) tea.Cmd {
    return func() tea.Msg {
        // Run all queries in a goroutine
        // Return DataLoadedMsg with all loaded data
        data, err := loader.LoadAll()
        if err != nil {
            return errMsg(err)
        }
        return DataLoadedMsg{Data: data}
    }
}
```

Polling every 5 seconds is sufficient. The TUI is a passive observer — there is no requirement for sub-second freshness.

### Push/pull write actions

Push and pull are the only write actions. They shell out via `exec.Command` rather than calling the Go API directly, because:
1. The Go API writes to the DB and filesystem — side effects that should follow the same validation path as CLI usage
2. Shelling out reuses existing output formatting and error handling

```go
// internal/tui/sync.go (inside Update for 'p' key)
func runPushCmd() tea.Cmd {
    return func() tea.Msg {
        out, err := exec.Command("cx", "memory", "push").CombinedOutput()
        return SyncResultMsg{
            Action:  "push",
            Result:  string(out),
            Success: err == nil,
        }
    }
}
```

### Deprecation action (View 2 only)

The one in-place write action available in the Memories Browser is deprecating a memory. This is done by shelling out to a new `cx memory deprecate <id>` command (which needs to be added as part of this feature — see Files to Modify). The deprecation action requires a confirmation dialog in the TUI before proceeding.

---

## Data Model Mapping

Each view maps to specific Go API calls:

| View | Primary API | Secondary API |
|------|------------|---------------|
| Home | `ListMemories(projectDB, ListOpts{Limit:0})` count query | `GetLatestSession(projectDB)`, `ListAgentRuns(projectDB, "")` with limit 3 |
| Memories | `ListMemories(projectDB, opts)` for browse | `SearchMemories(projectDB, q, opts)` for search, direct link query for graph preview |
| Sessions | Direct `db.Query` on `sessions` table | `ListAgentRuns(projectDB, sessionID)` for run count per session |
| Agent Runs | `db.Query` sessions for grouping | `ListAgentRuns(projectDB, sessionID)` per session |
| Graph | Direct `db.Query` on `memory_links` | `GetMemory(projectDB, id)` for node details |
| Cross-Project | `db.Query` on `index.db` projects table | `SearchAllProjects(globalDB, q, opts)` |
| Sync | Direct `db.Query` for pending push count | Doctor checks (schema validation) |
| Notes | Direct `db.Query` on `personal_notes` in personal DB | — |

**Note on sessions table**: `GetLatestSession` returns only one row. For the Sessions Timeline view, a direct `db.Query` is needed to list all sessions. Consider adding `ListSessions(db *sql.DB, limit int) ([]Session, error)` to `internal/memory/entities.go` as part of this feature.

**Note on personal notes**: Personal notes use a different table (`personal_notes`) in `~/.cx/memory.db`. The existing `personal_notes` schema is in `internal/memory/migrations.go`. A `ListPersonalNotes(db *sql.DB, query string, limit int) ([]PersonalNote, error)` function needs to be added to `internal/memory/entities.go`. Define a `PersonalNote` struct mirroring the table.

---

## Component Architecture

### Shared components (`internal/tui/components/`)

**Table** (`table.go`): Wraps `github.com/charmbracelet/bubbles/list`. Provides:
- Configurable columns
- Row selection with keyboard nav
- Scrollable viewport for large lists
- Optional row coloring by status/type

**Preview** (`preview.go`): Right-pane preview panel. Uses:
- `github.com/charmbracelet/glamour` for markdown rendering
- `github.com/charmbracelet/bubbles/viewport` for scrollable content
- `lipgloss.Border(lipgloss.RoundedBorder())` for visual separation

**SearchBar** (`search.go`): Wraps `github.com/charmbracelet/bubbles/textinput`. Activates on `/`, sends search text upstream via message.

**FilterPanel** (`filter.go`): Popup panel (rendered as a floating overlay via lipgloss positioning). Contains multiple filter controls as a form.

**StatusBar** (`statusbar.go`): Bottom bar showing: current view name, last refresh time, key binding hints, error messages. Fixed-height 1-2 lines.

**Spinner** (`spinner.go`): Wraps `github.com/charmbracelet/bubbles/spinner`. Used during data load and push/pull operations.

### Message types

```go
// Global messages
type TickMsg time.Time
type WindowSizeMsg struct{ Width, Height int }
type DataLoadedMsg struct{ Data *LoadedData }
type ErrMsg struct{ Err error }
type SyncResultMsg struct{ Action, Result string; Success bool }

// View-navigation messages
type NavigateToMemoryMsg struct{ ID string }  // from graph view → memories view
type SwitchViewMsg struct{ View View }

// Search/filter messages
type SearchChangedMsg struct{ Query string }
type FilterChangedMsg struct{ Opts ListOpts }

// Action confirmation messages
type ConfirmDeprecateMsg struct{ ID, Title string }
type DeprecateConfirmedMsg struct{ ID string }
type DeprecateCancelledMsg struct{}
```

### Layout

Use `lipgloss` for layout. The two-pane layout uses a horizontal join:

```go
// Conceptual layout for two-pane views
leftPane  := lipgloss.NewStyle().Width(leftWidth).Height(contentHeight)
rightPane := lipgloss.NewStyle().Width(rightWidth).Height(contentHeight)
content   := lipgloss.JoinHorizontal(lipgloss.Top, leftPane.Render(list), rightPane.Render(preview))
```

Widths are calculated from terminal width: left pane = 40%, right pane = 60%. On narrow terminals (<100 columns), collapse to single pane with toggle.

---

## Visual Design

### Color scheme (lipgloss)

```go
// internal/tui/styles.go
var (
    // Neutrals
    ColorText       = lipgloss.Color("15")   // bright white
    ColorSubtle     = lipgloss.Color("240")  // gray
    ColorBorder     = lipgloss.Color("238")  // dark gray

    // Accents
    ColorAccent     = lipgloss.Color("86")   // cyan-green (CX brand)
    ColorSelected   = lipgloss.Color("33")   // blue
    ColorSuccess    = lipgloss.Color("82")   // green
    ColorWarning    = lipgloss.Color("214")  // orange
    ColorError      = lipgloss.Color("196")  // red
    ColorDeprecated = lipgloss.Color("241")  // dimmed

    // Entity type colors
    ColorObservation = lipgloss.Color("75")   // blue
    ColorDecision    = lipgloss.Color("141")  // purple
    ColorSession     = lipgloss.Color("114")  // green
    ColorAgentRun    = lipgloss.Color("180")  // yellow
)
```

### Lipgloss styles

```go
var (
    StyleTitle = lipgloss.NewStyle().
        Foreground(ColorAccent).
        Bold(true)

    StyleSelected = lipgloss.NewStyle().
        Background(ColorSelected).
        Foreground(ColorText)

    StyleDeprecated = lipgloss.NewStyle().
        Foreground(ColorDeprecated).
        Strikethrough(true)

    StyleBorder = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(ColorBorder)

    StyleHeaderBar = lipgloss.NewStyle().
        Background(lipgloss.Color("235")).
        Foreground(ColorText).
        Padding(0, 1)

    StyleStatusBar = lipgloss.NewStyle().
        Background(lipgloss.Color("236")).
        Foreground(ColorSubtle)

    StyleTag = lipgloss.NewStyle().
        Background(lipgloss.Color("237")).
        Foreground(ColorAccent).
        Padding(0, 1)
)
```

### Terminal size adaptation

| Width | Layout mode |
|-------|-------------|
| < 80 cols | Single pane (no preview); paginator for navigation |
| 80-120 cols | Two pane 40/60 split |
| > 120 cols | Two pane 35/65 split (more preview space) |

Height adaptation: minimum 24 rows required. Below 24 rows, show a "terminal too small" message.

---

## Technical Considerations

### Performance

- **Lazy loading per view**: Data is only loaded for the active view + home stats. Switching to a new view triggers a load if data is stale (> 5 seconds old).
- **Pagination**: List views load a maximum of 200 rows by default. The user can load more via `Ctrl+D` ("load more"). Paginating via `ListOpts.Limit` + offset.
- **No goroutine-per-row**: All data fetching happens in a single `tea.Cmd` goroutine per refresh cycle, returning a single `DataLoadedMsg`.
- **Cached counts**: Home view stats (memory counts, session count, etc.) are computed once per refresh, not on every render.

### Database locking

WAL mode (`PRAGMA journal_mode=WAL`) is already set by `OpenProjectDB`. WAL allows unlimited concurrent readers — the TUI can safely read while `cx memory save` writes to the same DB. No additional locking is needed.

The TUI should not write to the DB directly. Push/pull are handled via `exec.Command("cx", ...)` to avoid write contention.

### `exec.Command` for push/pull

`exec.Command` for push/pull requires that `cx` is in PATH. This is a reasonable assumption since the user invoked `cx dashboard`. If `cx` is not found, show an error in the Sync view.

### Responsive layout

The app subscribes to `tea.WindowSizeMsg`. On receipt, it recalculates all widths/heights and re-renders. Bubble Tea delivers `WindowSizeMsg` on startup and on terminal resize.

### Error handling

- DB not found (project not initialized): show "Project not initialized. Run cx init." empty state per view.
- DB open error: show error in status bar, continue with degraded functionality.
- Refresh error: show stale indicator in status bar ("last updated 2m ago — refresh failed").
- Render continues with last-good data on error.

### Accessibility

- All information is text-based — no color-only indicators. Status badges also include text labels.
- Deprecated items are both strikethrough AND dimmed AND labeled "[deprecated]".
- Key binding help overlay (`?`) lists all bindings in plain text.

---

## Implementation Phases

### Phase 1: Foundation + Home + Memories Browser (MVP)

**Goal**: Runnable `cx dashboard` with the two highest-value views.

**Tasks:**
1. Add Bubble Tea, bubbles, lipgloss, glamour as Go module dependencies
2. Scaffold `internal/tui/` package structure with all planned files (empty stubs)
3. Implement `internal/tui/data/loader.go` — DB connections and initial data load
4. Implement `internal/tui/data/poller.go` — 5-second tick refresh
5. Implement `internal/tui/app.go` — top-level App model, tab navigation, window size handling
6. Implement `internal/tui/styles.go` — full color scheme and lipgloss styles
7. Implement `internal/tui/components/statusbar.go`
8. Implement `internal/tui/components/spinner.go`
9. Implement `internal/tui/home.go` — stats summary, recent activity, sync status summary, latest session
10. Implement `internal/tui/components/table.go` — reusable list component
11. Implement `internal/tui/components/preview.go` — preview pane with glamour markdown
12. Implement `internal/tui/components/search.go` — search bar
13. Implement `internal/tui/components/filter.go` — filter panel popup
14. Implement `internal/tui/memories.go` — full memories browser with search and filter
15. Implement `cmd/dashboard.go` — cobra command registration
16. Add `ListSessions(db *sql.DB, limit int) ([]Session, error)` to `internal/memory/entities.go`
17. Add `PersonalNote` struct and `ListPersonalNotes` to `internal/memory/entities.go`
18. Write unit tests for all data loader functions
19. Write integration tests for home and memories views against test SQLite DB

**Estimated complexity**: High (framework + two views). ~18-20 tasks.

### Phase 2: Sessions, Agent Runs, Detail Views, Personal Notes (Core)

**Goal**: Complete the personal-project-scoped views. All views except cross-project and graph.

**Tasks:**
1. Implement `internal/tui/sessions.go` — sessions timeline with accordion expand
2. Implement `internal/tui/runs.go` — agent runs grouped by session
3. Implement `internal/tui/detail.go` — shared full-screen detail view with glamour rendering
4. Implement `internal/tui/sync.go` — push/pull status panel with exec.Command integration
5. Implement `internal/tui/notes.go` — personal notes browser
6. Add `cx memory deprecate <id>` CLI command to `cmd/memory.go`
7. Implement deprecation confirmation dialog in memories view
8. Wire up cross-view navigation (graph → memories → detail)
9. Implement narrow-terminal single-pane layout mode
10. Write unit tests for sessions, runs, sync, notes views
11. Manual testing across: iTerm2, VS Code terminal, tmux

**Estimated complexity**: Medium (patterns established). ~10-12 tasks.

### Phase 3: Memory Graph, Cross-Project, Polish (Complete)

**Goal**: Cross-project federation view, memory link graph, and polish pass.

**Tasks:**
1. Implement `internal/tui/graph.go` — memory link adjacency display
2. Implement `internal/tui/crossproject.go` — federated search across all projects
3. Implement help overlay (`?` key) listing all key bindings
4. Add keyboard shortcut customization support (optional, low priority)
5. Performance audit: measure render time on 1000+ memories; add pagination if needed
6. Implement "load more" pagination in memories and sessions views
7. Add `cx dashboard` to skill files that reference direct developer commands (skills/spec.md catalog update)
8. Update `docs/overview.md` to reflect `cx dashboard` as implemented (not planned)
9. Write end-to-end test: open dashboard against populated test DB, verify view rendering

**Estimated complexity**: Medium. ~8-10 tasks.

---

## Files to Modify

### New files

| File | Purpose |
|------|---------|
| `cmd/dashboard.go` | Cobra command entry point |
| `internal/tui/app.go` | Top-level Bubble Tea app model |
| `internal/tui/home.go` | View 1: home/overview |
| `internal/tui/memories.go` | View 2: memories browser |
| `internal/tui/sessions.go` | View 3: sessions timeline |
| `internal/tui/runs.go` | View 4: agent runs |
| `internal/tui/graph.go` | View 5: memory graph |
| `internal/tui/crossproject.go` | View 6: cross-project |
| `internal/tui/sync.go` | View 7: sync status |
| `internal/tui/notes.go` | View 8: personal notes |
| `internal/tui/detail.go` | Shared full-detail view |
| `internal/tui/styles.go` | Color scheme and lipgloss styles |
| `internal/tui/components/table.go` | Reusable list/table component |
| `internal/tui/components/preview.go` | Preview pane component |
| `internal/tui/components/search.go` | Search bar component |
| `internal/tui/components/filter.go` | Filter panel component |
| `internal/tui/components/statusbar.go` | Status bar component |
| `internal/tui/components/spinner.go` | Loading spinner |
| `internal/tui/data/loader.go` | DB connections and data loading |
| `internal/tui/data/poller.go` | 5-second tick refresh |

### Modified files

| File | Change |
|------|--------|
| `cmd/root.go` | Register `dashboardCmd` |
| `internal/memory/entities.go` | Add `ListSessions`, `PersonalNote` struct, `ListPersonalNotes` |
| `cmd/memory.go` | Add `cx memory deprecate <id>` subcommand |
| `go.mod` / `go.sum` | Add bubbletea, bubbles, lipgloss, glamour dependencies |
| `docs/specs/skills/spec.md` | Note `cx dashboard` as a direct developer command (already referenced but mark as implemented) |
| `docs/overview.md` | Update `cx dashboard` description from planned to implemented |

---

## Affected Specs

| Spec | Change |
|------|--------|
| `skills` | Update catalog entry to confirm `cx dashboard` is implemented; add to list of direct developer commands |
| `memory` | Add `cx memory deprecate <id>` command to command reference table |
| `doctor` | Add check: warn if `cx dashboard` dependencies (bubbletea) are missing from go.mod (for contributors) |

No structural spec changes are needed. The dashboard is a new consumer of the existing memory API — it does not change the memory data model, storage, or sync behavior.

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Bubble Tea adds significant binary size | Low | Low | bubbletea + lipgloss + glamour add ~2MB to binary; acceptable for a dev tool |
| `modernc.org/sqlite` WAL concurrent read issues on Windows | Low | Medium | Test on Windows; WAL mode on Windows uses shared memory, should work; document if issues arise |
| Terminal compatibility (older terminals without 256-color) | Medium | Low | Use only 256-color palette (not true-color); degrade gracefully to 8-color on `$TERM=xterm` |
| `exec.Command("cx", ...)` for push/pull fails if cx not in PATH | Low | Low | Check cx binary path at startup; if not found, disable push/pull buttons with explanation |
| Data load is slow for large memory sets (1000+ rows) | Low | Medium | Apply `LIMIT 200` default; paginate; show row count so user knows truncation occurred |
| Personal notes table schema differs from memories schema | Known | Low | Handle separately with `PersonalNote` struct; do not attempt to unify schemas |
| Deprecated memory strikethrough not supported in all terminals | Medium | Low | Strikethrough is widely supported; fall back to `[deprecated]` text label |

---

## Testing

### Unit tests

- `internal/tui/data/loader_test.go`: Test each data loading function against an in-memory SQLite DB seeded with test data. Verify correct row counts, field mapping, filter behavior.
- `internal/memory/entities_test.go`: Add tests for `ListSessions` and `ListPersonalNotes`.

### Integration tests

- `internal/tui/app_test.go`: Use `bubbletea/v2`'s `Program.Send(msg)` and assertion helpers to simulate key presses and verify view rendering. Test: tab navigation switches view, `/` activates search, `q` quits.
- Test against populated SQLite DBs with seeded memories, sessions, agent runs, and links.

### Manual testing checklist

- [ ] `cx dashboard` starts without error on fresh initialized project
- [ ] `cx dashboard` shows empty state on uninitialized project (no `.cx/memory.db`)
- [ ] All 8 views render without panic
- [ ] Tab and `1-8` keys navigate between views
- [ ] Memories browser search filters correctly
- [ ] Memories browser filter panel applies all filters
- [ ] Session timeline shows sessions in reverse-chronological order
- [ ] Agent runs are grouped by session, collapsible
- [ ] Sync view shows correct pending count
- [ ] Push (`p`) calls `cx memory push` and shows result
- [ ] Cross-project view shows all registered projects
- [ ] Personal notes loads from `~/.cx/memory.db`
- [ ] Terminal resize is handled (no layout breakage)
- [ ] `q` and `Ctrl+C` exit cleanly (no zombie processes)
- [ ] Works in iTerm2, VS Code terminal, tmux

---

## Open Questions

1. **`cx memory deprecate <id>` command**: Should this be a new top-level `cx memory deprecate` subcommand, or a flag on an existing command (`cx memory forget --soft <id>`)? Recommend new subcommand for clarity.

2. **Load-more vs infinite scroll**: When a view has >200 memories, should `Ctrl+D` load another 200 rows (append to list), or use a paginator that replaces the current page? Recommend append (feel like infinite scroll) since users typically want to browse forward, not page back.

3. **Personal notes write access**: Should the dashboard allow creating or updating personal notes from within the TUI? Current direction: no — keep the dashboard read-only (plus push/pull). Personal notes can be managed via `cx memory note` in the CLI.

4. **Cross-project project path display**: On the cross-project view, should full absolute paths be shown or just project directory names? Recommend directory name as primary, full path in detail/tooltip on hover or `i` key.

5. **Export from TUI**: Should there be an action to copy a memory's content to clipboard (for pasting into a chat or doc)? This is a quality-of-life addition. Defer to Phase 3 using `atotto/clipboard` package.

6. **Offline personal notes in cross-project**: Should personal notes from `~/.cx/memory.db` appear in the Cross-Project view? They span all projects. Current direction: no — personal notes only appear in View 8. Cross-project is for project-scoped memories only.

7. **Navigation from graph to memory detail**: When the user presses Enter on a memory ID in the Graph view, should it: (a) switch to View 2 with that memory selected, or (b) open the detail view inline in the Graph view? Recommend (a) — switch to Memories view with the item highlighted — for consistency.

---

## References

- `docs/overview.md` — `cx dashboard` first mentioned at lines 52, 58
- `docs/specs/skills/spec.md` — "developers never call cx commands directly except cx init and cx dashboard"
- `docs/specs/memory/spec.md` — full memory data model, all entity types and fields
- `docs/archive/2026-03-20-memory-architecture/design.md` — DB topology, schema, Go API
- `internal/memory/` — existing Go API (the TUI's data layer)
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — Elm-architecture TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) — Bubble Tea component library
- [Lipgloss](https://github.com/charmbracelet/lipgloss) — terminal styling
- [Glamour](https://github.com/charmbracelet/glamour) — markdown rendering in terminal
