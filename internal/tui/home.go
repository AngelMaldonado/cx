// Package tui — home.go implements the Home/Overview dashboard view (tab 1).
// It renders three panels: a stats summary (left), the latest session (right),
// and a recent agent runs table (full-width bottom).
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/amald/cx/internal/tui/data"
)

// maxRecentRuns is the maximum number of agent runs shown in the home view table.
const maxRecentRuns = 5

// HomeModel is the View implementation for tab 1 (Home / Overview).
type HomeModel struct {
	width  int
	height int

	// data fields populated by SetData
	stats     statsSnapshot
	sessions  sessionSnapshot
	agentRuns []agentRunRow
	hasData   bool
}

// statsSnapshot holds the values displayed in the stats panel.
type statsSnapshot struct {
	observations int
	decisions    int
	sessions     int
	agentRuns    int
	links        int
}

// sessionSnapshot holds the fields displayed in the latest-session panel.
type sessionSnapshot struct {
	goal      string
	mode      string
	startedAt string
	endedAt   string
	found     bool
}

// agentRunRow holds pre-formatted fields for one row of the agent runs table.
type agentRunRow struct {
	agentType string
	status    string
	duration  string
	summary   string
}

// NewHome creates a new HomeModel ready to be registered in AppModel.
func NewHome() *HomeModel {
	return &HomeModel{}
}

// Init satisfies the View interface. Home has no background commands of its own.
func (m *HomeModel) Init() tea.Cmd {
	return nil
}

// IsInputFocused satisfies View. Home has no text input; always returns false.
func (m *HomeModel) IsInputFocused() bool { return false }

// Update satisfies the View interface.
// Home is currently read-only so no key events are consumed.
func (m *HomeModel) Update(msg tea.Msg) (View, tea.Cmd) {
	return m, nil
}

// SetSize satisfies the View interface. AppModel calls this on every resize.
func (m *HomeModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetData satisfies the View interface. AppModel calls this after every poll.
func (m *HomeModel) SetData(d *data.LoadedData) {
	if d == nil {
		m.hasData = false
		return
	}
	m.hasData = true

	// Stats panel
	m.stats = statsSnapshot{
		observations: d.Stats.TotalObservations,
		decisions:    d.Stats.TotalDecisions,
		sessions:     d.Stats.TotalSessions,
		agentRuns:    d.Stats.TotalAgentRuns,
		links:        d.Stats.TotalLinks,
	}

	// Latest session panel — use the first element of Sessions slice if available.
	if len(d.Sessions) > 0 {
		s := d.Sessions[0]
		m.sessions = sessionSnapshot{
			goal:      s.Goal,
			mode:      strings.ToUpper(s.Mode),
			startedAt: formatTimestamp(s.StartedAt),
			endedAt:   s.EndedAt,
			found:     true,
		}
	} else {
		m.sessions = sessionSnapshot{found: false}
	}

	// Recent agent runs — take the first maxRecentRuns entries.
	runs := d.AgentRuns
	if len(runs) > maxRecentRuns {
		runs = runs[:maxRecentRuns]
	}
	m.agentRuns = make([]agentRunRow, 0, len(runs))
	for _, r := range runs {
		m.agentRuns = append(m.agentRuns, agentRunRow{
			agentType: r.AgentType,
			status:    r.ResultStatus,
			duration:  formatDurationMs(r.DurationMs),
			summary:   r.ResultSummary,
		})
	}
}

// homeStackThreshold is the terminal width below which the two top panels
// stack vertically instead of sitting side-by-side.
const homeStackThreshold = 80

// View satisfies the View interface. It renders the full home content area.
func (m *HomeModel) View() string {
	if m.width == 0 || m.height == 0 {
		return MutedStyle.Render("loading…")
	}

	if !m.hasData {
		msg := EmptyStateStyle.Render("No data yet — run some cx commands to populate")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	const paneBorderH = 4 // border (1) + padding (1) on each side = 4
	const paneBorderV = 2 // border top + bottom = 2

	// Agent runs table gets the full width.
	runsInnerW := m.width - paneBorderH
	if runsInnerW < 20 {
		runsInnerW = 20
	}
	runsPanel := m.renderRunsPanel(runsInnerW)

	if m.width < homeStackThreshold {
		// Narrow terminal: stack stats and session panels vertically.
		panelInnerW := m.width - paneBorderH
		if panelInnerW < 10 {
			panelInnerW = 10
		}
		statsPanel := m.renderStatsPanel(panelInnerW, paneBorderV)
		sessionPanel := m.renderSessionPanel(panelInnerW, paneBorderV)
		return lipgloss.JoinVertical(lipgloss.Left, statsPanel, sessionPanel, runsPanel)
	}

	// Wide terminal: two top panels side by side.
	halfW := (m.width - 1) / 2
	panelInnerW := halfW - paneBorderH
	if panelInnerW < 10 {
		panelInnerW = 10
	}

	statsPanel := m.renderStatsPanel(panelInnerW, paneBorderV)
	sessionPanel := m.renderSessionPanel(panelInnerW, paneBorderV)
	topRow := lipgloss.JoinHorizontal(lipgloss.Top, statsPanel, " ", sessionPanel)

	return lipgloss.JoinVertical(lipgloss.Left, topRow, runsPanel)
}

// renderStatsPanel builds the left stats box.
func (m *HomeModel) renderStatsPanel(innerW, _ int) string {
	title := TitleStyle.Render("Memory Stats")

	rows := []string{
		title,
		"",
		statRow("Observations:", fmt.Sprintf("%d", m.stats.observations), ObservationBadge, innerW),
		statRow("Decisions:", fmt.Sprintf("%d", m.stats.decisions), DecisionBadge, innerW),
		statRow("Sessions:", fmt.Sprintf("%d", m.stats.sessions), SessionBadge, innerW),
		statRow("Agent Runs:", fmt.Sprintf("%d", m.stats.agentRuns), AgentRunBadge, innerW),
		statRow("Links:", fmt.Sprintf("%d", m.stats.links), MutedStyle, innerW),
	}

	content := strings.Join(rows, "\n")
	return PaneStyle.Width(innerW).Render(content)
}

// renderSessionPanel builds the right latest-session box.
func (m *HomeModel) renderSessionPanel(innerW, _ int) string {
	title := TitleStyle.Render("Latest Session")

	var content string
	if !m.sessions.found {
		content = strings.Join([]string{
			title,
			"",
			EmptyStateStyle.Render("No sessions recorded"),
		}, "\n")
	} else {
		status := "active"
		if m.sessions.endedAt != "" {
			status = "ended"
		}

		content = strings.Join([]string{
			title,
			"",
			kvRow("Goal:", truncate(m.sessions.goal, innerW-10), innerW),
			kvRow("Mode:", SessionBadge.Render(m.sessions.mode), innerW),
			kvRow("Started:", SubtleStyle.Render(m.sessions.startedAt), innerW),
			kvRow("Status:", renderStatusLabel(status), innerW),
		}, "\n")
	}

	return PaneStyle.Width(innerW).Render(content)
}

// renderRunsPanel builds the full-width recent agent runs table.
func (m *HomeModel) renderRunsPanel(innerW int) string {
	title := TitleStyle.Render("Recent Agent Runs")

	if len(m.agentRuns) == 0 {
		content := strings.Join([]string{
			title,
			"",
			EmptyStateStyle.Render("No agent runs recorded"),
		}, "\n")
		return PaneStyle.Width(innerW).Render(content)
	}

	// Column widths — distribute available space.
	// On narrow panels, shrink or drop secondary columns.
	typeW := 14
	statusW := 12
	durationW := 10
	if innerW < 60 {
		// Very narrow: drop duration column.
		durationW = 0
		typeW = 12
		statusW = 10
	}
	fixedW := typeW + statusW + durationW + 3 // 3 for spacing between columns
	summaryW := innerW - fixedW
	if summaryW < 10 {
		summaryW = 10
	}

	// Header row — skip DURATION column when it has zero width.
	headerStr := padRight("TYPE", typeW) + padRight("STATUS", statusW)
	if durationW > 0 {
		headerStr += padRight("DURATION", durationW)
	}
	headerStr += "SUMMARY"
	header := TableHeaderStyle.Render(headerStr)

	var rowStrings []string
	for _, r := range m.agentRuns {
		typeCol := padRight(r.agentType, typeW)
		statusCol := padRight(ResultStatusBadge(r.status).Render(r.status), statusW)
		summaryCol := truncate(r.summary, summaryW)

		var rowStr string
		if durationW > 0 {
			durationCol := padRight(SubtleStyle.Render(r.duration), durationW)
			rowStr = typeCol + statusCol + durationCol + summaryCol
		} else {
			rowStr = typeCol + statusCol + summaryCol
		}
		row := TableRowStyle.Render(rowStr)
		rowStrings = append(rowStrings, row)
	}

	content := strings.Join(append([]string{title, "", header}, rowStrings...), "\n")
	return PaneStyle.Width(innerW).Render(content)
}

// --- helpers ---

// statRow renders one label+value line in the stats panel.
// The value is styled with the given badge style.
func statRow(label, value string, badge lipgloss.Style, width int) string {
	styledLabel := LabelStyle.Render(label)
	styledValue := badge.Render(value)
	gap := width - lipgloss.Width(styledLabel) - lipgloss.Width(styledValue)
	if gap < 1 {
		gap = 1
	}
	return styledLabel + strings.Repeat(" ", gap) + styledValue
}

// kvRow renders one key:value line in a panel.
func kvRow(label, value string, _ int) string {
	return LabelStyle.Render(label) + " " + value
}

// renderStatusLabel returns a colored label for session status.
func renderStatusLabel(status string) string {
	switch status {
	case "active":
		return SuccessBadge.Render(status)
	case "ended":
		return MutedStyle.Render(status)
	default:
		return ValueStyle.Render(status)
	}
}

