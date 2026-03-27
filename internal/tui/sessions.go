// Package tui implements the cx dashboard TUI using Bubble Tea.
// sessions.go is the Sessions Timeline view (tab 3).
// It renders a scrollable list of sessions with a detail panel below.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AngelMaldonado/cx/internal/memory"
	"github.com/AngelMaldonado/cx/internal/tui/data"
)

// SessionsModel is the Bubble Tea sub-model for the Sessions Timeline view.
// It satisfies the View interface declared in app.go.
type SessionsModel struct {
	sessions []memory.Session
	cursor   int
	offset   int
	width    int
	height   int
}

// NewSessionsModel creates a new SessionsModel ready for registration in AppModel.
func NewSessionsModel() *SessionsModel {
	return &SessionsModel{}
}

// Init satisfies the View interface. Sessions has no background commands of its own.
func (m *SessionsModel) Init() tea.Cmd {
	return nil
}

// Update satisfies the View interface.
// Handles j/k and arrow key navigation for the session list.
func (m *SessionsModel) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.sessions)-1 {
				m.cursor++
				// Scroll list if cursor moves below visible area.
				if m.cursor >= m.offset+m.listHeight() {
					m.offset = m.cursor - m.listHeight() + 1
				}
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				// Scroll list if cursor moves above visible area.
				if m.cursor < m.offset {
					m.offset = m.cursor
				}
			}
		case "g", "home":
			m.cursor = 0
			m.offset = 0
		case "G", "end":
			if len(m.sessions) > 0 {
				m.cursor = len(m.sessions) - 1
				lh := m.listHeight()
				if m.cursor >= lh {
					m.offset = m.cursor - lh + 1
				}
			}
		case "ctrl+d":
			// Half-page down.
			half := m.listHeight() / 2
			if half < 1 {
				half = 1
			}
			m.cursor += half
			if m.cursor >= len(m.sessions) {
				m.cursor = len(m.sessions) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
			lh := m.listHeight()
			if m.cursor >= m.offset+lh {
				m.offset = m.cursor - lh + 1
			}
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		case "ctrl+u":
			// Half-page up.
			half := m.listHeight() / 2
			if half < 1 {
				half = 1
			}
			m.cursor -= half
			if m.cursor < 0 {
				m.cursor = 0
			}
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		case "ctrl+f":
			// Full-page down.
			page := m.listHeight()
			if page < 1 {
				page = 1
			}
			m.cursor += page
			if m.cursor >= len(m.sessions) {
				m.cursor = len(m.sessions) - 1
			}
			if m.cursor < 0 {
				m.cursor = 0
			}
			lh := m.listHeight()
			if m.cursor >= m.offset+lh {
				m.offset = m.cursor - lh + 1
			}
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		case "ctrl+b":
			// Full-page up.
			page := m.listHeight()
			if page < 1 {
				page = 1
			}
			m.cursor -= page
			if m.cursor < 0 {
				m.cursor = 0
			}
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		case "H":
			// Jump to top of visible area.
			m.cursor = m.offset
		case "M":
			// Jump to middle of visible area.
			visible := min(len(m.sessions)-m.offset, m.listHeight())
			m.cursor = m.offset + visible/2
		case "L":
			// Jump to bottom of visible area.
			visible := min(len(m.sessions)-m.offset, m.listHeight())
			if visible > 0 {
				m.cursor = m.offset + visible - 1
			}
		}
	}
	return m, nil
}

// SetSize satisfies the View interface. AppModel calls this on every resize.
func (m *SessionsModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	// Clamp cursor into visible range after resize.
	lh := m.listHeight()
	if m.offset > 0 && m.cursor < m.offset {
		m.offset = m.cursor
	}
	if lh > 0 && m.cursor >= m.offset+lh {
		m.offset = m.cursor - lh + 1
	}
}

// SetData satisfies the View interface. AppModel calls this after every poll.
func (m *SessionsModel) SetData(d *data.LoadedData) {
	if d == nil {
		m.sessions = nil
		return
	}
	m.sessions = d.Sessions
	// Clamp cursor to new slice length.
	if m.cursor >= len(m.sessions) {
		if len(m.sessions) == 0 {
			m.cursor = 0
		} else {
			m.cursor = len(m.sessions) - 1
		}
		m.offset = 0
	}
}

// IsInputFocused satisfies View. Sessions has no text input; always returns false.
func (m *SessionsModel) IsInputFocused() bool { return false }

// CursorPosition satisfies CursorPositioner so the status bar can show [cursor/total].
func (m *SessionsModel) CursorPosition() (cursor, total int) {
	return m.cursor, len(m.sessions)
}

// View satisfies the View interface. Renders the full sessions content area.
func (m *SessionsModel) View() string {
	if m.width == 0 || m.height == 0 {
		return MutedStyle.Render("loading…")
	}

	if len(m.sessions) == 0 {
		msg := EmptyStateStyle.Render("No sessions recorded yet.")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	listPanel := m.renderList()
	detailPanel := m.renderDetail()

	return lipgloss.JoinVertical(lipgloss.Left, listPanel, detailPanel)
}

// listHeight returns the number of visible rows available for the session list.
// The layout splits the content area: ~60% for the list, remainder for detail.
// Minimums: 5 rows for list, 3 rows for detail (plus their borders/headers).
func (m *SessionsModel) listHeight() int {
	const paneBorderV = 2  // top+bottom border rows per pane
	const headerRows = 2   // title row + blank row above column headers
	const colHeaderRow = 1 // column header row
	const minListRows = 5  // minimum visible session rows
	const minDetailRows = 3 // minimum visible detail rows (content, excl border/title)
	// Detail pane total = border(2) + title(1) + blank(1) + minDetailRows
	const detailMinHeight = paneBorderV + 2 + minDetailRows

	listPaneH := m.height*6/10 - paneBorderV
	if listPaneH < minListRows {
		listPaneH = minListRows
	}
	// Available rows = pane inner height minus header rows.
	available := listPaneH - headerRows - colHeaderRow
	if available < 1 {
		available = 1
	}
	// Ensure the remaining height accommodates the detail panel.
	if m.height-listPaneH-paneBorderV < detailMinHeight {
		listPaneH = m.height - detailMinHeight - paneBorderV
		if listPaneH < minListRows {
			listPaneH = minListRows
		}
		available = listPaneH - headerRows - colHeaderRow
		if available < 1 {
			available = 1
		}
	}
	return available
}

// renderList builds the top session list pane.
func (m *SessionsModel) renderList() string {
	const paneBorderH = 4 // border(1) + padding(1) on each side = 4
	innerW := m.width - paneBorderH
	if innerW < 20 {
		innerW = 20
	}

	title := TitleStyle.Render("Sessions")

	// Column widths.
	// MODE(10)  CHANGE(22)  GOAL(rest)  STARTED(17)
	modeW, changeW, startedW := 10, 22, 17
	if innerW < 56 {
		startedW = 0 // hide the started column entirely on narrow terminals
		changeW = min(changeW, max(innerW/3, 8))
	}
	separators := 3
	if startedW == 0 {
		separators = 2
	}
	fixedW := modeW + changeW + startedW + separators
	goalW := innerW - fixedW
	if goalW < 10 {
		goalW = 10
	}

	headerStr := padRight("MODE", modeW) +
		padRight("CHANGE", changeW) +
		padRight("GOAL", goalW)
	if startedW > 0 {
		headerStr += "STARTED"
	}
	header := TableHeaderStyle.Render(headerStr)

	lh := m.listHeight()
	end := m.offset + lh
	if end > len(m.sessions) {
		end = len(m.sessions)
	}
	visible := m.sessions[m.offset:end]

	var rowStrings []string
	for i, s := range visible {
		absIdx := m.offset + i
		modeCol := padRight(renderModeTag(s.Mode), modeW)
		changeCol := padRight(truncate(s.ChangeName, changeW-1), changeW)
		goalCol := padRight(truncate(s.Goal, goalW-1), goalW)

		var line string
		if absIdx == m.cursor {
			// Selected row: apply TableSelectedStyle to the raw text for consistent width.
			// Re-compose without mode badge color to avoid ANSI conflicts.
			cursorMark := SubtitleStyle.Render("►") + " "
			rawRow := padRight(strings.ToUpper(s.Mode), modeW) +
				padRight(truncate(s.ChangeName, changeW-1), changeW) +
				padRight(truncate(s.Goal, goalW-1), goalW)
			if startedW > 0 {
				rawRow += formatTimestamp(s.StartedAt)
			}
			line = cursorMark + TableSelectedStyle.Render(rawRow)
		} else {
			row := modeCol + changeCol + goalCol
			if startedW > 0 {
				row += SubtleStyle.Render(formatTimestamp(s.StartedAt))
			}
			line = "  " + TableRowStyle.Render(row)
		}
		rowStrings = append(rowStrings, line)
	}

	countNote := ""
	if len(m.sessions) > 0 {
		countNote = SubtleStyle.Render(fmt.Sprintf("  %d sessions", len(m.sessions)))
	}

	rows := append(
		[]string{title, countNote, "", header},
		rowStrings...,
	)
	content := strings.Join(rows, "\n")

	listPaneH := m.height * 6 / 10
	if listPaneH < 7 { // 5 list rows + 2 header/blank lines = 7 inner, plus borders = 9
		listPaneH = 7
	}
	return PaneStyle.Width(innerW).Height(listPaneH - 2).Render(content)
}

// renderDetail builds the bottom detail panel for the selected session.
func (m *SessionsModel) renderDetail() string {
	const paneBorderH = 4
	innerW := m.width - paneBorderH
	if innerW < 20 {
		innerW = 20
	}

	if len(m.sessions) == 0 || m.cursor >= len(m.sessions) {
		return PaneStyle.Width(innerW).Render(MutedStyle.Render("No session selected"))
	}

	s := m.sessions[m.cursor]

	title := TitleStyle.Render("Session Detail")

	startedLine := LabelStyle.Render("Started:") + " " + SubtleStyle.Render(formatTimestamp(s.StartedAt))
	endedPart := ""
	if s.EndedAt != "" {
		endedPart = "    " + LabelStyle.Render("Ended:") + " " + SubtleStyle.Render(formatTimestamp(s.EndedAt))
	}
	timeLine := startedLine + endedPart

	modeLine := LabelStyle.Render("Mode:") + " " + renderModeTag(s.Mode) +
		"    " + LabelStyle.Render("Change:") + " " + ValueStyle.Render(s.ChangeName)

	goalLine := LabelStyle.Render("Goal:") + " " + ValueStyle.Render(truncate(s.Goal, innerW-8))

	var summaryLines []string
	if s.Summary != "" {
		summaryLines = append(summaryLines, "")
		summaryLines = append(summaryLines, LabelStyle.Render("Summary:"))
		// Wrap summary text to innerW.
		wrapped := wrapText(s.Summary, innerW-2)
		for _, ln := range wrapped {
			summaryLines = append(summaryLines, "  "+ValueStyle.Render(ln))
		}
	}

	lines := []string{title, "", goalLine, modeLine, timeLine}
	lines = append(lines, summaryLines...)
	content := strings.Join(lines, "\n")

	detailH := m.height - m.height*6/10
	if detailH < 5 { // 3 content rows + title + blank = 5 inner minimum
		detailH = 5
	}
	return PaneStyle.Width(innerW).Height(detailH - 2).Render(content)
}

// renderModeTag returns a colored mode badge string for the given mode value.
// BUILD=ColorSuccess (green), PLAN=ColorSelected (blue), CONTINUE=ColorWarning (orange).
func renderModeTag(mode string) string {
	upper := strings.ToUpper(mode)
	switch strings.ToLower(mode) {
	case "build":
		return lipgloss.NewStyle().Foreground(ColorSuccess).Bold(true).Render(upper)
	case "plan":
		return lipgloss.NewStyle().Foreground(ColorSelected).Bold(true).Render(upper)
	case "continue":
		return lipgloss.NewStyle().Foreground(ColorWarning).Bold(true).Render(upper)
	default:
		return MutedStyle.Render(upper)
	}
}

