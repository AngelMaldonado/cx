// Package tui — runs.go implements the Agent Runs view (tab 4).
// Runs are grouped by session, with collapsible session headers.
// A detail panel at the bottom shows full info for the selected run.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AngelMaldonado/cx/internal/memory"
	"github.com/AngelMaldonado/cx/internal/tui/data"
)

// unattachedSessionID is the synthetic session group key for runs with no session.
const unattachedSessionID = "__unattached__"

// runsItem represents one navigable item in the flat cursor list.
// It is either a session header or a run row.
type runsItem struct {
	isHeader  bool
	sessionID string // for both header and run rows
	runIndex  int    // index into RunsModel.groupedRuns[sessionID]; -1 for headers
}

// RunsModel is the View implementation for tab 4 (Agent Runs).
type RunsModel struct {
	// Raw data populated by SetData.
	agentRuns []memory.AgentRun
	sessions  []memory.Session

	// Grouped data.
	groupedRuns  map[string][]memory.AgentRun // sessionID -> runs (ordered)
	sessionOrder []string                     // session IDs in display order

	// Session metadata for display (id -> session).
	sessionMeta map[string]memory.Session

	// Flat list of navigable items (headers + rows).
	items []runsItem

	// Navigation.
	cursor         int
	expandedGroups map[string]bool

	// Layout.
	width  int
	height int

	hasData bool
}

// NewRuns creates an empty RunsModel.
func NewRuns() *RunsModel {
	return &RunsModel{
		expandedGroups: make(map[string]bool),
		groupedRuns:    make(map[string][]memory.AgentRun),
		sessionMeta:    make(map[string]memory.Session),
	}
}

// Init satisfies the View interface. No background commands needed.
func (m *RunsModel) Init() tea.Cmd {
	return nil
}

// IsInputFocused satisfies View. Runs has no text input; always returns false.
func (m *RunsModel) IsInputFocused() bool { return false }

// Update satisfies the View interface and handles keyboard navigation.
func (m *RunsModel) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g", "home":
			m.cursor = 0
		case "G", "end":
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
			}
		case "ctrl+d":
			// Half-page down: use ~half of the visible list height.
			half := m.listVisibleRows() / 2
			if half < 1 {
				half = 1
			}
			m.cursor += half
			if m.cursor >= len(m.items) {
				m.cursor = len(m.items) - 1
			}
		case "ctrl+u":
			// Half-page up.
			half := m.listVisibleRows() / 2
			if half < 1 {
				half = 1
			}
			m.cursor -= half
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "ctrl+f":
			// Full-page down.
			page := m.listVisibleRows()
			if page < 1 {
				page = 1
			}
			m.cursor += page
			if m.cursor >= len(m.items) {
				m.cursor = len(m.items) - 1
			}
		case "ctrl+b":
			// Full-page up.
			page := m.listVisibleRows()
			if page < 1 {
				page = 1
			}
			m.cursor -= page
			if m.cursor < 0 {
				m.cursor = 0
			}
		case " ", "enter":
			// Toggle expand/collapse for session header items.
			if m.cursor >= 0 && m.cursor < len(m.items) {
				item := m.items[m.cursor]
				if item.isHeader {
					m.expandedGroups[item.sessionID] = !m.expandedGroups[item.sessionID]
					m.rebuildItems()
				}
			}
		case "H":
			// Jump to top of visible area.
			visibleRows := m.listVisibleRows()
			start := 0
			if m.cursor >= visibleRows {
				start = m.cursor - visibleRows + 1
			}
			m.cursor = start
		case "M":
			// Jump to middle of visible area.
			visibleRows := m.listVisibleRows()
			start := 0
			if m.cursor >= visibleRows {
				start = m.cursor - visibleRows + 1
			}
			visible := min(len(m.items)-start, visibleRows)
			m.cursor = start + visible/2
		case "L":
			// Jump to bottom of visible area.
			visibleRows := m.listVisibleRows()
			start := 0
			if m.cursor >= visibleRows {
				start = m.cursor - visibleRows + 1
			}
			visible := min(len(m.items)-start, visibleRows)
			if visible > 0 {
				m.cursor = start + visible - 1
			}
		}
	}
	return m, nil
}

// SetSize satisfies the View interface. Called on every terminal resize.
func (m *RunsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetData satisfies the View interface. Called after every poll cycle.
func (m *RunsModel) SetData(d *data.LoadedData) {
	if d == nil {
		m.hasData = false
		return
	}
	m.hasData = true
	m.agentRuns = d.AgentRuns
	m.sessions = d.Sessions

	// Build session metadata map.
	m.sessionMeta = make(map[string]memory.Session, len(d.Sessions))
	for _, s := range d.Sessions {
		m.sessionMeta[s.ID] = s
	}

	// Group runs by session ID.
	m.groupedRuns = make(map[string][]memory.AgentRun)
	for _, run := range d.AgentRuns {
		key := run.SessionID
		if key == "" {
			key = unattachedSessionID
		}
		m.groupedRuns[key] = append(m.groupedRuns[key], run)
	}

	// Build session order: known sessions first (by started_at DESC), then unattached.
	seen := make(map[string]bool)
	m.sessionOrder = make([]string, 0, len(d.Sessions)+1)
	for _, s := range d.Sessions {
		if _, hasRuns := m.groupedRuns[s.ID]; hasRuns {
			m.sessionOrder = append(m.sessionOrder, s.ID)
			seen[s.ID] = true
		}
	}
	// Add any session IDs referenced by runs but not in sessions list.
	for _, run := range d.AgentRuns {
		key := run.SessionID
		if key == "" {
			continue
		}
		if !seen[key] {
			m.sessionOrder = append(m.sessionOrder, key)
			seen[key] = true
		}
	}
	// Unattached last.
	if _, hasUnattached := m.groupedRuns[unattachedSessionID]; hasUnattached {
		m.sessionOrder = append(m.sessionOrder, unattachedSessionID)
	}

	// Default: expand all groups that have runs.
	for _, sid := range m.sessionOrder {
		if _, alreadySet := m.expandedGroups[sid]; !alreadySet {
			m.expandedGroups[sid] = true
		}
	}

	m.rebuildItems()

	// Clamp cursor.
	if m.cursor >= len(m.items) {
		m.cursor = max(0, len(m.items)-1)
	}
}

// listVisibleRows returns the number of visible rows in the runs list pane.
// This mirrors the calculation in View() for the list pane height.
func (m *RunsModel) listVisibleRows() int {
	const detailMinH = 7
	listHeight := m.height * 60 / 100
	if listHeight < 9 {
		listHeight = 9
	}
	if m.height-listHeight < detailMinH {
		listHeight = m.height - detailMinH
		if listHeight < 5 {
			listHeight = 5
		}
	}
	// Subtract pane borders and header rows (title + blank + header = 3 rows).
	visibleLines := listHeight - 2 - 3
	if visibleLines < 1 {
		visibleLines = 1
	}
	return visibleLines
}

// rebuildItems rebuilds the flat navigable item list from current groupedRuns,
// sessionOrder, and expandedGroups state.
func (m *RunsModel) rebuildItems() {
	m.items = m.items[:0]
	for _, sid := range m.sessionOrder {
		runs := m.groupedRuns[sid]
		if len(runs) == 0 {
			continue
		}
		m.items = append(m.items, runsItem{isHeader: true, sessionID: sid, runIndex: -1})
		if m.expandedGroups[sid] {
			for i := range runs {
				m.items = append(m.items, runsItem{isHeader: false, sessionID: sid, runIndex: i})
			}
		}
	}
}

// View satisfies the View interface and renders the full content area.
func (m *RunsModel) View() string {
	if m.width == 0 || m.height == 0 {
		return MutedStyle.Render("loading…")
	}

	if !m.hasData || len(m.agentRuns) == 0 {
		msg := EmptyStateStyle.Render("No agent runs recorded")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	// Split height: list area on top (~60%), detail panel below (~40%).
	// Minimum: 5 visible rows for list, 3 content rows for detail.
	// Detail pane = border(2) + title(1) + blank(1) + 3 content = 7 minimum.
	const detailMinH = 7
	listHeight := m.height * 60 / 100
	if listHeight < 5+4 { // 5 rows + title + blank + header + border
		listHeight = 9
	}
	if m.height-listHeight < detailMinH {
		listHeight = m.height - detailMinH
		if listHeight < 5 {
			listHeight = 5
		}
	}

	const paneBorderH = 4 // border (1) + padding (1) on each side
	innerW := m.width - paneBorderH
	if innerW < 20 {
		innerW = 20
	}

	listPane := m.renderListPane(innerW, listHeight-2) // subtract pane borders
	detailPane := m.renderDetailPane(innerW)

	return lipgloss.JoinVertical(lipgloss.Left,
		PaneStyle.Width(innerW).Height(listHeight-2).Render(listPane),
		PaneStyle.Width(innerW).Render(detailPane),
	)
}

// renderListPane builds the grouped, scrollable run list.
func (m *RunsModel) renderListPane(innerW, visibleLines int) string {
	title := TitleStyle.Render("Agent Runs")

	// Column widths within the run rows.
	// TYPE(14) STATUS(12) DURATION(10) SUMMARY(rest)
	const typeW, statusW, durationW = 14, 12, 10
	fixedW := typeW + statusW + durationW + 3
	summaryW := innerW - fixedW
	if summaryW < 10 {
		summaryW = 10
	}

	header := "  " + TableHeaderStyle.Render(
		padRight("TYPE", typeW)+
			padRight("STATUS", statusW)+
			padRight("DURATION", durationW)+
			"SUMMARY",
	)

	// Determine visible window for scrolling.
	if visibleLines < 1 {
		visibleLines = 1
	}

	// Build all rows first, then window.
	var allRows []string
	for i, item := range m.items {
		selected := i == m.cursor
		row := m.renderItem(item, selected, typeW, statusW, durationW, summaryW)
		allRows = append(allRows, row)
	}

	// Scroll window: keep cursor visible.
	start := 0
	if m.cursor >= visibleLines {
		start = m.cursor - visibleLines + 1
	}
	end := start + visibleLines
	if end > len(allRows) {
		end = len(allRows)
	}
	if start > end {
		start = end
	}

	visible := allRows[start:end]

	lines := append([]string{title, "", header}, visible...)
	return strings.Join(lines, "\n")
}

// renderItem renders a single item (header or run row) in the list.
func (m *RunsModel) renderItem(item runsItem, selected bool, typeW, statusW, durationW, summaryW int) string {
	if item.isHeader {
		return m.renderSessionHeader(item.sessionID, selected)
	}
	return m.renderRunRow(item.sessionID, item.runIndex, selected, typeW, statusW, durationW, summaryW)
}

// renderSessionHeader renders a collapsible session group header.
// A gutter ► indicator is prepended when selected, two spaces otherwise.
func (m *RunsModel) renderSessionHeader(sessionID string, selected bool) string {
	expanded := m.expandedGroups[sessionID]
	chevron := "▶"
	if expanded {
		chevron = "▼"
	}

	label := m.sessionLabel(sessionID)
	runs := m.groupedRuns[sessionID]
	count := fmt.Sprintf("(%d)", len(runs))

	// Use a plain text format for the selected style to avoid ANSI conflicts.
	text := fmt.Sprintf("%s Session: %s %s",
		chevron,
		label,
		SubtleStyle.Render(count),
	)

	if selected {
		return SubtitleStyle.Render("►") + " " + TableSelectedStyle.Render(text)
	}
	return "  " + SessionBadge.Render(text)
}

// renderRunRow renders one agent run row.
// A gutter ► indicator is prepended when selected, two spaces otherwise.
func (m *RunsModel) renderRunRow(sessionID string, idx int, selected bool, typeW, statusW, durationW, summaryW int) string {
	run := m.groupedRuns[sessionID][idx]

	typeCol := padRight(run.AgentType, typeW)
	statusText := run.ResultStatus
	if statusText == "" {
		statusText = "pending"
	}
	statusStyled := ResultStatusBadge(run.ResultStatus).Render(statusText)
	statusCol := padRight(statusStyled, statusW)
	durationCol := padRight(SubtleStyle.Render(formatDurationMs(run.DurationMs)), durationW)
	summaryCol := truncate(run.ResultSummary, summaryW)

	prefix := "    " // 4 spaces: 2-char gutter + 2-char indent under session header
	row := prefix + typeCol + statusCol + durationCol + summaryCol

	if selected {
		return SubtitleStyle.Render("►") + "   " + TableSelectedStyle.Render(typeCol+statusCol+durationCol+summaryCol)
	}
	return TableRowStyle.Render(row)
}

// renderDetailPane renders the bottom detail panel for the currently selected run.
func (m *RunsModel) renderDetailPane(innerW int) string {
	title := TitleStyle.Render("Run Detail")

	run := m.selectedRun()
	if run == nil {
		return strings.Join([]string{title, "", EmptyStateStyle.Render("Select a run to see details")}, "\n")
	}

	statusText := run.ResultStatus
	if statusText == "" {
		statusText = "pending"
	}

	agentLine := LabelStyle.Render("Agent:") + " " + AgentRunBadge.Render(run.AgentType) +
		"    " + LabelStyle.Render("Status:") + " " + ResultStatusBadge(run.ResultStatus).Render(statusText) +
		"    " + LabelStyle.Render("Duration:") + " " + SubtleStyle.Render(formatDurationMs(run.DurationMs))

	promptLine := LabelStyle.Render("Prompt:") + " " + truncate(run.PromptSummary, innerW-10)
	resultLine := LabelStyle.Render("Result:") + " " + truncate(run.ResultSummary, innerW-10)
	artifactsLine := LabelStyle.Render("Artifacts:") + " " + truncate(run.Artifacts, innerW-13)

	lines := []string{title, "", agentLine, promptLine, resultLine, artifactsLine}
	return strings.Join(lines, "\n")
}

// selectedRun returns the AgentRun for the current cursor position, or nil
// if the cursor is on a header or there are no items.
func (m *RunsModel) selectedRun() *memory.AgentRun {
	if len(m.items) == 0 || m.cursor < 0 || m.cursor >= len(m.items) {
		return nil
	}
	item := m.items[m.cursor]
	if item.isHeader {
		return nil
	}
	runs := m.groupedRuns[item.sessionID]
	if item.runIndex < 0 || item.runIndex >= len(runs) {
		return nil
	}
	r := runs[item.runIndex]
	return &r
}

// CursorPosition satisfies CursorPositioner so the status bar can show [cursor/total].
func (m *RunsModel) CursorPosition() (cursor, total int) {
	return m.cursor, len(m.items)
}

// sessionLabel returns a display label for the given session ID.
// Format: "<change-name or id-prefix> (YYYY-MM-DD)" for known sessions,
// or "Unattached" for the synthetic group.
func (m *RunsModel) sessionLabel(sessionID string) string {
	if sessionID == unattachedSessionID {
		return "Unattached"
	}
	if s, ok := m.sessionMeta[sessionID]; ok {
		name := s.ChangeName
		if name == "" {
			name = s.Goal
		}
		if name == "" {
			// Fallback to first 12 chars of the session ID.
			if len(sessionID) > 12 {
				name = sessionID[:12]
			} else {
				name = sessionID
			}
		}
		date := ""
		if len(s.StartedAt) >= 10 {
			date = " (" + s.StartedAt[:10] + ")"
		}
		return name + date
	}
	// Unknown session — show truncated ID.
	if len(sessionID) > 12 {
		return sessionID[:12]
	}
	return sessionID
}

