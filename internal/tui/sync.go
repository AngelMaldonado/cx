// Package tui implements the cx dashboard TUI using Bubble Tea.
// sync.go is the Sync Status view (tab 5) showing push/pull state and
// the list of memories pending export.
package tui

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AngelMaldonado/cx/internal/memory"
	"github.com/AngelMaldonado/cx/internal/tui/data"
)

// pullConflictRe matches lines emitted by "cx memory pull" for each conflict.
// Format: "conflict: <id> — local and shared versions differ"
var pullConflictRe = regexp.MustCompile(`(?im)^conflict:`)

// syncOpResultMsg is returned when a push or pull shell-out completes.
type syncOpResultMsg struct {
	action  string // "push" or "pull"
	output  string // combined stdout+stderr
	success bool
}

// refreshAfterSyncMsg signals AppModel to reload all data after a sync operation.
type refreshAfterSyncMsg struct{}

// SyncModel is the Bubble Tea sub-model for the Sync Status view (tab 5).
// It satisfies the View interface declared in app.go.
type SyncModel struct {
	// allMemories holds the full memory set from the latest poll.
	allMemories []memory.Memory

	// pendingExport contains memories with visibility=project and shared_at=="".
	pendingExport []memory.Memory

	// List navigation for the pending export table.
	cursor int
	offset int

	// Layout dimensions.
	width  int
	height int

	// Operation state.
	pushing bool
	pulling bool

	// Result messages from the last push/pull operations.
	lastPushResult *string
	lastPullResult *string
	lastError      *string

	// Whether cx is found in PATH (checked once at construction).
	cxAvailable bool
	// Resolved path to cx binary (stored to avoid repeated PATH lookups).
	cxPath string
}

// NewSyncModel creates a SyncModel ready to be registered in AppModel.
func NewSyncModel() *SyncModel {
	m := &SyncModel{}
	if path, err := exec.LookPath("cx"); err == nil {
		m.cxAvailable = true
		m.cxPath = path
	}
	return m
}

// ---- View interface ---------------------------------------------------------

// Init satisfies View. Sync has no background commands of its own.
func (m *SyncModel) Init() tea.Cmd { return nil }

// SetSize satisfies View. AppModel calls this on every resize.
func (m *SyncModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetData satisfies View. Called after every poll cycle.
func (m *SyncModel) SetData(d *data.LoadedData) {
	if d == nil {
		m.allMemories = nil
		m.pendingExport = nil
		return
	}
	m.allMemories = d.Memories
	m.rebuildPending()
	m.clampCursor()
}

// IsInputFocused satisfies View. Sync has no text input; always returns false.
func (m *SyncModel) IsInputFocused() bool { return false }

// Update satisfies View.
func (m *SyncModel) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {

	case syncOpResultMsg:
		return m.handleSyncResult(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// View satisfies View. Renders the Sync Status content area.
func (m *SyncModel) View() string {
	if m.width == 0 || m.height == 0 {
		return MutedStyle.Render("Sync (loading…)")
	}

	summaryPanel := m.renderSummaryPanel()
	pendingPanel := m.renderPendingPanel()
	hintBar := m.renderHintBar()

	return lipgloss.JoinVertical(lipgloss.Left, summaryPanel, pendingPanel, hintBar)
}

// ---- Key handling -----------------------------------------------------------

func (m *SyncModel) handleKey(msg tea.KeyMsg) (View, tea.Cmd) {
	switch msg.String() {

	case "j", "down":
		if m.cursor < len(m.pendingExport)-1 {
			m.cursor++
			listH := m.tableHeight()
			if m.cursor >= m.offset+listH {
				m.offset = m.cursor - listH + 1
			}
		}

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}

	case "p":
		if !m.pushing && !m.pulling && m.cxAvailable {
			m.pushing = true
			m.lastError = nil
			return m, m.runSyncCmd("push")
		}

	case "P":
		if !m.pushing && !m.pulling && m.cxAvailable {
			m.pushing = true
			m.lastError = nil
			return m, m.runSyncCmd("push", "--all")
		}

	case "u":
		if !m.pushing && !m.pulling && m.cxAvailable {
			m.pulling = true
			m.lastError = nil
			return m, m.runSyncCmd("pull")
		}
	}

	return m, nil
}

// handleSyncResult processes a completed push or pull operation.
func (m *SyncModel) handleSyncResult(msg syncOpResultMsg) (View, tea.Cmd) {
	result := strings.TrimSpace(msg.output)
	if result == "" {
		if msg.success {
			result = msg.action + " completed"
		} else {
			result = msg.action + " failed"
		}
	}

	switch msg.action {
	case "push":
		m.pushing = false
		m.lastPushResult = &result
	case "pull":
		m.pulling = false
		m.lastPullResult = &result
	}

	if !msg.success {
		errMsg := msg.action + " failed: " + result
		m.lastError = &errMsg
	}

	// Trigger a data refresh so pending list re-builds after push.
	return m, func() tea.Msg { return refreshAfterSyncMsg{} }
}

// ---- Data helpers -----------------------------------------------------------

// rebuildPending filters allMemories for project-visible, not-yet-shared entries.
func (m *SyncModel) rebuildPending() {
	m.pendingExport = nil
	for _, mem := range m.allMemories {
		if mem.Visibility == "project" && mem.SharedAt == "" {
			m.pendingExport = append(m.pendingExport, mem)
		}
	}
}

// clampCursor ensures cursor and offset are within bounds of pendingExport.
func (m *SyncModel) clampCursor() {
	if len(m.pendingExport) == 0 {
		m.cursor = 0
		m.offset = 0
		return
	}
	if m.cursor >= len(m.pendingExport) {
		m.cursor = len(m.pendingExport) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	tableH := m.tableHeight()
	if m.offset > m.cursor {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+tableH {
		m.offset = m.cursor - tableH + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// lastSharedAt returns the most recent shared_at timestamp across all memories,
// or "" if no memories have been shared yet.
func (m *SyncModel) lastSharedAt() string {
	latest := ""
	for _, mem := range m.allMemories {
		if mem.SharedAt != "" && mem.SharedAt > latest {
			latest = mem.SharedAt
		}
	}
	return latest
}

// ---- Shell-out command ------------------------------------------------------

// runSyncCmd returns a Cmd that shells out to "cx memory <action> [args...]".
// It uses the resolved cxPath stored at construction time to avoid repeated PATH lookups.
func (m *SyncModel) runSyncCmd(action string, args ...string) tea.Cmd {
	cxPath := m.cxPath
	fullArgs := append([]string{"memory", action}, args...)
	return func() tea.Msg {
		out, err := exec.Command(cxPath, fullArgs...).CombinedOutput()
		return syncOpResultMsg{
			action:  action,
			output:  string(out),
			success: err == nil,
		}
	}
}

// ---- Rendering --------------------------------------------------------------

// summaryPanelHeight is the fixed number of lines the summary panel occupies
// (including its border).
const summaryPanelHeight = 9

// tableHeight returns the number of visible rows available for the pending table.
func (m *SyncModel) tableHeight() int {
	// Subtract: summary panel, hint bar (1), table header (2), table borders (2).
	h := m.height - summaryPanelHeight - 5
	if h < 1 {
		h = 1
	}
	return h
}

// renderSummaryPanel renders the top push/pull summary section.
// Below 80 columns, push and pull panels are stacked vertically.
func (m *SyncModel) renderSummaryPanel() string {
	const paneBorderH = 4 // border (1) + padding (1) on each side

	var topRow string
	if m.width < homeStackThreshold {
		// Stack vertically on narrow terminals.
		innerW := m.width - paneBorderH
		if innerW < 10 {
			innerW = 10
		}
		pushPanel := m.renderPushPanel(innerW)
		pullPanel := m.renderPullPanel(innerW)
		topRow = lipgloss.JoinVertical(lipgloss.Left, pushPanel, pullPanel)
	} else {
		halfW := (m.width - 1) / 2
		innerW := halfW - paneBorderH
		if innerW < 10 {
			innerW = 10
		}
		pushPanel := m.renderPushPanel(innerW)
		pullPanel := m.renderPullPanel(innerW)
		topRow = lipgloss.JoinHorizontal(lipgloss.Top, pushPanel, " ", pullPanel)
	}

	// Result / error message below the two panels.
	var msgLine string
	if m.lastError != nil {
		msgLine = ErrorStyle.Render(truncate(*m.lastError, m.width-2))
	} else {
		var parts []string
		if m.lastPushResult != nil {
			parts = append(parts, SubtleStyle.Render("push: ")+ValueStyle.Render(*m.lastPushResult))
		}
		if m.lastPullResult != nil {
			parts = append(parts, SubtleStyle.Render("pull: ")+ValueStyle.Render(*m.lastPullResult))
		}
		if len(parts) > 0 {
			msgLine = strings.Join(parts, "   ")
		}
	}

	if msgLine != "" {
		return lipgloss.JoinVertical(lipgloss.Left, topRow, msgLine)
	}
	return topRow
}

// renderPushPanel renders the left push-status pane.
func (m *SyncModel) renderPushPanel(innerW int) string {
	title := TitleStyle.Render("Push Status")
	divider := MutedStyle.Render(strings.Repeat("─", innerW))

	pendingCount := len(m.pendingExport)
	pendingLabel := LabelStyle.Render("Pending export:")
	pendingValue := fmt.Sprintf("%d memories", pendingCount)
	if pendingCount == 0 {
		pendingValue = SuccessBadge.Render("up to date")
	} else {
		pendingValue = SubtitleStyle.Render(pendingValue)
	}

	lastPush := m.lastSharedAt()
	lastPushLabel := LabelStyle.Render("Last push:")
	lastPushValue := MutedStyle.Render("never")
	if lastPush != "" {
		lastPushValue = SubtleStyle.Render(formatTimestamp(lastPush))
	}

	var statusLine string
	if m.pushing {
		statusLine = SubtitleStyle.Render("pushing…")
	} else if !m.cxAvailable {
		statusLine = MutedStyle.Render("cx not in PATH")
	}

	rows := []string{
		title,
		divider,
		pendingLabel + " " + pendingValue,
		lastPushLabel + " " + lastPushValue,
	}
	if statusLine != "" {
		rows = append(rows, statusLine)
	}

	content := strings.Join(rows, "\n")
	return PaneStyle.Width(innerW).Render(content)
}

// renderPullPanel renders the right pull-status pane.
func (m *SyncModel) renderPullPanel(innerW int) string {
	title := TitleStyle.Render("Pull Status")
	divider := MutedStyle.Render(strings.Repeat("─", innerW))

	// Count conflicts from the last pull output.
	// "cx memory pull" emits one "conflict: <id> — ..." line per conflict.
	// pullConflictRe matches these lines; len(matches) is the conflict count.
	conflicts := 0
	if m.lastPullResult != nil {
		matches := pullConflictRe.FindAllString(*m.lastPullResult, -1)
		if len(matches) > 0 {
			conflicts = len(matches)
		}
	}

	conflictLabel := LabelStyle.Render("Conflicts:")
	var conflictValue string
	if conflicts > 0 {
		conflictValue = PendingBadge.Render(fmt.Sprintf("%d", conflicts))
	} else {
		conflictValue = SubtleStyle.Render("0")
	}

	lastPullLabel := LabelStyle.Render("Last pull:")
	lastPullValue := MutedStyle.Render("never")
	if m.lastPullResult != nil {
		lastPullValue = SubtleStyle.Render("just now")
	}

	var statusLine string
	if m.pulling {
		statusLine = SubtitleStyle.Render("pulling…")
	} else if !m.cxAvailable {
		statusLine = MutedStyle.Render("cx not in PATH")
	}

	rows := []string{
		title,
		divider,
		lastPullLabel + " " + lastPullValue,
		conflictLabel + " " + conflictValue,
	}
	if statusLine != "" {
		rows = append(rows, statusLine)
	}

	content := strings.Join(rows, "\n")
	return PaneStyle.Width(innerW).Render(content)
}

// renderPendingPanel renders the full-width table of pending-export memories.
func (m *SyncModel) renderPendingPanel() string {
	const paneBorderH = 4

	innerW := m.width - paneBorderH
	if innerW < 20 {
		innerW = 20
	}

	title := TitleStyle.Render(fmt.Sprintf("Pending Exports  (%d)", len(m.pendingExport)))

	if len(m.pendingExport) == 0 {
		content := strings.Join([]string{
			title,
			"",
			EmptyStateStyle.Render("Nothing pending — all project memories have been pushed."),
		}, "\n")
		return PaneStyle.Width(innerW).Render(content)
	}

	// Column widths: TYPE(14) TITLE(rest) CREATED(20)
	const typeW = 14
	const createdW = 20
	const spacing = 2
	titleW := innerW - typeW - createdW - spacing*2
	if titleW < 10 {
		titleW = 10
	}

	header := TableHeaderStyle.Render(
		padRight("TYPE", typeW) +
			strings.Repeat(" ", spacing) +
			padRight("TITLE", titleW) +
			strings.Repeat(" ", spacing) +
			"CREATED",
	)

	tableH := m.tableHeight()
	end := m.offset + tableH
	if end > len(m.pendingExport) {
		end = len(m.pendingExport)
	}

	var rowStrings []string
	for i := m.offset; i < end; i++ {
		mem := m.pendingExport[i]
		selected := i == m.cursor

		typeStr := padRight(mem.EntityType, typeW)
		titleStr := padRight(truncate(mem.Title, titleW), titleW)
		createdStr := formatTimestamp(mem.CreatedAt)

		typeRendered := EntityTypeBadge(mem.EntityType).Render(typeStr)

		var rowStyle lipgloss.Style
		if selected {
			rowStyle = TableSelectedStyle
		} else {
			rowStyle = TableRowStyle
		}

		row := rowStyle.Render(
			typeRendered +
				strings.Repeat(" ", spacing) +
				titleStr +
				strings.Repeat(" ", spacing) +
				createdStr,
		)
		rowStrings = append(rowStrings, row)
	}

	// Pagination indicator.
	var footerLine string
	if len(m.pendingExport) > tableH {
		footerLine = SubtleStyle.Render(fmt.Sprintf(
			"showing %d–%d of %d",
			m.offset+1, end, len(m.pendingExport),
		))
	}

	parts := []string{title, "", header}
	parts = append(parts, rowStrings...)
	if footerLine != "" {
		parts = append(parts, footerLine)
	}

	content := strings.Join(parts, "\n")
	return PaneStyle.Width(innerW).Render(content)
}

// renderHintBar renders the key hint line at the bottom of the view.
func (m *SyncModel) renderHintBar() string {
	hints := []struct{ key, desc string }{
		{"↑↓/jk", "navigate"},
		{"p", "push"},
		{"P", "push --all"},
		{"u", "pull"},
	}

	if !m.cxAvailable {
		noOp := MutedStyle.Render("cx not in PATH — push/pull unavailable")
		return StatusBarStyle.Width(m.width).Render(noOp)
	}

	if m.pushing || m.pulling {
		op := "pushing"
		if m.pulling {
			op = "pulling"
		}
		spinner := SubtitleStyle.Render(op + "…")
		return StatusBarStyle.Width(m.width).Render(spinner)
	}

	var parts []string
	for _, h := range hints {
		parts = append(parts,
			StatusKey(h.key)+" "+h.desc,
		)
	}
	bar := strings.Join(parts, "  ")
	if m.width > 0 && lipgloss.Width(bar) > m.width-2 {
		bar = truncateANSI(bar, m.width-2)
	}
	return StatusBarStyle.Width(m.width).Render(bar)
}
