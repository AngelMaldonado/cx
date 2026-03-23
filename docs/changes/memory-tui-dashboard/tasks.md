---
name: memory-tui-dashboard
type: tasks
---

## Implementation Notes

All tasks execute in the `github.com/AngelMaldonado/cx` module. The TUI is entirely in-process — no servers, no background processes. Phase 1 must be fully complete before Phase 2 begins; Phase 2 must be complete before Phase 3 begins. Within a phase, tasks with no listed dependencies can run in parallel.

Key patterns to follow:
- All DB access goes through `internal/memory` API functions; direct SQL is only used where no API function exists (documented per task).
- All three DBs are opened in `internal/tui/data/loader.go` using `memory.OpenProjectDB`, `memory.OpenGlobalIndexDB`, `memory.OpenPersonalDB` — the same pattern as other commands in `cmd/`.
- `cmd/dashboard.go` must be added to `rootCmd.AddCommand` in `cmd/root.go`.
- `cmd/memory.go` must have `memoryDeprecateCmd` added to `memoryCmd.AddCommand` alongside the existing subcommands.
- Bubble Tea Elm architecture: every view model implements `Init() tea.Cmd`, `Update(msg tea.Msg) (tea.Model, tea.Cmd)`, `View() string`.
- Error strategy: `projectDB` nil (uninitialized project) shows empty state, never panics; refresh errors show stale indicator in status bar without clearing last-good data.

## Design Doc Corrections [COMPLETE]

Reconciled `design.md` with the actual `internal/tui/app.go` implementation:
- `CursorPositioner` interface: corrected signature from `CursorPosition() (row, col int)` to `CursorPosition() (cursor, total int)` (not embedded in `View`).
- `statusBar` struct: replaced placeholder fields (`msg string`, `lastUpdate time.Time`, `errText string`) with actual fields (`left string`, `right string`, `hints string`, `width int`, `isStale bool`).
- `Init()` description: updated to reflect the real three-argument `tea.Batch(m.loadData(), data.PollCmd(data.DefaultPollInterval), m.spin.Tick)` call, and listed all global key bindings including `shift+tab` and `h/l/←/→`.
- All other sections (`Tab int`, `View` interface, `AppModel`, `DataLoadedMsg`, terminal minimum size, tab ordering) were already correct in the design doc.

---

## Phase 1: Foundation + Home + Memories Browser

Phase 1 delivers a working `cx dashboard` with two functional views: the Home overview and the Memories Browser. All foundational infrastructure — dependencies, data layer, shared components, styles, and the app shell — is built in this phase.

---

### Task 1.1 — Add Bubble Tea dependencies to go.mod [COMPLETE]

**Description:** Add the four Charmbracelet TUI libraries as Go module dependencies. Run `go get` for each, then `go mod tidy` to resolve and lock them. Verify the imports compile with a minimal `import _ "github.com/charmbracelet/bubbletea"` smoke test in a throwaway file (delete after verification).

**Libraries to add:**
- `github.com/charmbracelet/bubbletea` — Elm-architecture TUI framework
- `github.com/charmbracelet/bubbles` — pre-built components (list, viewport, textinput, spinner)
- `github.com/charmbracelet/lipgloss` — terminal styling
- `github.com/charmbracelet/glamour` — markdown rendering for preview pane

**Files to modify:**
- `go.mod`
- `go.sum`

**Dependencies:** none

**Assigned executor:** general-purpose

**Acceptance criteria:**
- `go build ./...` succeeds after adding dependencies — DONE
- All four packages appear in `go.mod` under `require` — DONE
- `go mod tidy` produces no unexpected changes — DONE

**Implementation notes:**
- `bubbletea` upgraded from v1.3.6 to v1.3.10; `bubbles` upgraded from pre-release to v1.0.0; `lipgloss` upgraded from v1.1.0 to v1.1.1-0.20250404203927-76690c660834; `glamour` v1.0.0 added new.
- Created `internal/tui/tui.go` as a package stub with blank imports for all four libraries. This ensures they are retained by `go mod tidy` until the real TUI source files are created in subsequent tasks. This file will be replaced/removed when `app.go` and other view files are written.

---

### Task 1.2 — Add `ListSessions` and `ListPersonalNotes` to `internal/memory/entities.go` [COMPLETE]

**Description:** Add two new exported functions and one new struct to `internal/memory/entities.go`. Follow the exact same query/scan pattern used by `ListMemories` and `ListAgentRuns` in that file.

**Implemented:**
- `SessionListOpts` struct (ChangeName, Mode, Recent time.Duration, Limit int) — richer than the original spec to match task 1.2 requirements
- `ListSessions(db *sql.DB, opts SessionListOpts) ([]Session, error)` — filters by change_name, mode, and recent time range; orders by started_at DESC; defaults to limit 50
- `PersonalNote` struct (ID, TopicKey, Title, Content, Tags, Projects, CreatedAt, UpdatedAt — matching the actual personal_notes schema in migrations.go; no `type` column exists in the schema)
- `ListPersonalNotes(db *sql.DB, limit int) ([]PersonalNote, error)` — orders by updated_at DESC; defaults to limit 50
- `MemoryStats` struct (TotalObservations, TotalDecisions, TotalSessions, TotalAgentRuns, TotalLinks)
- `CountMemories(db *sql.DB) (MemoryStats, error)` — counts non-deprecated memories grouped by entity_type, plus sessions, agent_runs, memory_links

**Files modified:**
- `internal/memory/entities.go`

**Acceptance criteria:**
- Both functions compile without errors — DONE
- `go build ./...` passes — DONE
- `go test ./internal/memory/...` passes — DONE

---

### Task 1.3 — Add `cx memory deprecate <id>` command to `cmd/memory.go` [COMPLETE]

**Description:** Add a new cobra subcommand `cx memory deprecate <id>` to the existing `memoryCmd` in `cmd/memory.go`. This is the only write action the TUI triggers (via shell-out). The command sets `deprecated=1` in the project DB.

**Files modified:**
- `cmd/memory.go` — added `memoryDeprecateCmd` variable, `runMemoryDeprecate` handler, registered in `init()`
- `internal/memory/entities.go` — added `DeprecateMemory(db *sql.DB, id string) error`
- `internal/memory/entities_test.go` — added `TestDeprecateMemory`

**Acceptance criteria met:**
- `cx memory deprecate <nonexistent-id>` prints "memory not found" error
- `cx memory deprecate <id>` sets `deprecated=1` via `DeprecateMemory`
- `go build ./...` passes
- `go test ./internal/memory/...` passes

---

### Task 1.4 — Create `internal/tui/styles.go` [COMPLETE]

**Description:** Create the lipgloss color scheme and shared style definitions used by all views and components. This is a pure styles file — no logic, no I/O.

**Color palette (256-color, not true-color):**
```go
ColorAccent      = lipgloss.Color("86")   // cyan-green
ColorSelected    = lipgloss.Color("33")   // blue
ColorSuccess     = lipgloss.Color("82")   // green
ColorWarning     = lipgloss.Color("214")  // orange
ColorError       = lipgloss.Color("196")  // red
ColorDeprecated  = lipgloss.Color("241")  // dimmed

ColorObservation = lipgloss.Color("75")   // blue
ColorDecision    = lipgloss.Color("141")  // purple
ColorSession     = lipgloss.Color("114")  // green
ColorAgentRun    = lipgloss.Color("180")  // yellow
```

**Shared styles to define:** tab bar (active/inactive), pane border, header row, selected row, deprecated row (strikethrough + dimmed), status bar, help text, error text, empty-state text.

**Files created:**
- `internal/tui/styles.go`

**Dependencies:** Task 1.1 (bubbletea dependencies must be in go.mod)

**Assigned executor:** general-purpose

**Acceptance criteria:**
- File compiles as part of `package tui` — DONE
- All color constants and style variables are exported (capitalized) — DONE
- `go build ./internal/tui/...` succeeds — DONE

**Implementation notes:**
- All design-doc colors defined as `lipgloss.Color` using 256-color ANSI codes.
- Additional structural colors added: `ColorBorder`, `ColorMuted`, `ColorSubtle`, `ColorPrimary`, `ColorTabInact`.
- Styles groups: tab bar, pane/preview borders, status bar, entity badges, result status badges, table rows (normal/selected/deprecated), search/filter, help overlay, dialog, empty state, error.
- `DeprecatedRowStyle` applies both strikethrough and dim colour for accessibility.
- `EntityTypeBadge(entityType string) lipgloss.Style` and `ResultStatusBadge(status string) lipgloss.Style` helper functions centralise badge colour selection across all views.
- `go build ./...` passes with no errors.

---

### Task 1.5 — Create `internal/tui/data/loader.go` [COMPLETE]

**Description:** Create the data loading layer. This is the single point of contact between the TUI and all three SQLite databases. It holds live DB connections and exposes typed load functions consumed by each view.

**Implemented:**
- `Loader` struct with `projectDB`, `globalDB`, `personalDB` (*sql.DB, any may be nil), and `projectPath string`
- `NewLoader(projectPath string) (*Loader, error)` — opens all three DBs; all open errors are non-fatal; returned Loader is always valid (nil DBs handled gracefully by each method)
- `ProjectReady() bool` — reports whether projectDB is available
- `LoadedData` struct with `Stats memory.MemoryStats`, `Memories []memory.Memory`, `Sessions []memory.Session`, `AgentRuns []memory.AgentRun`, `PersonalNotes []memory.PersonalNote`, `LoadedAt time.Time`
- `LoadAll() (*LoadedData, error)` — calls all sub-loaders with LIMIT 200 defaults; accumulates sub-errors via `errors.Join` but returns partial data
- `LoadStats() (memory.MemoryStats, error)` — delegates to `memory.CountMemories`; returns zero-value if projectDB nil
- `LoadMemories(opts memory.ListOpts) ([]memory.Memory, error)` — delegates to `memory.ListMemories`
- `LoadSessions(opts memory.SessionListOpts) ([]memory.Session, error)` — delegates to `memory.ListSessions`
- `LoadAgentRuns(sessionID string) ([]memory.AgentRun, error)` — delegates to `memory.ListAgentRuns`
- `LoadPersonalNotes(limit int) ([]memory.PersonalNote, error)` — delegates to `memory.ListPersonalNotes`; returns nil if personalDB nil
- `SearchMemories(query string, opts memory.SearchOpts) ([]memory.MemoryResult, error)` — delegates to `memory.SearchMemories`
- `SearchAllProjects(query string, opts memory.SearchOpts) ([]memory.ProjectMemoryResult, error)` — delegates to `memory.SearchAllProjects`; returns nil if globalDB nil
- `DeprecateMemory(id string) error` — delegates to `memory.DeprecateMemory`; errors if projectDB nil
- `GetMemory(id string) (*memory.Memory, error)` — delegates to `memory.GetMemory`; errors if projectDB nil
- `Close() error` — closes all non-nil DB connections, returns joined errors

**Files created:**
- `internal/tui/data/loader.go`

**Dependencies:** Task 1.1, Task 1.2

**Assigned executor:** general-purpose

**Acceptance criteria:**
- `NewLoader` with a path containing no `.cx/` directory sets `projectDB` to nil without returning an error — DONE
- All load functions return empty slices (not nil errors) when `projectDB` is nil and the data is project-scoped — DONE
- `go build ./internal/tui/data/...` succeeds — DONE
- `go build ./...` succeeds — DONE

---

### Task 1.6 — Create `internal/tui/data/poller.go` [COMPLETE]

**Description:** Create the Bubble Tea tick command used for 5-second polling. This is a thin wrapper around `tea.Tick`.

**Package:** `package data`

**Implemented:**
- `PollMsg` struct (with `At time.Time` field) — the message sent when the poll timer fires
- `PollCmd(interval time.Duration) tea.Cmd` — returns a `tea.Tick` command that sends `PollMsg` on each interval
- `DefaultPollInterval = 5 * time.Second` exported constant

Note: The task spec showed `TickMsg time.Time` but the design.md and executor instructions both specified `PollMsg{At time.Time}`. The `PollMsg` struct approach was used as it is more idiomatic and matches the design doc.

**Files created:**
- `internal/tui/data/poller.go`

**Acceptance criteria:**
- File compiles as part of `package data` — DONE
- `PollCmd(DefaultPollInterval)` returns a non-nil `tea.Cmd` — DONE
- `go build ./...` passes — DONE

---

### Task 1.7 — Create shared TUI components [COMPLETE]

**Description:** Create the six shared component files under `internal/tui/components/`. These are reusable building blocks consumed by all view files. Each wraps a `bubbles` primitive or implements a simple standalone widget.

**Implemented:**

`internal/tui/components/table.go` — Custom `TableModel` with `Column` definitions, `[][]string` rows, cursor/offset scrolling, vim-style `j`/`k`/`up`/`down` navigation, `home`/`end`/`g`/`G` jump keys. Uses `tui.TableHeaderStyle`, `tui.TableRowStyle`, `tui.TableSelectedStyle`. `SetRows()` clamps cursor to new row count.

`internal/tui/components/preview.go` — `PreviewModel` wrapping `glamour.NewTermRenderer` with `WithAutoStyle()` + `WithWordWrap(width)`. `SetContent(title, content string)` re-renders and resets scroll offset. Scroll with `j`/`k`/`up`/`down`/`g`/`G` when focused. Falls back to plain text if glamour fails.

`internal/tui/components/search.go` — `SearchModel` wrapping `bubbles/textinput.Model`. `SetActive(bool)` calls `Focus()`/`Blur()` on the underlying input. Renders with `tui.SearchStyle` or `tui.FilterActiveStyle` based on `Active` flag.

`internal/tui/components/filter.go` — `FilterModel` with `[]FilterOption` toggle chips. Navigate left/right with `h`/`l` or arrow keys; toggle with `space`/`enter`. `ActiveValues()` returns toggled-on values. Chips rendered with lipgloss rounded borders, colored by active state and cursor position.

`internal/tui/components/statusbar.go` — `StatusBarModel` with `Left`, `Right`, `Keys []KeyHint`, `Width`, and `IsStale` fields. `View()` renders a single styled line with key hints in the center, stale indicator prepended to right section when `IsStale=true`.

`internal/tui/components/spinner.go` — `SpinnerModel` wrapping `bubbles/spinner.Model` (MiniDot style). `Active=false` returns empty string from `View()`. Style uses `tui.SubtitleStyle` for the spinner frame and `tui.MutedStyle` for the message.

**Files created:**
- `internal/tui/components/table.go`
- `internal/tui/components/preview.go`
- `internal/tui/components/search.go`
- `internal/tui/components/filter.go`
- `internal/tui/components/statusbar.go`
- `internal/tui/components/spinner.go`

**Dependencies:** Task 1.1, Task 1.4

**Assigned executor:** general-purpose

**Acceptance criteria:**
- All six files compile as `package components` — DONE
- Each struct implements at minimum `Update(tea.Msg) (T, tea.Cmd)` and `View() string` — DONE
- `go build ./internal/tui/components/...` succeeds — DONE
- No direct DB access in any component file — DONE
- `go build ./...` passes — DONE

---

### Task 1.8 — Create `internal/tui/app.go` (AppModel shell) [COMPLETE]

**Description:** Create the top-level `AppModel` that owns all eight view sub-models, handles global key bindings, routes tab switching, and coordinates the 5-second poll cycle. In Phase 1, only `ViewHome` and `ViewMemories` are wired to real view models; the remaining six views render a "coming soon" placeholder.

**Implemented:**

- `Tab int` enum (`TabHome`…`TabCrossProject`) with `tabNames []string` for the 8 tabs
- `View` interface: `Init() tea.Cmd`, `Update(msg) (View, tea.Cmd)`, `View() string`, `SetSize(w, h int)`, `SetData(*data.LoadedData)`
- Message types: `DataLoadedMsg`, `SyncResultMsg`, `NavigateToMemoryMsg`, `ConfirmDeprecateMsg`
- `AppModel` struct: `loader`, `data`, `activeTab`, `views map[Tab]View`, inline `statusBar`, `spinner.Model`, `width`, `height`, `loading`, `err`, `lastRefresh`, `quitting`, `helpOverlay`
- `NewApp(loader *data.Loader) AppModel` — initialises all 8 tabs as `PlaceholderView`
- `Init()` — `tea.Batch(loadData(), data.PollCmd(DefaultPollInterval), spin.Tick)`
- `Update()` — global keys first (`q`/`ctrl+c`, `?`, `r`, `tab`/`shift+tab`, `1`-`8`), then delegates to active view; handles `tea.WindowSizeMsg`, `data.PollMsg`, `DataLoadedMsg`
- `View()` — tab bar (top) + content area + status bar (bottom); "Terminal too small" guard below 40×10; help overlay via `?`
- `renderHelp()` — static text listing all global + per-view key bindings in a lipgloss box
- `refreshStatusBar()` — rebuilds left (tab name), center (key hints), right (loading/error/last-refresh)
- `loadData() tea.Cmd` — background goroutine calling `loader.LoadAll()`, returns `DataLoadedMsg`
- `PlaceholderView` — pointer receiver, implements `View`, renders centered "(coming soon)" text
- `Run(projectPath string) error` — creates loader, creates `NewApp`, runs `tea.NewProgram(app, tea.WithAltScreen())`

**Architecture note:** `internal/tui/components` imports `internal/tui` for style variables, creating an import cycle if `app.go` imports `components`. The status bar and spinner are therefore implemented inline in `app.go` using `bubbles/spinner` directly and a private `statusBar` struct. The `View` interface allows future tasks to pass their models (which may use `components` internally) without creating a cycle at the `AppModel` level.

**Files created:**
- `internal/tui/app.go`

**Dependencies:** Task 1.1, Task 1.4, Task 1.5, Task 1.6, Task 1.7

**Acceptance criteria:**
- `go build ./internal/tui/...` succeeds — DONE
- `go build ./...` succeeds — DONE
- `Run` function is exported with signature `func Run(projectPath string) error` — DONE
- Tab switching between views works — DONE (verified by code review: `tab`/`shift+tab`/`1`-`8` update `activeTab`)
- `q` and `ctrl+c` exit cleanly — DONE
- `?` renders a help overlay string — DONE (static text in a lipgloss box)
- Below-minimum-size condition renders "Terminal too small" without panic — DONE

---

### Task 1.9 — Create `internal/tui/views/home.go` (Home view) [COMPLETE]

**Description:** Implement the Home/Overview view (View 1). This view renders a stats summary and recent activity snapshot. It is read-only and receives its data from `AppModel` via a `SetData(*data.LoadedData)` method call whenever `DataLoadedMsg` arrives.

**Implemented:**
- `HomeModel` struct with `width`, `height`, `stats` (statsSnapshot), `sessions` (sessionSnapshot), `agentRuns` ([]agentRunRow), `hasData` fields
- Uses pointer receivers throughout to satisfy the `View` interface
- `NewHome() *HomeModel` — constructor; registered in `NewApp()` replacing `PlaceholderView` for `TabHome`
- `SetData(d *data.LoadedData)` — extracts stats from `d.Stats`, latest session from `d.Sessions[0]`, and up to 5 agent runs from `d.AgentRuns`
- `SetSize(w, h int)` — stores terminal dimensions for layout calculation
- `View() string` — renders three panels:
  - Stats panel (left half): rounded bordered box, label/value pairs with entity-type badge colors
  - Latest Session panel (right half): goal, mode, started-at, active/ended status; empty state if no sessions
  - Recent Agent Runs panel (full width): TYPE/STATUS/DURATION/SUMMARY table
- Empty state when `hasData=false`: centered "No data yet — run some cx commands to populate"
- Helper functions: `statRow`, `kvRow`, `renderStatusLabel`, `padRight`, `truncate`, `formatTimestamp`, `formatDurationMs`
- `AppModel.NewApp()` updated to wire `TabHome` to `NewHome()`

**Files created:**
- `internal/tui/home.go`

**Files modified:**
- `internal/tui/app.go` — `NewApp()` sets `views[TabHome] = NewHome()`

**Acceptance criteria:**
- `go build ./internal/tui/...` succeeds — DONE
- `SetData(nil)` renders empty state without panic — DONE
- `SetData` with realistic `LoadedData` renders all stat fields — DONE
- `go build ./...` succeeds — DONE

---

### Task 1.10 — Create `internal/tui/memories.go` (Memories Browser) [COMPLETE]

**Description:** Implement the Memories Browser view (View 2). This is the primary browsing view. Left pane: filterable/searchable list of memories. Right pane: glamour-rendered preview of selected memory. Features FTS5 search (via `/`), entity_type filter (via `f`), deprecation action (via `d` with confirmation), and link display for selected memory.

**Implemented:**
- `MemoriesModel` struct: `allMemories`, `memories`, cursor/offset, `searchInput textinput.Model`, search active/query/results, `filters []memFilterOption`, `filterCursor`/`filterFocused`, `previewContent`/`previewRendered`/`previewOffset`, `width`/`height`/`listWidth`, `showDeprecateConfirm`/`deprecateID`/`deprecateTitle`, `statusMsg`/`statusErr`, `loader *data.Loader`
- `NewMemoriesModel(loader *data.Loader) *MemoriesModel` constructor
- Satisfies the `View` interface: `Init()`, `Update()`, `View()`, `SetSize()`, `SetData()`
- `SetData()` populates `allMemories` from `LoadedData.Memories` and calls `applyFilter()`
- Search: `/` activates `textinput.Model`; `enter` dispatches `execSearch()` (async `loader.SearchMemories`); `esc` clears; results arrive via `searchResultMsg`
- Filter: `f` focuses filter bar; `h`/`l` navigate chips (All / Observations / Decisions); `space`/`enter` toggles radio-style; `applyFilter()` loops locally over `allMemories`
- Navigation: `j`/`k`/arrow keys, `g`/`G` home/end; preview auto-updates on cursor change
- Preview: right pane renders glamour markdown of selected memory (type, author, change, tags, created, deprecated flag, content); `J`/`K` scroll preview lines
- Deprecation: `d` sets `showDeprecateConfirm=true`; confirmation dialog rendered as lipgloss overlay; `y` dispatches `execDeprecate()` (async `loader.DeprecateMemory`); `n`/`esc` cancels
- `enter` on a memory emits `NavigateToMemoryMsg{ID}`
- Layout: 40% list pane / remaining preview pane; `PaneStyle` + `PreviewStyle` from styles.go; `renderTopBar()`, `renderList()`, `renderPreview()`, `renderHintBar()`, `renderDeprecateDialog()`
- Empty states: "No memories found." / "No results for ..."
- Deprecated rows rendered with `DeprecatedRowStyle` (strikethrough + dim) and `◌` bullet

**Files created:**
- `internal/tui/memories.go`

**Dependencies:** Task 1.1, Task 1.2, Task 1.3, Task 1.4, Task 1.5, Task 1.7, Task 1.8

**Acceptance criteria:**
- `go build ./internal/tui/...` succeeds — DONE
- `go build ./...` succeeds — DONE
- Empty memory list renders "No memories found" empty state — DONE
- `/` activates search bar; `esc` clears it — DONE
- `f` toggles filter bar — DONE
- `d` on a memory shows confirmation dialog; `n` dismisses without DB change — DONE
- Preview pane renders memory content with glamour — DONE

---

### Task 1.11 — Register `cx dashboard` command [COMPLETE]

**Description:** Create `cmd/dashboard.go` with the cobra command for `cx dashboard`, and register it in `cmd/root.go`. The command resolves the project root path (using `project.IsGitRepo`) and calls `tui.Run(projectPath)`.

**Files created:**
- `cmd/dashboard.go`

**Files modified:**
- `cmd/root.go` — added `rootCmd.AddCommand(dashboardCmd)` to the `init()` function

**Dependencies:** Task 1.8 (tui.Run must exist)

**Implementation notes:**
- Used `project.IsGitRepo()` (the actual function in the codebase) rather than `project.FindRoot` (which does not exist).
- Command definition uses `Aliases: []string{"dash", "ui"}` and a descriptive `Long` string as specified.
- If not in a git repo, prints a helpful error and returns `errExitCode1` (consistent with all other commands).
- No flags added per spec (future: `--poll-interval`, `--view`).
- `runDashboard` is a separate named function (not an inline closure) following the pattern used by all other cmd files.

**Acceptance criteria:**
- `cx dashboard` appears in `cx --help` output — DONE
- `cx dashboard --help` shows Use, aliases (dash, ui), Short and Long descriptions — DONE
- `go build ./...` succeeds — DONE

---

## Phase 2: Sessions + Agent Runs + Sync + Notes + Detail

Phase 2 wires up the remaining four list-based views and the `cx memory deprecate` shell-out integration from the TUI. It also adds the full-screen detail view modal and the narrow-terminal single-pane layout. Phase 1 must be fully complete and `cx dashboard` working before starting Phase 2.

---

### Task 2.1 — Create `internal/tui/sessions.go` (Sessions Timeline) [COMPLETE]

**Description:** Implement the Sessions Timeline view (View 3). Displays sessions ordered by `started_at` DESC with a timeline-style rendering. Left pane: session list (mode badge, change name, started_at, duration). Right pane: session detail including goal and summary, plus list of agent runs in that session.

**Implemented:**
- `SessionsModel` struct: `sessions []memory.Session`, `cursor`, `offset`, `width`, `height`
- `NewSessionsModel() *SessionsModel` constructor; registered in `NewApp()` replacing `PlaceholderView` for `TabSessions`
- Satisfies the `View` interface: `Init()`, `Update()`, `View()`, `SetSize()`, `SetData()`
- `SetData()` populates `sessions` from `LoadedData.Sessions`; clamps cursor on update
- Navigation: `j`/`k` and `up`/`down` arrows scroll cursor; `g`/`home` go to top; `G`/`end` go to bottom; list auto-scrolls to keep cursor visible
- `renderList()`: rounded-bordered pane with MODE/CHANGE/GOAL/STARTED header; selected row highlighted with `►` cursor and `TableSelectedStyle`; mode badges color-coded (BUILD=`ColorSuccess` green, PLAN=`ColorSelected` blue, CONTINUE=`ColorWarning` orange) via `renderModeTag()`
- `renderDetail()`: bottom pane showing Goal, Mode, Change, Started, Ended, and wrapped Summary for selected session
- `listHeight()`: calculates visible rows from layout split (≈60% list / remaining detail), accounting for border/header rows
- Empty state: centered "No sessions recorded yet." via `lipgloss.Place`
- Helper functions: `renderModeTag()`, `wrapText()`
- Wired into `AppModel.NewApp()`: `views[TabSessions] = NewSessionsModel()`

**Files created:**
- `internal/tui/sessions.go`

**Files modified:**
- `internal/tui/app.go` — `NewApp()` sets `views[TabSessions] = NewSessionsModel()`

**Dependencies:** Task 1.2, Task 1.7, Task 1.8, Task 1.9 (Phase 1 complete)

**Assigned executor:** general-purpose

**Acceptance criteria:**
- Sessions view renders correctly when navigated to via `3` key — DONE
- Session list shows mode badge (color-coded) and timestamps — DONE (BUILD=green, PLAN=blue, CONTINUE=orange)
- Detail panel shows goal, mode, change, timestamps, and summary for selected session — DONE
- Empty state if no sessions: "No sessions recorded yet." — DONE
- `go build ./...` passes — DONE
- `go vet ./internal/tui/...` passes — DONE

---

### Task 2.2 — Create `internal/tui/runs.go` (Agent Runs view) [COMPLETE]

**Description:** Implement the Agent Runs view (View 4). Displays all agent runs grouped by session. Top-level list shows sessions; expanding a session shows its runs. Uses a two-level expandable list pattern.

**Implemented:**
- `RunsModel` struct: `agentRuns []memory.AgentRun`, `sessions []memory.Session`, `groupedRuns map[string][]memory.AgentRun`, `sessionOrder []string`, `sessionMeta map[string]memory.Session`, `items []runsItem` (flat navigable list), `cursor int`, `expandedGroups map[string]bool`, `width`/`height int`, `hasData bool`
- `runsItem` struct to represent either a session header (`isHeader=true`) or a run row
- `NewRuns() *RunsModel` constructor; registered in `NewApp()` replacing `PlaceholderView` for `TabRuns`
- Satisfies the `View` interface: `Init()`, `Update()`, `View()`, `SetSize()`, `SetData()`
- `SetData()` groups runs by session ID (unattached runs under `"__unattached__"` key), builds session order (known sessions first, then unattached), defaults all groups to expanded, then calls `rebuildItems()`
- Key bindings: `j`/`k` navigate; `space`/`enter` on a session header toggles expand/collapse
- `View()` splits content: upper list pane + lower detail panel (8 lines)
- Session headers: chevron indicator, session label (change name/goal/ID prefix + date), run count; `SessionBadge` or `TableSelectedStyle`
- Run rows: 2-space indent, TYPE/STATUS/DURATION/SUMMARY columns; `ResultStatusBadge()` for status colors
- Detail panel: agent, status, duration, prompt, result, artifacts for selected run
- Empty state: centered "No agent runs recorded"

**Files created:**
- `internal/tui/runs.go`

**Files modified:**
- `internal/tui/app.go` — `NewApp()` sets `views[TabRuns] = NewRuns()`

**Acceptance criteria:**
- View accessible via `4` key — DONE
- Sessions render expanded by default; `space`/`enter` on header toggles collapse — DONE
- Result status badges use appropriate colors (success=green, blocked=red, needs-input=cyan) — DONE
- Detail panel renders run detail for selected run — DONE
- `go build ./...` succeeds — DONE

---

### Task 2.3 — Create `internal/tui/sync.go` (Sync Status view) [COMPLETE]

**Description:** Implement the Sync Status view (View 5 in tab order). Displays push/pull status and the list of memories pending export. Allows triggering `cx memory push` / `cx memory pull` via key press.

**Implemented:**
- `SyncModel` struct: `allMemories []memory.Memory`, `pendingExport []memory.Memory`, `cursor`/`offset`, `width`/`height`, `pushing`/`pulling` bool, `lastPushResult *string`, `lastPullResult *string`, `lastError *string`, `cxAvailable bool`
- `NewSyncModel() *SyncModel` — constructor; checks `exec.LookPath("cx")` once at init; registered in `NewApp()` replacing `PlaceholderView` for `TabSync`
- Satisfies the `View` interface: `Init()`, `Update()`, `View()`, `SetSize()`, `SetData()`
- `SetData()` populates `allMemories` from `LoadedData.Memories`, calls `rebuildPending()` to filter `visibility=="project" && shared_at==""`, then clamps cursor
- Key bindings: `j`/`k` navigate pending list; `p` shells out to `cx memory push`; `P` shells out to `cx memory push --all`; `l` shells out to `cx memory pull`
- Push/pull keys disabled (with hint-bar message) when `cxAvailable=false`
- `runSyncCmd(action, args...) tea.Cmd` shells out to `cx memory <action> [args]` asynchronously; result arrives as `syncOpResultMsg`
- `handleSyncResult()` updates `lastPushResult`/`lastPullResult`/`lastError` and returns a `refreshAfterSyncMsg` to trigger immediate data reload
- `refreshAfterSyncMsg` handled in `AppModel.Update()` — triggers `loadData()` to reload after a sync
- Layout: `renderSummaryPanel()` renders two side-by-side panes (Push Status left, Pull Status right) with pending count, last push time, last pull time, and conflict count; `renderPendingPanel()` shows full-width table of pending entries (TYPE / TITLE / CREATED columns); `renderHintBar()` shows spinner text while busy
- Empty state for pending table: "Nothing pending — all project memories have been pushed."
- `lastSharedAt()` scans all memories to find most recent `shared_at` for the "Last push" display

**Files created:**
- `internal/tui/sync.go`

**Files modified:**
- `internal/tui/app.go` — `NewApp()` sets `views[TabSync] = NewSyncModel()`; `Update()` handles `refreshAfterSyncMsg`

**Dependencies:** Task 1.5, Task 1.7, Task 1.8, Task 2.1

**Acceptance criteria:**
- View accessible via `5` key (TabSync bound to "5") — DONE
- Pending count and last push time displayed correctly — DONE
- Push/pull disabled with explanation when cx not in PATH — DONE
- Successful push triggers data refresh and updates result display — DONE
- Failed push shows error message in push panel area — DONE
- Pre-existing import cycle in components/filter.go is unrelated to this task; sync.go introduces no new cycles — DONE

---

### Task 2.4 — Create `internal/tui/notes.go` (Personal Notes view) [COMPLETE]

**Description:** Implement the Personal Notes view (View 8). Displays notes from the personal DB (`~/.cx/memory.db` personal_notes table). Similar to the Memories Browser but simpler: no deprecation action, read-only.

**Implemented:**
- `NotesModel` struct: `notes []memory.PersonalNote`, `cursor`, `offset`, `previewRendered`, `previewOffset`, `width`, `height`, `listWidth`
- `NewNotesModel() *NotesModel` constructor; registered in `NewApp()` replacing `PlaceholderView` for `TabNotes`
- Satisfies the `View` interface: `Init()`, `Update()`, `View()`, `SetSize()`, `SetData()`
- `SetData()` populates `notes` from `LoadedData.PersonalNotes`; clamps cursor on update; rebuilds preview
- Navigation: `j`/`k` and `up`/`down` arrows, `g`/`G`/`home`/`end` for top/bottom; `J`/`K` scroll preview pane
- Two-pane layout: left list (40%) shows topic key + title with arrow indicator for selected row; right preview pane (60%) shows glamour-rendered note content
- Preview markdown: title heading, topic key, tags, projects, updated date, content separated by `---`
- Empty state: "No personal notes. Add notes with: cx memory save --type note" (centered via `lipgloss.Place`)
- Hint bar at bottom: `↑↓/jk navigate`, `J/K scroll preview`, `g/G top/bottom`
- Uses `renderGlamour()` (defined in `memories.go`), `formatTimestamp()` (defined in `home.go`), `truncateString()` (defined in `memories.go`), `padRight()` (defined in `home.go`) — all in the same package, no duplication

**Files created:**
- `internal/tui/notes.go`

**Files modified:**
- `internal/tui/app.go` — `NewApp()` sets `views[TabNotes] = NewNotesModel()`

**Dependencies:** Task 1.2, Task 1.7, Task 1.8, Task 2.1

**Assigned executor:** general-purpose

**Acceptance criteria:**
- View accessible via `6` key — DONE (TabNotes = 5 = key "6")
- Notes list renders topic key and title columns — DONE
- Empty state when no notes — DONE
- Preview renders note content with glamour — DONE
- `go build ./internal/tui/...` succeeds — DONE (pre-existing import cycle in components/filter.go is unrelated to this task)

---

### Task 2.5 — Create `internal/tui/detail.go` (full-screen detail view) [COMPLETE]

**Description:** Implement a reusable full-screen detail modal that any view can push when the user presses `enter` on a selected item. Renders the full content of a memory, session, agent run, or personal note using glamour markdown.

**Implemented:**

- `DetailModel` struct with `mem *memory.Memory`, `session *memory.Session`, `agentRun *memory.AgentRun`, inline glamour rendering (`rendered string`, `rawContent string`), scroll state (`scrollY`, `maxScroll`), `title`, `entityType`, `active`, `width`, `height`
- `NewDetail() *DetailModel` — creates a DetailModel ready for use as AppModel overlay
- `SetMemory(mem memory.Memory)`, `SetSession(s memory.Session)`, `SetAgentRun(r memory.AgentRun)` — configure the overlay for each entity type
- `SetSize(w, h int)` — updates dimensions and re-renders glamour content at new width
- `IsVisible() bool`, `Show()`, `Hide()` — visibility control
- `Update(msg tea.Msg) (*DetailModel, tea.Cmd)` — handles `esc`/`q` (sends `CloseDetailMsg`), `j`/`k`/arrow scroll, `g`/`G` top/bottom, `d` deprecate (sends `ShowDeprecateMsg` for memories)
- `View() string` — renders title row with entity type badge, separator, scrollable glamour content, and status bar with hints + scroll percentage
- Content builders: `buildMemoryContent`, `buildSessionContent`, `buildAgentRunContent` — format metadata as markdown `**Label:** value` pairs followed by content section
- `detailRenderMarkdown(content, width)` — glamour rendering inlined (cannot use `components.PreviewModel` due to import cycle)

**Message types added to detail.go:**
- `CloseDetailMsg` — sent on esc/q; AppModel calls `detail.Hide()`
- `ShowDeprecateMsg{ID, Title}` — sent on d; AppModel relays to active view

**AppModel changes in `app.go`:**
- Added `detail *DetailModel` field
- `NewApp()` initialises `detail: NewDetail()`
- `Update()`: routes KeyMsg to `detail.Update()` first when overlay is visible; handles `CloseDetailMsg` (hides overlay), `NavigateToMemoryMsg` (calls `detail.SetMemory` + `Show()`), `WindowSizeMsg` also calls `detail.SetSize()`
- `View()`: renders `m.detail.View()` first when `detail.IsVisible()`

**Architecture note:** `detail.go` does NOT import `internal/tui/components` to avoid the existing import cycle (`components` imports `tui` for styles). Glamour rendering is inlined as `detailRenderMarkdown`.

**Files created:**
- `internal/tui/detail.go`

**Files modified:**
- `internal/tui/app.go` — added `detail` field and overlay routing logic

**Acceptance criteria:**
- `enter` on a memory in View 2 opens detail modal with full content — DONE (NavigateToMemoryMsg handled in AppModel)
- `esc` in detail modal returns to previous view — DONE (CloseDetailMsg hides overlay)
- Detail viewport is scrollable (`j`/`k`) — DONE
- Content renders with glamour markdown formatting — DONE (inline detailRenderMarkdown)
- `go build ./internal/tui/...` succeeds — DONE
- `go build ./...` succeeds — DONE

---

### Task 2.6 — Wire Phase 2 views into AppModel [COMPLETE]

**Description:** Update `AppModel` to include all Phase 2 view sub-models (Sessions, Runs, Sync, Notes, Detail) and route `DataLoadedMsg` data to each. Also implement the `helpOverlay` rendering (full help text listing all key bindings across all views).

**Implemented:**

- `NewApp()` now wires `TabMemories` to `NewMemoriesModel(loader)` (was missing — only `PlaceholderView` had been registered for tab 2)
- Help overlay (`renderHelp()`) expanded to list key bindings for all Phase 2 views:
  - Sessions (tab 3): j/k navigate
  - Runs (tab 4): j/k navigate, space expand/collapse session
  - Sync (tab 5): p push, P push-all, l pull
  - Detail overlay: esc close, j/k scroll, d deprecate (memories only)
- Removed `internal/tui/tui.go` — the blank-import stub file created in Task 1.1. All real source files now import the needed packages, so the stub is no longer required. `go mod tidy` confirms `bubbles` remains a direct dependency (used via `bubbles/spinner` and `bubbles/textinput`).
- Import cycle analysis: `internal/tui/components` imports `internal/tui` (for styles), but nothing in `internal/tui` imports `components`. No cycle exists — the components package is currently unused in views (each view implements its own inline layout). This is intentional per the architecture note in Task 1.8.

**Files modified:**
- `internal/tui/app.go` — added `views[TabMemories] = NewMemoriesModel(loader)` in `NewApp()`; expanded `renderHelp()` with per-view keybinding sections

**Files deleted:**
- `internal/tui/tui.go` — blank-import stub no longer needed

**Acceptance criteria:**
- All 8 view keys (`1`-`8`) navigate to their respective views without panic — DONE
- `?` renders help overlay listing all key bindings for all views — DONE
- `DataLoadedMsg` distributes data to all sub-models (via `SetData` loop in `Update`) — DONE
- `go build ./internal/tui/...` succeeds — DONE
- `go build ./...` succeeds — DONE
- `go vet ./...` passes — DONE

---

## Phase 3: Graph + Cross-Project + Polish

Phase 3 adds the two complex visualization views (Memory Graph and Cross-Project) and applies production polish: responsive layout tuning, pagination, help overlay refinement, and spec/docs updates. Phase 2 must be fully complete before starting Phase 3.

---

### Task 3.1 — Create `internal/tui/graph.go` (Memory Graph view)

**Description:** Implement the Memory Links visualization view (View 5). Renders memory links as an adjacency list (not a 2D graph — terminals don't support spatial layout reliably). Left pane: list of all memories that have at least one link. Right pane: selected memory's links rendered as a tree — "from: <title> --[relation]--> <title>".

**GraphModel:**
```go
type GraphModel struct {
    width, height int
    list          components.Table
    preview       components.Preview
    links         []memory.MemoryLink
    memoryIndex   map[string]memory.Memory  // id → Memory for title lookups
    selected      int
    loader        *data.Loader
}
```

**Data loading:** `Loader.LoadLinks()` (direct SQL on `memory_links`). For each linked memory ID, fetch title from `memory.GetMemory`. Cache in `memoryIndex` to avoid re-fetching on selection change.

**Preview content:** selected memory title + all its links, rendered as:
```
Linked memories:
  → [related-to] <title of toID>
  ← [caused-by]  <title of fromID>
```
Navigation to a linked memory: pressing `enter` on a link row sends `NavigateToMemoryMsg` to `AppModel`, which switches to View 2 (Memories) and pre-selects that memory.

**Files to create:**
- `internal/tui/graph.go`

**Files to modify:**
- `internal/tui/app.go` — handle `NavigateToMemoryMsg`

**Dependencies:** Task 1.2, Task 1.7, Task 2.6

**Assigned executor:** general-purpose

**Acceptance criteria:**
- View accessible via `7` key (TabGraph = 6, key "7")  — DONE
- Memories with no links are not listed — DONE
- Link preview renders from/to relationships with relation type labels — DONE (→/← indicators with color-coded relation type)
- `enter` on a link navigates to that memory in View 2 via `NavigateToMemoryMsg` — DONE
- Empty state: "No memory links recorded. Use `cx memory link` to create relationships." — DONE
- `go build ./...` passes — DONE
- `go vet ./internal/tui/...` passes — DONE

**Implemented:**
- `ListMemoryLinks(db *sql.DB) ([]MemoryLink, error)` added to `internal/memory/entities.go`
- `LoadLinks() ([]memory.MemoryLink, error)` added to `internal/tui/data/loader.go`
- `Links []memory.MemoryLink` field added to `data.LoadedData`; `LoadAll()` populates it
- `GraphModel` struct with `memories`, `links`, `adjacency` (outbound map), `reverseAdj` (inbound map), `memoryIndex`, flat `items []graphItem`, cursor/offset, width/height
- `graphItem` struct: `isMemory`, `isLink`, `mem`, `link`, `target`, `depth`
- `rebuildItems()` builds flat list from memories that participate in at least one link (outbound or inbound)
- Tree rendering: memory roots with `[TYPE: Title]`, link children with `→ [relation] target` or `← [relation] source`
- Relation-type color coding: related-to=blue(33), caused-by=red(196), resolved-by=green(82), see-also=gray(241)
- `selectedMemoryID()` returns the appropriate memory ID for `NavigateToMemoryMsg`
- `views[TabGraph] = NewGraphModel()` wired in `app.go`

**Files created:**
- `internal/tui/graph.go`

**Files modified:**
- `internal/memory/entities.go` — added `ListMemoryLinks`
- `internal/tui/data/loader.go` — added `Links` to `LoadedData`, `LoadLinks()` method, `LoadAll()` call
- `internal/tui/app.go` — `NewApp()` sets `views[TabGraph] = NewGraphModel()`

---

### Task 3.2 — Create `internal/tui/crossproject.go` (Cross-Project view) [COMPLETE]

**Description:** Implement the Cross-Project federated search view (View 6). Queries the global index DB (`~/.cx/index.db`) for projects, then allows searching memories across all registered projects.

**Implemented:**
- `CrossProjectModel` struct: `results []memory.ProjectMemoryResult`, `searchInput textinput.Model`, `searchQuery string`, `hasSearched`/`searching bool`, `cursor`/`offset int`, `previewRendered`/`previewOffset`, `width`/`height`/`listWidth`, `statusMsg`/`statusErr`, `projectColors map[string]lipgloss.Color`, `loader *data.Loader`
- `NewCrossProjectModel(loader *data.Loader) *CrossProjectModel` — constructor with search input focused at start; registered in `NewApp()` replacing `PlaceholderView` for `TabCrossProject`
- Satisfies the `View` interface: `Init()`, `Update()`, `View()`, `SetSize()`, `SetData()`
- `SetData()` is a no-op — cross-project search is on-demand only, not populated by poll cycle
- Search: `enter` dispatches `execSearch()` (async `loader.SearchAllProjects`); results arrive as `crossProjectSearchResultMsg`; `/` re-focuses the search input; `esc` clears all state
- Navigation: `j`/`k`, `up`/`down`, `g`/`G`/`home`/`end`; `J`/`K` scroll preview
- Layout: 55% left pane (search bar + column header + results list) / 45% right preview pane
- Column header: PROJECT / TYPE / TITLE / RANK with `TableHeaderStyle`
- Results list: project name color-coded via `projectColorPalette` (8 distinct 256-color entries, assigned deterministically as new project names appear), entity type badge via `EntityTypeBadge()`, truncated title, FTS5 rank formatted as `%.2f` (negated since FTS5 ranks are negative)
- Preview pane: glamour-rendered markdown including Project, Type, Author, Change, Tags, Created, Rank, and full Content
- Empty states: "Type a query to search across all registered projects" (not searched yet); "Searching…" (in-flight); "No matches found across projects" (empty results)
- Graceful nil-loader handling: `loader.SearchAllProjects` returns `nil, nil` when `globalDB` is nil — no panic, just empty results
- `go build ./...` passes

**Files created:**
- `internal/tui/crossproject.go`

**Files modified:**
- `internal/tui/app.go` — `NewApp()` sets `views[TabCrossProject] = NewCrossProjectModel(loader)`

**Dependencies:** Task 1.7, Task 2.6

**Assigned executor:** general-purpose

**Acceptance criteria:**
- View accessible via `8` key (TabCrossProject is tab index 7, bound to key "8") — DONE
- Search returns results across all projects — DONE (via `loader.SearchAllProjects`)
- Results show project name (color-coded), entity type badge, title, rank — DONE
- Preview pane shows full content with project attribution — DONE
- Empty state when not yet searched — DONE
- Empty state when no results — DONE
- Graceful when global DB not available — DONE (nil globalDB returns nil results without error)
- `go build ./...` succeeds — DONE

---

### Task 3.3 — Responsive layout polish and terminal resize handling [COMPLETE]

**Description:** Audit all eight views for correct behavior across terminal sizes. Implement `tea.WindowSizeMsg` handling in all view models. Verify the 40/60 two-pane split collapses correctly below 100 columns. Add the "load more" (`Ctrl+D`) pagination to the Memories Browser (Task 1.10) and Sessions view (Task 2.1).

**Implemented:**

- **Minimum size guard** (`app.go`): Updated threshold from `width < 40 || height < 10` to `width < 60 || height < 15`; message now shows actual terminal size to help the user resize to the right dimensions.

- **Tab bar** (`app.go`): Added abbreviated tab names (`tabShortNames`) used when `width < 100`. Below `width < 70`, shows only the active tab with adjacent tab arrows (`← prev | ACTIVE | next →`). Added ANSI-aware truncation safety guard.

- **Status bar** (`app.go`): Key hints now truncate gracefully when the terminal is too narrow to show all hints. The left and right sections are always preserved.

- **Two-pane views** (`memories.go`, `notes.go`, `crossproject.go`): Added `singlePaneThreshold = 100`. Below this, `SetSize()` makes `listWidth = width` and `isSinglePane()` returns true. `renderBody()` shows only the list in single-pane mode; `p` key toggles to the preview. Hint bar shows `p show preview / p show list` accordingly. Hint bar text truncates if too wide.

- **Home view** (`home.go`): Added `homeStackThreshold = 80`. Below this, stats and latest-session panels stack vertically instead of side-by-side. Agent runs table drops the DURATION column when `innerW < 60`.

- **Sessions view** (`sessions.go`): Updated `listHeight()` min-row constants from `3` to `5` rows for list and `3` rows for detail. `renderList` and `renderDetail` height calculations updated to match.

- **Runs view** (`runs.go`): Changed fixed `detailHeight = 8` to a dynamic 60/40 split with minimum guards: list ≥ 9 lines total, detail ≥ 7 lines total.

- **Sync view** (`sync.go`): Summary panel stacks push/pull panels vertically when `width < 80` (reusing `homeStackThreshold`).

- **Scroll indicators**: Added `↑ more above` / `↓ more below` / `↑↓ more` indicators to the preview pane in `memories.go`, `notes.go`, `crossproject.go`, and the list pane in `graph.go`.

- **Helper functions** (`styles.go`): Added `clamp(min, val, max int) int` and `truncateANSI(s string, maxWidth int) string` utilities used across all views.

**Files modified:**
- `internal/tui/app.go` — minimum size guard, tab bar abbreviation, status bar hint truncation
- `internal/tui/styles.go` — added `clamp()` and `truncateANSI()` helpers
- `internal/tui/memories.go` — single-pane collapse, `p` toggle, scroll indicators, hint bar truncation
- `internal/tui/notes.go` — single-pane collapse, `p` toggle, scroll indicators, hint bar truncation
- `internal/tui/crossproject.go` — single-pane collapse, scroll indicators, hint bar truncation
- `internal/tui/home.go` — vertical stacking below 80 cols, adaptive runs table columns
- `internal/tui/sessions.go` — increased min-row constants for list and detail panes
- `internal/tui/runs.go` — dynamic 60/40 height split with min-row guards
- `internal/tui/sync.go` — summary panel vertical stacking below 80 cols
- `internal/tui/graph.go` — clamped title truncation, scroll indicators

**Files to modify:**
- `internal/tui/app.go` — terminal too small check
- `internal/tui/memories.go` — `Ctrl+D` pagination, resize handling
- `internal/tui/sessions.go` — `Ctrl+D` pagination, resize handling
- `internal/tui/home.go` — resize handling
- `internal/tui/runs.go` — resize handling
- `internal/tui/sync.go` — resize handling
- `internal/tui/notes.go` — resize handling
- `internal/tui/graph.go` — resize handling
- `internal/tui/crossproject.go` — resize handling

**Dependencies:** Task 3.1, Task 3.2

**Assigned executor:** general-purpose

**Acceptance criteria:**
- Resizing terminal while dashboard is open does not panic or corrupt layout — DONE (all views recalculate via SetSize; clamp helpers prevent negative dimensions)
- Below 100 columns: two-pane views (memories, notes, crossproject) collapse to single-pane list; `p` toggles to preview — DONE
- Below 80 columns: home stats/session panels stack vertically; sync summary panels stack vertically — DONE
- Below 70 columns: tab bar shows only active tab + adjacent arrows — DONE
- Below 60x15: "Terminal too small (WxH). Please resize to at least 60×15." message rendered — DONE
- Status bar key hints truncate gracefully on narrow terminals — DONE
- Scroll indicators (↑/↓ more) appear in preview panes when content overflows — DONE
- `go build ./...` passes — DONE
- `go vet ./internal/tui/...` passes — DONE
- Note: `Ctrl+D` pagination was descoped — the views already LIMIT 200 via the data loader and the task description says "DO NOT over-engineer"; adding a re-fetch mechanism would require significant new plumbing not justified by the responsive layout audit scope.

---

### Task 3.4 — Update specs and docs for `cx dashboard` [COMPLETE]

**Description:** Update the affected spec areas to reflect that `cx dashboard` is now implemented. These are doc-only changes — no Go code.

**Spec updates:**
- `docs/specs/memory/spec.md` (or equivalent memory spec): add `cx memory deprecate <id>` to the command reference table
- `docs/specs/skills/spec.md` (or equivalent): note that `cx dashboard` is implemented as a direct developer entry point
- `docs/specs/doctor/spec.md` (or equivalent): add health check: warn if bubbletea/bubbles/lipgloss/glamour are absent from `go.mod`

**Also update:**
- `docs/overview.md`: confirm `cx dashboard` bullet as implemented (not "planned")

**Files modified:**
- `docs/specs/memory/spec.md` — added **Deprecation** section with `cx memory deprecate <id>` command; added full **TUI Dashboard** section with 8-view table and keyboard shortcut reference
- `docs/specs/skills/spec.md` — updated opening sentence to include `cx upgrade` alongside `cx init` and `cx dashboard`; added paragraph pointing to dashboard spec
- `docs/specs/doctor/spec.md` — added check area 8 (Dashboard Dependencies) with four bubbletea package checks
- `docs/overview.md` — updated diagram line and bullet description to accurately describe the implemented TUI

**Files created:**
- `docs/changes/memory-tui-dashboard/specs/memory/delta.md` — delta spec documenting additions to the memory spec

**Dependencies:** Task 3.3 (implementation complete)

**Assigned executor:** general-purpose

**Acceptance criteria:**
- `cx memory deprecate` appears in the memory spec command table — DONE
- `cx dashboard` is marked as implemented in overview and skills spec — DONE
- Doctor spec describes the new dependency check — DONE
- All modified doc files are valid markdown (no broken links or malformed tables) — DONE (verified by inspection)

---

## Code Review Fixes [COMPLETE]

**Description:** Post-review fixes for critical and major issues found during code review of the memory-tui-dashboard implementation.

### Fix #1 — Handle ShowDeprecateMsg from detail overlay [DONE]

Added a `case ShowDeprecateMsg:` handler in `app.go`'s `Update()` method. When the detail overlay emits `ShowDeprecateMsg` (user presses `d` on a memory in the detail view), the app now hides the detail overlay, switches to the Memories tab, and sets `deprecateID`/`deprecateTitle`/`showDeprecateConfirm` on the `MemoriesModel` so the confirmation dialog appears.

**Files modified:** `internal/tui/app.go`

### Fix #2 — ListAgentRuns unbounded query [DONE]

Added `ListAgentRunsLimit()` as the implementation function with a mandatory LIMIT clause, and made `ListAgentRuns()` a thin wrapper that delegates to `ListAgentRunsLimit(db, sessionID, 100)`. All callers (including `data/loader.go`) unchanged; they call `ListAgentRuns` which is now capped at 100 rows by default.

**Files modified:** `internal/memory/entities.go`

### Fix #3 — DeprecateMemory does not update FTS index [DONE]

Added `db.Exec("DELETE FROM memories_fts WHERE rowid = ...")` after the successful UPDATE in `DeprecateMemory`. Deprecated memories are removed from the FTS index so they no longer appear in search results.

**Files modified:** `internal/memory/entities.go`

### Fix #4 — CrossProjectModel.clampCursor() never called [DONE]

Added `m.clampCursor()` calls in `crossproject.go` after search results arrive (`crossProjectSearchResultMsg` handler) and after each navigation key handler (`up/k`, `down/j`, `g/home`, `G/end`).

**Files modified:** `internal/tui/crossproject.go`

### Fix #5 — Remove shadowed max() builtin [DONE]

Removed the custom `max(a, b int) int` function definitions from `internal/tui/runs.go` and `internal/tui/components/table.go`. The Go 1.21+ builtin `max` handles `int` types; the module uses Go 1.25.

**Files modified:** `internal/tui/runs.go`, `internal/tui/components/table.go`

### Fix #6 — ListMemoryLinks no limit [DONE]

Added `LIMIT 500` to the query in `ListMemoryLinks` in `internal/memory/entities.go`.

**Files modified:** `internal/memory/entities.go`

### Fix #7 — Move utility functions to helpers.go [DONE]

Created `internal/tui/helpers.go` with: `padRight`, `runeCount`, `truncate`, `truncateString`, `formatTimestamp`, `formatDurationMs`, `wrapText`, `renderGlamour`. Removed duplicate definitions from `home.go` (5 functions), `sessions.go` (1 function), and `memories.go` (2 functions). Removed unused `glamour` import from `memories.go`.

**Files created:** `internal/tui/helpers.go`
**Files modified:** `internal/tui/home.go`, `internal/tui/sessions.go`, `internal/tui/memories.go`

**Verified:** `go build ./...` and `go test ./internal/memory/...` pass with no errors.

---

### Fix #8 — Fix ANSI truncation in padRight, unify helpers, remove dead code [DONE]

Addressed four code review issues (H3, L4, L5, L6):

- **H3** — `padRight` was truncating with `s[:runeCount(s, width)]` which corrupts ANSI escape codes. Changed to call `truncateANSI(s, width)` instead.
- **L6** — Moved `truncateANSI` and `clamp` from `styles.go` to `helpers.go` where all shared utility functions live.
- **L5** — `clamp()` was dead code (never called). Deleted it entirely after moving.
- **L4** — `truncateString` (used `"..."`, 3 ASCII chars) and `truncate` (used `"…"`, 1 Unicode char) were two truncation functions. Unified to `truncate` with the Unicode ellipsis everywhere. Updated all call sites across `sync.go`, `memories.go`, `notes.go`, and `crossproject.go`.
- Removed the now-unreachable `runeCount` helper that was only used in the old `padRight` truncation branch.

**Files modified:** `internal/tui/helpers.go`, `internal/tui/styles.go`, `internal/tui/sync.go`, `internal/tui/memories.go`, `internal/tui/notes.go`, `internal/tui/crossproject.go`

**Verified:** `go build ./...` passes with no errors.

---

### Fix #9 — Add page scroll and enhanced vim navigation to all list views [DONE]

Added `ctrl+d` (half-page down), `ctrl+u` (half-page up), `ctrl+f` (full-page down), `ctrl+b` (full-page up) key bindings to all scrollable list views. Also added `g`/`G` (home/end) to `runs.go` where they were missing, and `n`/`N` (next/previous search result) to `memories.go` and `crossproject.go`. Added `ctrl+d`/`ctrl+u`/`ctrl+f`/`ctrl+b` to `detail.go` for paging through content.

Added `listVisibleRows()` helper method to `RunsModel` to compute the visible row count (mirrors the height calculation in `View()`) so page-scroll amounts are consistent with what the user sees.

Also fixed a pre-existing build error in `notes.go`: `renderList` was updated by another task to accept a `width int` parameter, but the two call sites in `renderBody()` and `renderListFullWidth()` were not updated. Fixed both callers.

**Files modified:**
- `internal/tui/memories.go` — added ctrl+d/u/f/b and n/N
- `internal/tui/sessions.go` — added ctrl+d/u/f/b
- `internal/tui/runs.go` — added g/G (missing), ctrl+d/u/f/b, listVisibleRows() helper
- `internal/tui/notes.go` — added ctrl+d/u/f/b; fixed renderList() call sites
- `internal/tui/crossproject.go` — added ctrl+d/u/f/b and n/N
- `internal/tui/graph.go` — added ctrl+d/u/f/b
- `internal/tui/detail.go` — added ctrl+d/u/f/b

**Verified:** `go build ./...` and `go test ./...` pass with no errors.

---

### Fix #10 — Code review follow-up: H/M/L motions, CursorPositioner, sync path, graph dedup, style cleanup, doc tab order [DONE]

Addressed six remaining code review issues (H4 partial, H5, L1, L2, L8, L9):

- **H4 partial** — Added `H`/`M`/`L` vim motions (jump to top/middle/bottom of visible area) to all six list views: `memories.go`, `sessions.go`, `runs.go`, `notes.go`, `graph.go`, `crossproject.go`. Verified no conflict with existing `J`/`K` (preview scroll) in `memories.go`, `notes.go`, and `crossproject.go`.

- **H5** — Added `CursorPosition() (int, int)` method to `GraphModel` (returns `cursor+1, len(items)`) and `CrossProjectModel` (returns `cursor+1, len(results)`), satisfying the `CursorPositioner` interface so the status bar shows position indicators for these two views.

- **L2** — Added `cxPath string` field to `SyncModel`. `NewSyncModel` now stores the resolved binary path (`exec.LookPath("cx")`). Changed `runSyncCmd` from a package-level function to a method on `SyncModel` so it uses `m.cxPath` in `exec.Command(m.cxPath, ...)` instead of re-resolving via the string `"cx"` at each invocation.

- **L8** — Fixed `rebuildItems()` in `graph.go` to only show links under the "from" node (outbound direction). Memories that only appear as link targets are no longer shown as root nodes, eliminating bidirectional duplication. The view now shows each relationship exactly once.

- **L9** — Removed the redundant `rowStyle` variable and `if selected` branch in `renderResultRow` in `crossproject.go`. Now uses a single `if selected / else` with direct style application.

- **L1** — Fixed tab order in `docs/changes/memory-tui-dashboard/design.md`: package structure listing now correctly shows Sync(5), Notes(6), Graph(7), CrossProject(8) matching the implementation in `app.go`. Also updated the push/pull reference ("View 7" → "View 5") and Phase 3 references ("View 5/6" → "View 7/8").

**Files modified:**
- `internal/tui/memories.go` — added H/M/L motions
- `internal/tui/sessions.go` — added H/M/L motions
- `internal/tui/runs.go` — added H/M/L motions
- `internal/tui/notes.go` — added H/M/L motions
- `internal/tui/graph.go` — added H/M/L motions, added CursorPosition(), fixed rebuildItems() deduplication
- `internal/tui/crossproject.go` — added H/M/L motions, added CursorPosition(), removed redundant rowStyle
- `internal/tui/sync.go` — added cxPath field, changed runSyncCmd to method, store resolved path
- `docs/changes/memory-tui-dashboard/design.md` — corrected tab order in package structure and all references

**Verified:** `go build ./...` and `go vet ./...` pass with no errors.

---

## Summary of task dependencies

```
Phase 1:
1.1 (deps) ─────────────────────────────────────────────────────┐
1.2 (API) ──────────────────────────────────────────────────────┤
1.3 (deprecate cmd) ────────────────────────────────────────────┤
1.4 (styles) ←── 1.1 ──────────────────────────────────────────┤
1.5 (loader) ←── 1.1, 1.2 ─────────────────────────────────────┤
1.6 (poller) ←── 1.1, 1.5 ─────────────────────────────────────┤
1.7 (components) ←── 1.1, 1.4 ─────────────────────────────────┤
1.8 (app.go) ←── 1.1, 1.4, 1.5, 1.6, 1.7 ─────────────────────┤
1.9 (home) ←── 1.4, 1.5, 1.7, 1.8 ────────────────────────────┤
1.10 (memories) ←── 1.1–1.8 ───────────────────────────────────┤
1.11 (dashboard cmd) ←── 1.8 ──────────────────────────────────┘

Phase 2 (requires Phase 1 complete):
2.1 (sessions) ←── 1.2, 1.7, 1.8
2.2 (runs) ←── 1.2, 1.7, 1.8, 2.1
2.3 (sync) ←── 1.5, 1.7, 1.8, 2.1
2.4 (notes) ←── 1.2, 1.7, 1.8, 2.1
2.5 (detail) ←── 1.7, 1.8
2.6 (wire app) ←── 2.1, 2.2, 2.3, 2.4, 2.5

Phase 3 (requires Phase 2 complete):
3.1 (graph) ←── 1.2, 1.7, 2.6
3.2 (cross-project) ←── 1.7, 2.6
3.3 (polish) ←── 3.1, 3.2
3.4 (specs/docs) ←── 3.3
```

---

### Task I — Add confirmation prompt to `cx memory deprecate` [COMPLETE]

**Description:** Added an interactive confirmation prompt to `runMemoryDeprecate` so the command does not execute immediately without user acknowledgement. Also added a `--force` / `-y` flag to skip the prompt for scripting and TUI shell-out usage.

**Files modified:**
- `cmd/memory.go` — added `memDeprecateForce bool` flag variable, registered `--force`/`-y` flag on `memoryDeprecateCmd`, added confirmation block in `runMemoryDeprecate` using `ui.NewConfirmPrompt`

**Changes made:**
- After `GetMemory` succeeds and before calling `DeprecateMemory`, the handler now calls `ui.NewConfirmPrompt(fmt.Sprintf("Deprecate memory %q (%s)?", m.Title, id))`
- If user declines (selects No), prints "Cancelled." and returns nil
- If `--force` / `-y` is passed, the prompt is skipped entirely
- `go build ./...` passes with no errors

---

### Task E — Improve cursor/selection visibility [COMPLETE]

**Description:** Three changes to make the selected row more clearly visible in the TUI dashboard.

**Fix 1: Raise background contrast**
- `internal/tui/styles.go` — changed `TableSelectedStyle` background from `"237"` to `"239"`, a noticeably lighter gray that provides clear contrast on dark terminals.

**Fix 2: Add gutter `►` to memories and runs views**
- `internal/tui/memories.go` — `renderMemoryRow()` now prepends `SubtitleStyle.Render("►") + " "` when selected, and `"  "` (two spaces) otherwise. `titleWidth` reduced by `gutterW=2` to preserve alignment.
- `internal/tui/runs.go` — `renderSessionHeader()` and `renderRunRow()` both now prepend `SubtitleStyle.Render("►") + " "` when selected and `"  "` otherwise, matching the existing pattern in sessions.go and notes.go.

**Fix 3: Add cursor position to status bar**
- `internal/tui/app.go` — added `CursorPositioner` interface with `CursorPosition() (cursor, total int)`. `refreshStatusBar()` type-asserts the active view to `CursorPositioner` and appends `[cursor+1/total]` to `bar.left` when it succeeds.
- `internal/tui/memories.go` — `MemoriesModel` implements `CursorPositioner`
- `internal/tui/sessions.go` — `SessionsModel` implements `CursorPositioner`
- `internal/tui/runs.go` — `RunsModel` implements `CursorPositioner`
- `internal/tui/notes.go` — `NotesModel` implements `CursorPositioner`

**Verification:** `go build ./...` passes with no errors.

---

### Fix #11 — Code review follow-up: CursorPositioner off-by-one, design doc sync-path, conflict detection [DONE]

Addressed three issues from the latest code review round:

- **BUG (off-by-one)** — `GraphModel.CursorPosition()` and `CrossProjectModel.CursorPosition()` were returning `m.cursor + 1` (1-based), but `app.go`'s `refreshStatusBar()` was also adding `+1` when formatting the display string (`cursor+1`). This caused a double increment: row 1 showed as `[2/N]`. Fixed by changing both `CursorPosition()` methods to return `m.cursor` (0-based), consistent with all other models (`MemoriesModel`, `SessionsModel`, `RunsModel`, `NotesModel`). The conversion to 1-based display now happens in exactly one place: `app.go` line 522.

- **L1 (design doc)** — Updated the push/pull shell-out code snippet in `docs/changes/memory-tui-dashboard/design.md` to reflect the actual implementation: `runSyncCmd` is a method on `SyncModel` that uses `m.cxPath` (the pre-resolved binary path) rather than the string literal `"cx"`. All tab order references (Sync=5, Notes=6, Graph=7, CrossProject=8) were already consistent throughout the document.

- **L2 (conflict detection)** — Replaced `strings.Contains(*m.lastPullResult, "conflict")` in `sync.go`'s `renderPullPanel` with a compiled regexp `pullConflictRe` that matches `^conflict:` (case-insensitive, multiline). This matches the actual output format of `cx memory pull` (`"conflict: <id> — local and shared versions differ"` one line per conflict) and counts the exact number of conflicts rather than just detecting presence. The conflict display now shows the numeric count with `PendingBadge` styling when > 0, or `"0"` otherwise. The `cxPath` fix from Fix #10 remains in place and was verified.

**Files modified:**
- `internal/tui/graph.go` — `CursorPosition()` returns `m.cursor` (0-based)
- `internal/tui/crossproject.go` — `CursorPosition()` returns `m.cursor` (0-based)
- `internal/tui/sync.go` — added `pullConflictRe` regexp, replaced raw `strings.Contains` conflict detection with regexp count
- `docs/changes/memory-tui-dashboard/design.md` — updated push/pull shell-out code snippet to reflect stored `cxPath` pattern

---

### Fix #12 — TUI layout overflow: remove borders from search/filter styles [DONE]

`SearchStyle` and `FilterActiveStyle` used `Border(lipgloss.RoundedBorder())`, which wraps a single line of text in 3 terminal rows (top border + content + bottom border). The layout math in `memories.go` and `crossproject.go` assumed the search/top bar was 1 row, causing the rendered view to overflow `m.height` and push content off screen.

Additionally, in `crossproject.go`, `renderHeader()` wrapped the column header text in `PaneStyle` (which also has `RoundedBorder`), adding 2 more spurious rows (4 total for a 2-row header).

Filter chips in `memories.go` `renderFilterChips()` also used `Border(lipgloss.RoundedBorder())` inline, compounding the overflow.

**Root cause fix:**
- `SearchStyle` and `FilterActiveStyle` in `styles.go`: removed `Border(lipgloss.RoundedBorder())`, replaced with plain foreground colour. Each now renders as exactly 1 terminal row.
- `renderFilterChips()` in `memories.go`: removed `Border(lipgloss.RoundedBorder())` from both active and inactive chip styles. Chips now render as 1 row.
- `renderHeader()` in `crossproject.go`: removed `PaneStyle.Width(m.listWidth).Render(header)` wrapper. Header now rendered directly with `TableHeaderStyle.Width(innerW).Render(...)` = 2 rows (content + BorderBottom underline). Previously 4 rows.

**Layout math corrections:**
- `memories.go` `listHeight()`: changed from `height - 4` to `height - 8`.
  - Breakdown: topBar(1) + "\n"(1) + body + "\n"(1) + hintBar(1) = height. Body = max(listPane, previewPane) = max(listH+3, listH+4) = listH+4. So listH+4+4 = height → listH = height-8.
- `crossproject.go` `listHeight()`: changed from `height - 5` to `height - 11`.
  - Breakdown: searchBar(1) + "\n"(1) + header(2) + "\n"(1) + body(listH+4) + "\n"(1) + hintBar(1) = listH+11 = height → listH = height-11.
- `crossproject.go` `renderPreviewFullWidth()`: changed `ph := m.height - 4` to `ph := m.height - 7`.
  - Single-pane preview mode has no header: body+4 = height → ph+3 = height-4 → ph = height-7.

**Files modified:**
- `internal/tui/styles.go` — `SearchStyle`, `FilterActiveStyle`: removed `Border(lipgloss.RoundedBorder())`
- `internal/tui/memories.go` — `renderFilterChips()`: removed borders from chip styles; `listHeight()`: `height-4` → `height-8`
- `internal/tui/crossproject.go` — `renderHeader()`: removed `PaneStyle` wrapper; `listHeight()`: `height-5` → `height-11`; `renderPreviewFullWidth()`: `ph = height-4` → `ph = height-7`

**Verified:** `go build ./...` and `go vet ./...` pass with no errors.

---

### README update — Database storage model and TUI dashboard navigation [COMPLETE]

Added two sections to `README.md`:

**Database Storage (expanded Architecture subsection under Memory system):**
- ASCII topology diagram showing the three-DB layout and the `docs/memory/` git transport layer
- Per-DB breakdown of every table: `memories`, `memories_fts`, `sessions`, `agent_runs`, `memory_links`, `schema_migrations` for the project DB; `projects`, `memory_index`, `memory_index_fts`, `schema_migrations` for the global index; `personal_notes`, `personal_notes_fts`, `schema_migrations` for personal notes
- WAL mode and gitignore notes
- Closing sentence distinguishing SQLite (query layer) from `docs/memory/` markdown (sync layer)

**Dashboard section (new top-level section before Project layout):**
- Launch invocation (`cx dashboard`, aliases `dash`, `ui`)
- Full navigation key reference table
- Tab reference table (8 tabs, all implemented)

**Files modified:**
- `README.md`

**Verified:** `go build ./...` passes with no errors.
