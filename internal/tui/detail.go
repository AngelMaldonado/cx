// Package tui implements the cx dashboard TUI using Bubble Tea.
// detail.go is a full-screen overlay modal that displays the complete content
// of any entity (memory, session, agent run). It is NOT a tab — it overlays
// the active tab when the user presses enter on a list item.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/amald/cx/internal/memory"
)

// CloseDetailMsg is sent when the user presses esc or q inside the detail
// overlay. AppModel handles this by hiding the detail overlay.
type CloseDetailMsg struct{}

// ShowDeprecateMsg is sent from the detail overlay when the user presses d on
// a memory. AppModel relays this to the active view as a ConfirmDeprecateMsg.
type ShowDeprecateMsg struct {
	ID    string
	Title string
}

// DetailModel is a full-screen overlay for a single entity.
// It is not a tab — it is managed by AppModel as an optional overlay.
// Activate it by calling Show(); dismiss it by pressing esc or q.
type DetailModel struct {
	// Entity data — at most one is non-nil at a time.
	mem      *memory.Memory
	session  *memory.Session
	agentRun *memory.AgentRun

	// Pre-rendered content (glamour markdown).
	rendered string
	// Raw markdown source, re-rendered on resize.
	rawContent string

	// Overlay display title and entity type badge label.
	title      string
	entityType string

	// Scroll state.
	scrollY   int
	maxScroll int

	// Whether this overlay is currently visible.
	active bool

	width  int
	height int
}

// NewDetail creates a DetailModel ready for use as an AppModel overlay.
func NewDetail() *DetailModel {
	return &DetailModel{}
}

// IsVisible reports whether the overlay is currently active.
func (m *DetailModel) IsVisible() bool {
	return m.active
}

// Show makes the overlay visible.
func (m *DetailModel) Show() {
	m.active = true
}

// Hide hides the overlay without clearing its content.
func (m *DetailModel) Hide() {
	m.active = false
}

// SetSize updates the overlay dimensions and recalculates scroll bounds.
// Must be called before Show() if the terminal dimensions are known.
func (m *DetailModel) SetSize(w, h int) {
	m.width = w
	m.height = h
	// Re-render content at the new width if we have content.
	if m.rawContent != "" {
		m.render()
	}
}

// SetMemory configures the detail overlay to show a memory entity.
// Call Show() after to make the overlay visible.
func (m *DetailModel) SetMemory(mem memory.Memory) {
	m.mem = &mem
	m.session = nil
	m.agentRun = nil
	m.title = mem.Title
	m.entityType = mem.EntityType
	m.scrollY = 0
	m.rawContent = buildMemoryContent(mem)
	m.render()
}

// SetSession configures the detail overlay to show a session entity.
func (m *DetailModel) SetSession(s memory.Session) {
	m.mem = nil
	m.session = &s
	m.agentRun = nil
	m.title = s.Goal
	if m.title == "" {
		m.title = s.ID
	}
	m.entityType = "session"
	m.scrollY = 0
	m.rawContent = buildSessionContent(s)
	m.render()
}

// SetAgentRun configures the detail overlay to show an agent run entity.
func (m *DetailModel) SetAgentRun(r memory.AgentRun) {
	m.mem = nil
	m.session = nil
	m.agentRun = &r
	m.title = r.AgentType
	if r.PromptSummary != "" {
		m.title = r.AgentType + ": " + truncate(r.PromptSummary, 50)
	}
	m.entityType = "agent_run"
	m.scrollY = 0
	m.rawContent = buildAgentRunContent(r)
	m.render()
}

// render re-renders rawContent via glamour at the current width and updates
// maxScroll. Called by SetSize and each Set* method.
func (m *DetailModel) render() {
	contentW := m.contentWidth()
	m.rendered = detailRenderMarkdown(m.rawContent, contentW)
	lines := strings.Split(m.rendered, "\n")
	visible := m.contentHeight()
	m.maxScroll = len(lines) - visible
	if m.maxScroll < 0 {
		m.maxScroll = 0
	}
	if m.scrollY > m.maxScroll {
		m.scrollY = m.maxScroll
	}
}

// contentHeight returns the number of content lines available for scrolling.
// Subtracts: title row (1), separator (1), status bar (1), plus 1 for safety.
func (m *DetailModel) contentHeight() int {
	h := m.height - 4
	if h < 1 {
		h = 1
	}
	return h
}

// contentWidth returns the usable width for glamour rendering.
func (m *DetailModel) contentWidth() int {
	w := m.width - 4
	if w < 10 {
		w = 10
	}
	return w
}

// Update handles key events for the detail overlay.
// Does NOT implement the View interface — AppModel calls this directly.
func (m *DetailModel) Update(msg tea.Msg) (*DetailModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.active = false
			return m, func() tea.Msg { return CloseDetailMsg{} }

		case "j", "down":
			if m.scrollY < m.maxScroll {
				m.scrollY++
			}

		case "k", "up":
			if m.scrollY > 0 {
				m.scrollY--
			}

		case "g", "home":
			m.scrollY = 0

		case "G", "end":
			m.scrollY = m.maxScroll

		case "ctrl+d":
			// Half-page down.
			half := m.contentHeight() / 2
			if half < 1 {
				half = 1
			}
			m.scrollY += half
			if m.scrollY > m.maxScroll {
				m.scrollY = m.maxScroll
			}

		case "ctrl+u":
			// Half-page up.
			half := m.contentHeight() / 2
			if half < 1 {
				half = 1
			}
			m.scrollY -= half
			if m.scrollY < 0 {
				m.scrollY = 0
			}

		case "ctrl+f":
			// Full-page down.
			page := m.contentHeight()
			if page < 1 {
				page = 1
			}
			m.scrollY += page
			if m.scrollY > m.maxScroll {
				m.scrollY = m.maxScroll
			}

		case "ctrl+b":
			// Full-page up.
			page := m.contentHeight()
			if page < 1 {
				page = 1
			}
			m.scrollY -= page
			if m.scrollY < 0 {
				m.scrollY = 0
			}

		case "d":
			// Deprecation only makes sense for memories.
			if m.mem != nil {
				id := m.mem.ID
				title := m.mem.Title
				return m, func() tea.Msg {
					return ShowDeprecateMsg{ID: id, Title: title}
				}
			}
		}
	}

	return m, nil
}

// View renders the full-screen overlay. AppModel calls this when active=true.
func (m *DetailModel) View() string {
	if !m.active || m.width == 0 || m.height == 0 {
		return ""
	}

	// ---- Title bar row ----
	badge := EntityTypeBadge(m.entityType).Render("[" + m.entityType + "]")
	titleText := TitleStyle.Render(m.title)

	titleW := lipgloss.Width(titleText)
	badgeW := lipgloss.Width(badge)
	gap := m.width - titleW - badgeW - 4 // 4 for left padding
	if gap < 1 {
		gap = 1
	}
	titleRow := "  " + titleText + strings.Repeat(" ", gap) + badge

	// ---- Separator ----
	sep := MutedStyle.Render(strings.Repeat("─", m.width-2))

	// ---- Scrollable content area ----
	lines := strings.Split(m.rendered, "\n")
	contentH := m.contentHeight()
	end := m.scrollY + contentH
	if end > len(lines) {
		end = len(lines)
	}
	var visible []string
	if m.scrollY < len(lines) {
		visible = lines[m.scrollY:end]
	}
	// Pad to contentH so the status bar stays at the bottom.
	for len(visible) < contentH {
		visible = append(visible, "")
	}
	content := strings.Join(visible, "\n")

	// ---- Status bar row at bottom ----
	hints := buildDetailHints(m.mem != nil, m.scrollY, m.maxScroll)
	statusRow := StatusBarStyle.Width(m.width).Render(hints)

	return strings.Join([]string{titleRow, sep, content, statusRow}, "\n")
}

// buildDetailHints returns the key-hint string for the detail overlay status bar.
func buildDetailHints(isMemory bool, scrollY, maxScroll int) string {
	hints := []string{
		StatusKeyStyle.Render("esc") + StatusValueStyle.Render(" back"),
		StatusKeyStyle.Render("j/k") + StatusValueStyle.Render(" scroll"),
	}
	if isMemory {
		hints = append(hints, StatusKeyStyle.Render("d")+StatusValueStyle.Render(" deprecate"))
	}
	scrollInfo := ""
	if maxScroll > 0 {
		pct := scrollY * 100 / maxScroll
		scrollInfo = SubtleStyle.Render(fmt.Sprintf(" (%d%%)", pct))
	}
	return "  " + strings.Join(hints, "  ") + scrollInfo
}

// detailRenderMarkdown renders content as markdown using glamour's auto style.
// Falls back to plain text on error.
func detailRenderMarkdown(content string, width int) string {
	if content == "" {
		return ""
	}
	r := getRenderer(width)
	if r == nil {
		return content
	}
	rendered, err := r.Render(content)
	if err != nil {
		return content
	}
	return rendered
}

// ---- Content builders ----

// buildMemoryContent assembles the full markdown content for a memory entity.
func buildMemoryContent(m memory.Memory) string {
	var sb strings.Builder

	// Metadata section
	sb.WriteString("## Metadata\n\n")
	writeDetailField(&sb, "Author", m.Author)
	writeDetailField(&sb, "Created", formatTimestamp(m.CreatedAt))
	writeDetailField(&sb, "Updated", formatTimestamp(m.UpdatedAt))
	if m.ChangeID != "" {
		writeDetailField(&sb, "Change", m.ChangeID)
	}
	if m.Source != "" {
		writeDetailField(&sb, "Source", m.Source)
	}
	if m.Tags != "" {
		writeDetailField(&sb, "Tags", m.Tags)
	}
	if m.FileRefs != "" && m.FileRefs != "[]" && m.FileRefs != "null" {
		writeDetailField(&sb, "Files", m.FileRefs)
	}
	if m.SpecRefs != "" && m.SpecRefs != "[]" && m.SpecRefs != "null" {
		writeDetailField(&sb, "Specs", m.SpecRefs)
	}
	if m.Visibility != "" {
		writeDetailField(&sb, "Visibility", m.Visibility)
	}
	if m.SharedAt != "" {
		writeDetailField(&sb, "Shared", formatTimestamp(m.SharedAt))
	}
	if m.Deprecated == 1 {
		writeDetailField(&sb, "Status", "**deprecated**")
	} else if m.Status != "" {
		writeDetailField(&sb, "Status", m.Status)
	}

	sb.WriteString("\n---\n\n")

	// Content section
	if m.Content != "" {
		sb.WriteString(m.Content)
	} else {
		sb.WriteString("*No content.*")
	}

	return sb.String()
}

// buildSessionContent assembles the full content for a session entity.
func buildSessionContent(s memory.Session) string {
	var sb strings.Builder

	sb.WriteString("## Session Details\n\n")
	writeDetailField(&sb, "ID", s.ID)
	writeDetailField(&sb, "Mode", strings.ToUpper(s.Mode))
	if s.ChangeName != "" {
		writeDetailField(&sb, "Change", s.ChangeName)
	}
	writeDetailField(&sb, "Started", formatTimestamp(s.StartedAt))
	if s.EndedAt != "" {
		writeDetailField(&sb, "Ended", formatTimestamp(s.EndedAt))
	}
	if s.Goal != "" {
		writeDetailField(&sb, "Goal", s.Goal)
	}

	sb.WriteString("\n---\n\n")

	if s.Summary != "" {
		sb.WriteString("## Summary\n\n")
		sb.WriteString(s.Summary)
	} else {
		sb.WriteString("*No summary recorded.*")
	}

	return sb.String()
}

// buildAgentRunContent assembles the full content for an agent run entity.
func buildAgentRunContent(r memory.AgentRun) string {
	var sb strings.Builder

	sb.WriteString("## Agent Run Details\n\n")
	writeDetailField(&sb, "ID", r.ID)
	writeDetailField(&sb, "Type", r.AgentType)
	if r.SessionID != "" {
		writeDetailField(&sb, "Session", r.SessionID)
	}
	writeDetailField(&sb, "Status", r.ResultStatus)
	writeDetailField(&sb, "Duration", formatDurationMs(r.DurationMs))
	writeDetailField(&sb, "Created", formatTimestamp(r.CreatedAt))

	if r.PromptSummary != "" {
		sb.WriteString("\n---\n\n")
		sb.WriteString("## Prompt\n\n")
		sb.WriteString(r.PromptSummary)
	}

	sb.WriteString("\n---\n\n")

	if r.ResultSummary != "" {
		sb.WriteString("## Result\n\n")
		sb.WriteString(r.ResultSummary)
	} else {
		sb.WriteString("*No result summary recorded.*")
	}

	if r.Artifacts != "" && r.Artifacts != "[]" && r.Artifacts != "null" {
		sb.WriteString("\n\n## Artifacts\n\n")
		sb.WriteString(fmt.Sprintf("```\n%s\n```\n", r.Artifacts))
	}

	return sb.String()
}

// writeDetailField writes a single "**Label:** value\n\n" line to the builder.
func writeDetailField(sb *strings.Builder, label, value string) {
	if value == "" {
		return
	}
	sb.WriteString(fmt.Sprintf("**%s:** %s\n\n", label, value))
}
