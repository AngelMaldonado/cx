---
name: memory-tui-dashboard
type: proposal
---

## Problem

The `cx` memory system accumulates valuable institutional knowledge — observations, decisions, session histories, agent run logs, and cross-project connections — but today this knowledge is only accessible through CLI commands that return flat text output. Developers must mentally assemble a picture of their project's memory landscape across multiple invocations of `cx memory search`, `cx memory list`, `cx agent-run list`, and separate commands for push/pull sync status.

Specific pain points:

1. **No ambient visibility.** A developer starting work has no quick way to see "what's been happening here?" — recent sessions, agent runs, and sync state are invisible without active querying.

2. **Search is the only entry point.** Without a browse mode, developers who don't know what to search for can't discover relevant memories. FTS5 requires you to have a query in mind.

3. **Relationship opacity.** Memory links (`related-to`, `caused-by`, `resolved-by`, `see-also`) are stored in the DB but invisible in CLI output. A developer cannot see which decisions caused which observations.

4. **Sync state is opaque.** There is no single view of "what has been pushed, what is pending, what conflicts exist."

5. **Agent run history is buried.** `cx agent-run list` dumps rows with no grouping by session or pattern visibility.

6. **Cross-project discovery is awkward.** Managing multiple projects requires knowing what to query for.

`docs/overview.md` already calls out `cx dashboard` as one of three developer entry points: "cx dashboard — TUI for visibility into specs, changes, memory, team sync." This feature delivers that entry point.

## Approach

Implement `cx dashboard` as a fullscreen Bubble Tea TUI with eight navigable views. The TUI reads all three SQLite databases directly via the existing `internal/memory` Go API, polls for updates every 5 seconds, and renders structured browsable views for memories, sessions, agent runs, memory links, cross-project data, and sync status.

The design follows browse-first principles: the default view is a navigable list with a side-by-side preview pane, not a search prompt. Search is always available via `/` but is not required to start. Navigation uses keyboard-native vim-style keys (`j/k`, `g/G`) plus tab, enter, escape, and number keys `1-8`.

The two exceptions to the read-only rule are triggering `cx memory push` and `cx memory pull` from the Sync view — these shell out to the `cx` binary. A third limited write action, `cx memory deprecate <id>`, is available from the Memories Browser with a confirmation prompt.

## Scope

**In scope:**
- `cmd/dashboard.go` — new cobra command `cx dashboard`
- `internal/tui/` — new package with 8 view files, shared components, and data loader
- 8 views: Home/Overview, Memories Browser, Sessions Timeline, Agent Runs, Memory Graph, Cross-Project, Sync Status, Personal Notes
- Read-only access to all three SQLite databases (`.cx/memory.db`, `~/.cx/index.db`, `~/.cx/memory.db`)
- Live 5-second polling refresh via Bubble Tea tick command
- Search, filter, and preview pane in the Memories Browser
- Push and pull trigger actions in the Sync view (shelled out via `exec.Command`)
- Deprecation action in Memories Browser (via new `cx memory deprecate <id>` CLI command)
- Two new Go API functions: `ListSessions` and `ListPersonalNotes` in `internal/memory/entities.go`
- Responsive terminal layout: two-pane (80+ cols) and single-pane (< 80 cols)

**Out of scope:**
- Writing memories, decisions, or sessions from within the TUI
- Web UI, electron, or any separate process/server
- Mouse support
- Sub-second refresh or fsnotify-based file watching
- Keyboard shortcut customization
- Spec, change, or docs viewing (those are future `cx dashboard` expansions)

## Affected Specs

- **memory** — Add `cx memory deprecate <id>` command to the command reference table
- **skills** — Update catalog to confirm `cx dashboard` is implemented; note it in the list of direct developer commands (alongside `cx init`)
- **doctor** — Add check: warn if `cx dashboard` TUI dependencies (bubbletea, bubbles, lipgloss, glamour) are absent from `go.mod` (contributor health check)
- **orchestration** — No structural changes; `cx dashboard` is a new consumer of existing memory API, not a new agent or dispatch path
