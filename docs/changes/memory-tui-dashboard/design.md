---
name: memory-tui-dashboard
type: design
---

## Architecture

The dashboard runs entirely within the `cx` binary as a Bubble Tea fullscreen TUI. There is no server, no background process, and no network I/O. The data flow is:

```
cx dashboard (cmd/dashboard.go)
    └── tui.Run(projectPath)         (internal/tui/app.go)
            └── data.NewLoader()     (internal/tui/data/loader.go)
                    ├── memory.OpenProjectDB(projectPath)   → .cx/memory.db
                    ├── memory.OpenGlobalIndexDB()          → ~/.cx/index.db
                    └── memory.OpenPersonalDB()             → ~/.cx/memory.db

AppModel (tea.Model)
    ├── data.Loader → all DB reads via internal/memory API
    ├── 5-second tick → DataLoadedMsg → refresh all active tab data
    ├── views map[Tab]View — 8 sub-models implementing the View interface, only the active one receives key events
    ├── detail *DetailModel — shared full-screen glamour overlay
    └── push/pull write actions → exec.Command("cx", "memory", "push"|"pull")
```

All DB reads go through the existing `internal/memory` Go API — no new SQL is written in the TUI layer except for two cases where the API does not have a list function (sessions list, memory links list). These two are direct `db.Query` calls inside `internal/tui/data/loader.go`.

## Technical Decisions

**Bubble Tea + bubbles + lipgloss + glamour stack.** Bubble Tea (Charmbracelet) is the idiomatic Go TUI framework. It follows the Elm architecture (`Model`, `Update(msg) (Model, Cmd)`, `View() string`). The companion libraries provide pre-built components (`bubbles/list`, `bubbles/viewport`, `bubbles/textinput`, `bubbles/spinner`) and terminal styling (`lipgloss`) and markdown rendering (`glamour`). These four libraries become new `go.mod` dependencies.

**Polling, not fsnotify.** The TUI polls all three DBs every 5 seconds via `tea.Tick`. This avoids a new OS dependency (`fsnotify`), is sufficient for the passive-observer use case, and keeps the implementation simpler. Sub-second freshness is not required.

**Direct DB reads via `internal/memory` API, not CLI shelling.** For reads, the TUI calls Go functions directly. This is faster, avoids subprocess overhead on every refresh, and keeps the data loading in-process. CLI shelling is only used for write operations (push/pull).

**Push/pull via `exec.Command`, not Go API.** The two write operations (`cx memory push`, `cx memory pull`) shell out to the `cx` binary. Rationale: the write path follows existing validation, formatting, and error handling. Shelling out reuses this rather than duplicating it. The TUI itself holds no write lock on the DB.

**WAL concurrent reads.** All three DBs are opened with `PRAGMA journal_mode=WAL` (handled by the existing `memory.OpenProjectDB`, `memory.OpenGlobalIndexDB`, `memory.OpenPersonalDB` functions). WAL mode allows unlimited concurrent readers. The TUI will never block agent writes, and agent writes will never corrupt TUI reads.

**Responsive layout: 40/60 two-pane split, collapsing to single-pane below 80 columns.** Left pane (list) = 40% terminal width; right pane (preview) = 60%. Below 80 columns: single pane, user toggles preview with `p`. Below 60 columns × 15 rows: "terminal too small" message. The app subscribes to `tea.WindowSizeMsg` and recalculates on every resize.

**`ListSessions` and `ListPersonalNotes` added to `internal/memory/entities.go`.** The existing API has `GetLatestSession` (one row) and no personal note list function. Two new functions are needed: `ListSessions(db *sql.DB, limit int) ([]Session, error)` and `ListPersonalNotes(db *sql.DB, query string, limit int) ([]PersonalNote, error)`. A `PersonalNote` struct mirrors the `personal_notes` table schema.

**`cx memory deprecate <id>` as a new CLI command.** The one in-place write from the Memories Browser is deprecation. This is implemented as a new `cx memory deprecate <id>` cobra subcommand in `cmd/memory.go`. The TUI shells out to it with a confirmation dialog before proceeding.

**Default pagination: 200 rows per view.** Data loads apply `LIMIT 200` to prevent slow renders on large memory sets. Users load more with `Ctrl+D`. Row counts in the view header tell users when truncation has occurred.

## Implementation Notes

### Package structure

```
cmd/
└── dashboard.go            ← cobra command: cx dashboard

internal/
├── memory/
│   └── entities.go         ← MODIFIED: add ListSessions, PersonalNote, ListPersonalNotes
└── tui/
    ├── app.go              ← top-level AppModel, tab routing, global key handling
    ├── home.go             ← View 1: overview stats and recent activity
    ├── memories.go         ← View 2: memories browser with filter/search/deprecate
    ├── sessions.go         ← View 3: sessions timeline
    ├── runs.go             ← View 4: agent runs grouped by session
    ├── sync.go             ← View 5: push/pull status + exec.Command integration
    ├── notes.go            ← View 6: personal notes browser
    ├── graph.go            ← View 7: memory link adjacency display
    ├── crossproject.go     ← View 8: cross-project federated search
    ├── detail.go           ← shared full-screen detail view (glamour-rendered)
    ├── styles.go           ← lipgloss color scheme and shared styles
    ├── components/
    │   ├── table.go        ← reusable list (wraps bubbles/list)
    │   ├── preview.go      ← right-pane preview (glamour + bubbles/viewport)
    │   ├── search.go       ← search bar (wraps bubbles/textinput)
    │   ├── filter.go       ← filter panel popup (lipgloss overlay)
    │   ├── statusbar.go    ← bottom status bar (fixed 1-2 lines)
    │   └── spinner.go      ← loading spinner (wraps bubbles/spinner)
    └── data/
        ├── loader.go       ← DB connections, data loading functions, caching
        └── poller.go       ← 5-second tick Cmd via tea.Tick
```

### Top-level App model

```go
// internal/tui/app.go

// Tab identifies which tab is active.
type Tab int

const (
    TabHome        Tab = iota  // tab 1
    TabMemories                // tab 2
    TabSessions                // tab 3
    TabRuns                    // tab 4
    TabSync                    // tab 5
    TabNotes                   // tab 6
    TabGraph                   // tab 7
    TabCrossProject            // tab 8
)

// View is implemented by every tab sub-model.
type View interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (View, tea.Cmd)
    View() string
    SetSize(width, height int)
    SetData(data *data.LoadedData)
    IsInputFocused() bool
}

// CursorPositioner is an optional interface that views can implement to
// expose their cursor position for the status bar position indicator.
type CursorPositioner interface {
    CursorPosition() (cursor, total int)
}

// statusBar holds the rendered state for the bottom status bar.
// It is a local struct (not components.StatusBar) to avoid an import cycle.
type statusBar struct {
    left    string
    right   string
    hints   string // pre-rendered key hints
    width   int
    isStale bool
}

type AppModel struct {
    activeTab   Tab
    views       map[Tab]View
    width, height int
    loader      *data.Loader
    data        *data.LoadedData
    detail      *DetailModel
    spin        spinner.Model
    spinActive  bool
    bar         statusBar
    helpOverlay bool
    lastRefresh time.Time
    loading     bool
    quitting    bool
    err         error
}

type TickMsg time.Time
type DataLoadedMsg struct {
    Data *data.LoadedData
    Err  error
}
type SyncResultMsg struct{ Action, Result string; Success bool }
type NavigateToMemoryMsg struct{ ID string }
type ConfirmDeprecateMsg struct{ ID, Title string }
```

`Init()` returns `tea.Batch(m.loadData(), data.PollCmd(data.DefaultPollInterval), m.spin.Tick)` — initial data load, the 5-second poll timer, and the spinner tick. `Update()` handles global keys (`tab`, `shift+tab`, `h/l/←/→`, `1-8`, `q`, `?`, `r`) and delegates view-specific keys to the active sub-model. Global keys are suppressed when `IsInputFocused()` returns true on the active view.

### Data loader

```go
// internal/tui/data/loader.go
type Loader struct {
    projectDB  *sql.DB  // nil if project not initialized
    globalDB   *sql.DB
    personalDB *sql.DB
}

func NewLoader(projectPath string) (*Loader, error)
// Opens all three DBs. projectDB may be nil (graceful empty state).

type LoadedData struct {
    // Home stats
    MemoryCounts   map[string]int  // by entity_type
    DeprecatedCount int
    LatestSession  *memory.Session
    RecentRuns     []memory.AgentRun
    LinkCount      int
    PendingPush    int
    LastPushAt     string
    PersonalCount  int

    // Per-view data (loaded lazily per active view)
    Memories   []memory.Memory
    Sessions   []memory.Session
    Runs       map[string][]memory.AgentRun  // keyed by session_id
    Links      []memory.MemoryLink
    Notes      []memory.PersonalNote
}
```

### DB topology and which APIs are used per view

| Tab | Primary API | Secondary / Direct SQL |
|-----|-------------|------------------------|
| Home (Tab 1) | `memory.ListMemories(projectDB, ListOpts{})` count | `memory.GetLatestSession(projectDB)`, `memory.ListAgentRuns(projectDB, "")` limit 3, `SELECT COUNT(*) FROM memories WHERE ... AND shared_at IS NULL`, `SELECT COUNT(*) FROM memory_links` |
| Memories (Tab 2) | `memory.ListMemories(projectDB, opts)` / `memory.SearchMemories(projectDB, q, opts)` | `SELECT from_id, to_id, relation_type FROM memory_links WHERE from_id=? OR to_id=?` |
| Sessions (Tab 3) | `memory.ListSessions(projectDB, limit)` (new) | — |
| Runs (Tab 4) | `memory.ListAgentRuns(projectDB, sessionID)` per session | `SELECT id, mode, change_id, started_at FROM sessions ORDER BY started_at DESC` |
| Sync (Tab 5) | — | `SELECT COUNT(*), MAX(shared_at) FROM memories WHERE visibility='project' AND shared_at IS NULL`, `SELECT id, title, entity_type, created_at FROM memories WHERE ... AND shared_at IS NULL ORDER BY created_at DESC` |
| Notes (Tab 6) | `memory.ListPersonalNotes(personalDB, q, limit)` (new) | — |
| Graph (Tab 7) | `memory.GetMemory(projectDB, id)` for preview | `SELECT from_id, to_id, relation_type, created_at FROM memory_links ORDER BY from_id` |
| Cross-Project (Tab 8) | `memory.SearchAllProjects(globalDB, q, opts)` | `SELECT id, name, path FROM projects` on `~/.cx/index.db` |

### New API functions in `internal/memory/entities.go`

```go
// ListSessions returns sessions ordered by started_at DESC.
func ListSessions(db *sql.DB, limit int) ([]Session, error)

// PersonalNote mirrors the personal_notes table.
type PersonalNote struct {
    ID        int
    Type      string   // pattern | preference | tool_tip | reminder
    Title     string
    Content   string
    TopicKey  string
    Projects  []string // JSON array
    Tags      []string // comma-separated
    CreatedAt string
    UpdatedAt string
}

// ListPersonalNotes returns personal notes ordered by updated_at DESC.
// If query is non-empty, filters via FTS5 on personal_notes_fts.
func ListPersonalNotes(db *sql.DB, query string, limit int) ([]PersonalNote, error)
```

### New CLI command in `cmd/memory.go`

```go
// cx memory deprecate <id>
// Marks a memory as deprecated in .cx/memory.db (sets deprecated=1).
// Requires user confirmation prompt. Prints memory title before prompting.
// Does NOT write to the docs/memory/ markdown files (deprecation is DB-level).
var memoryDeprecateCmd = &cobra.Command{
    Use:   "deprecate <id>",
    Short: "Mark a memory as deprecated",
    Args:  cobra.ExactArgs(1),
    RunE:  runMemoryDeprecate,
}
```

### Push/pull shell-out pattern

```go
// internal/tui/sync.go
// cxPath is resolved once at construction via exec.LookPath("cx") and stored on SyncModel.
// Subsequent push/pull calls use the stored path to avoid repeated PATH lookups.
func (m *SyncModel) runSyncCmd(action string, args ...string) tea.Cmd {
    cxPath := m.cxPath
    fullArgs := append([]string{"memory", action}, args...)
    return func() tea.Msg {
        out, err := exec.Command(cxPath, fullArgs...).CombinedOutput()
        return syncOpResultMsg{action: action, output: string(out), success: err == nil}
    }
}
```

The TUI checks at startup that `cx` is in PATH. If not found, the push/pull buttons in Tab 5 (Sync) are disabled with an explanatory message.

### Lipgloss color scheme (internal/tui/styles.go)

```go
ColorAccent     = lipgloss.Color("86")   // cyan-green
ColorSelected   = lipgloss.Color("33")   // blue
ColorSuccess    = lipgloss.Color("82")   // green
ColorWarning    = lipgloss.Color("214")  // orange
ColorError      = lipgloss.Color("196")  // red
ColorDeprecated = lipgloss.Color("241")  // dimmed

ColorObservation = lipgloss.Color("75")  // blue
ColorDecision    = lipgloss.Color("141") // purple
ColorSession     = lipgloss.Color("114") // green
ColorAgentRun    = lipgloss.Color("180") // yellow
```

All colors use the 256-color palette (not true-color) for broad terminal compatibility. On `$TERM=xterm` (8-color), lipgloss degrades gracefully.

### Error handling strategy

- `projectDB` nil (not initialized): each project-scoped view shows "Project not initialized. Run cx init." empty state.
- DB open error: status bar shows error, affected views show empty state.
- Refresh error: stale indicator appears in status bar ("last updated 2m ago — refresh failed"); last-good data remains rendered.
- Deprecated items rendered: strikethrough + dimmed + `[deprecated]` text label (all three for accessibility; strikethrough is not color-only).

### Implementation phases

**Phase 1 (MVP):** Foundation, Home view, Memories Browser. Bubble Tea dependencies, `internal/tui/` scaffold, `data/loader.go`, `data/poller.go`, `app.go`, `styles.go`, all `components/`, `home.go`, `memories.go`, `cmd/dashboard.go`, new `ListSessions`/`ListPersonalNotes` API additions, unit tests.

**Phase 2 (Core):** Sessions timeline, Agent runs, Detail view, Sync view, Personal notes, `cx memory deprecate` command, deprecation dialog, cross-view navigation, narrow-terminal layout.

**Phase 3 (Complete):** Memory graph (View 7), Cross-project view (View 8), help overlay, pagination ("load more"), performance audit, spec/docs updates for `cx dashboard` as implemented.
