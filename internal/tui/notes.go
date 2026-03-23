// Package tui implements the cx dashboard TUI using Bubble Tea.
// notes.go is the Personal Notes browser view (tab 6).
// It provides a read-only two-pane layout: left list of notes, right preview.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AngelMaldonado/cx/internal/memory"
	"github.com/AngelMaldonado/cx/internal/tui/data"
)

// NotesModel is the Bubble Tea sub-model for the Personal Notes browser (tab 6).
// It satisfies the View interface declared in app.go.
type NotesModel struct {
	notes  []memory.PersonalNote
	cursor int
	offset int

	// Preview pane.
	previewRendered string // glamour-rendered markdown cache
	previewOffset   int   // scroll position in lines
	showPreview     bool  // user-toggled preview visibility in single-pane mode

	// Layout dimensions.
	width     int
	height    int
	listWidth int // left pane usable width (≈40% minus borders)
}

// isSinglePane returns true when the terminal is narrow enough to collapse
// to a single list pane.
func (m *NotesModel) isSinglePane() bool {
	return m.width < singlePaneThreshold
}

// NewNotesModel creates a NotesModel ready to be registered in AppModel.
func NewNotesModel() *NotesModel {
	return &NotesModel{}
}

// ---- View interface --------------------------------------------------------

// Init satisfies View. Notes has no background commands of its own.
func (m *NotesModel) Init() tea.Cmd { return nil }

// SetSize satisfies View. Recalculates pane widths.
func (m *NotesModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	if width >= singlePaneThreshold {
		// Two-pane: left pane ≈ 40% of width (minimum 20 columns).
		lw := width * 40 / 100
		if lw < 20 {
			lw = 20
		}
		m.listWidth = lw
	} else {
		// Single-pane: list occupies the full width.
		m.listWidth = width
	}
	// Re-render preview with new width.
	m.rebuildPreview()
}

// SetData satisfies View. Called after every poll cycle.
func (m *NotesModel) SetData(d *data.LoadedData) {
	if d == nil {
		return
	}
	m.notes = d.PersonalNotes
	m.clampCursor()
	m.rebuildPreview()
}

// IsInputFocused satisfies View. Notes has no text input; always returns false.
func (m *NotesModel) IsInputFocused() bool { return false }

// Update satisfies View.
func (m *NotesModel) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

// View satisfies View. Renders the full personal notes browser.
func (m *NotesModel) View() string {
	if m.width == 0 || m.height == 0 {
		return MutedStyle.Render("Notes (loading…)")
	}

	if len(m.notes) == 0 {
		msg := EmptyStateStyle.Render("No personal notes. Add notes with: cx memory save --type note")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	body := m.renderBody()
	hintBar := m.renderHintBar()

	return body + "\n" + hintBar
}

// ---- Key handling ----------------------------------------------------------

func (m *NotesModel) handleKey(msg tea.KeyMsg) (View, tea.Cmd) {
	switch msg.String() {
	case "p":
		if m.isSinglePane() {
			m.showPreview = !m.showPreview
		}

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
			m.rebuildPreview()
		}

	case "down", "j":
		if m.cursor < len(m.notes)-1 {
			m.cursor++
			listH := m.listHeight()
			if m.cursor >= m.offset+listH {
				m.offset = m.cursor - listH + 1
			}
			m.rebuildPreview()
		}

	case "g", "home":
		m.cursor = 0
		m.offset = 0
		m.rebuildPreview()

	case "G", "end":
		if len(m.notes) > 0 {
			m.cursor = len(m.notes) - 1
			listH := m.listHeight()
			if m.cursor >= listH {
				m.offset = m.cursor - listH + 1
			}
			m.rebuildPreview()
		}

	case "ctrl+d":
		// Half-page down.
		half := m.listHeight() / 2
		if half < 1 {
			half = 1
		}
		m.cursor += half
		if m.cursor >= len(m.notes) {
			m.cursor = len(m.notes) - 1
		}
		m.clampCursor()
		m.rebuildPreview()

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
		m.clampCursor()
		m.rebuildPreview()

	case "ctrl+f":
		// Full-page down.
		page := m.listHeight()
		if page < 1 {
			page = 1
		}
		m.cursor += page
		if m.cursor >= len(m.notes) {
			m.cursor = len(m.notes) - 1
		}
		m.clampCursor()
		m.rebuildPreview()

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
		m.clampCursor()
		m.rebuildPreview()

	// Preview scroll (J/K scroll preview pane).
	case "J":
		lines := strings.Split(m.previewRendered, "\n")
		maxOff := len(lines) - m.previewHeight()
		if maxOff > 0 && m.previewOffset < maxOff {
			m.previewOffset++
		}
	case "K":
		if m.previewOffset > 0 {
			m.previewOffset--
		}

	case "H":
		// Jump to top of visible area.
		m.cursor = m.offset
		m.rebuildPreview()

	case "M":
		// Jump to middle of visible area.
		visible := min(len(m.notes)-m.offset, m.listHeight())
		m.cursor = m.offset + visible/2
		m.rebuildPreview()

	case "L":
		// Jump to bottom of visible area.
		visible := min(len(m.notes)-m.offset, m.listHeight())
		if visible > 0 {
			m.cursor = m.offset + visible - 1
		}
		m.rebuildPreview()
	}

	return m, nil
}

// ---- Layout helpers --------------------------------------------------------

// listHeight returns the number of visible rows in the list pane.
// Subtracts 1 for the hint bar and 2 for the pane borders.
func (m *NotesModel) listHeight() int {
	h := m.height - 3 // hint bar + 2 border rows
	if h < 1 {
		h = 1
	}
	return h
}

// previewHeight returns the height of the preview pane content area.
func (m *NotesModel) previewHeight() int {
	h := m.listHeight()
	if h < 1 {
		h = 1
	}
	return h
}

// previewWidth returns the usable width of the right pane.
// Left pane outer = listWidth (content) + 2 (borders) + 2 (padding) = listWidth + 4.
// Right pane outer = pw (content) + 2 (borders) + 2 (padding) = pw + 4.
// Total: listWidth + 4 + pw + 4 = m.width → pw = m.width - listWidth - 8.
func (m *NotesModel) previewWidth() int {
	pw := m.width - m.listWidth - 8 // left-pane outer (4) + right-pane outer overhead (4)
	if pw < 10 {
		pw = 10
	}
	return pw
}

// clampCursor ensures cursor and offset are in bounds.
func (m *NotesModel) clampCursor() {
	if len(m.notes) == 0 {
		m.cursor = 0
		m.offset = 0
		return
	}
	if m.cursor >= len(m.notes) {
		m.cursor = len(m.notes) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	listH := m.listHeight()
	if m.offset > m.cursor {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+listH {
		m.offset = m.cursor - listH + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

// CursorPosition satisfies CursorPositioner so the status bar can show [cursor/total].
func (m *NotesModel) CursorPosition() (cursor, total int) {
	return m.cursor, len(m.notes)
}

// selectedNote returns the currently selected *memory.PersonalNote or nil.
func (m *NotesModel) selectedNote() *memory.PersonalNote {
	if len(m.notes) == 0 || m.cursor >= len(m.notes) {
		return nil
	}
	return &m.notes[m.cursor]
}

// ---- Preview rendering -----------------------------------------------------

// rebuildPreview updates previewRendered for the currently selected note.
func (m *NotesModel) rebuildPreview() {
	sel := m.selectedNote()
	if sel == nil {
		m.previewRendered = ""
		m.previewOffset = 0
		return
	}
	pw := m.previewWidth()
	md := m.buildPreviewMarkdown(sel)
	m.previewRendered = renderGlamour(md, pw)
	m.previewOffset = 0
}

// buildPreviewMarkdown formats a PersonalNote into the markdown shown in
// the preview pane.
func (m *NotesModel) buildPreviewMarkdown(n *memory.PersonalNote) string {
	var sb strings.Builder
	sb.WriteString("## " + n.Title + "\n\n")

	if n.TopicKey != "" {
		sb.WriteString("**Topic:** " + n.TopicKey + "  \n")
	}
	if n.Tags != "" {
		sb.WriteString("**Tags:** " + n.Tags + "  \n")
	}
	if n.Projects != "" {
		sb.WriteString("**Projects:** " + n.Projects + "  \n")
	}
	if n.UpdatedAt != "" {
		sb.WriteString("**Updated:** " + formatTimestamp(n.UpdatedAt) + "  \n")
	}
	if n.Content != "" {
		sb.WriteString("\n---\n\n")
		sb.WriteString(n.Content)
	}

	return sb.String()
}

// ---- Sub-renderers ---------------------------------------------------------

// renderBody renders the two-pane list + preview area.
// In single-pane mode, `p` toggles between list and preview.
func (m *NotesModel) renderBody() string {
	if m.isSinglePane() {
		if m.showPreview {
			return m.renderPreviewFullWidth()
		}
		return m.renderListFullWidth()
	}
	listPane := m.renderList(m.listWidth)
	previewPane := m.renderPreview()
	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, previewPane)
}

// renderListFullWidth renders the list occupying the full terminal width.
func (m *NotesModel) renderListFullWidth() string {
	return m.renderList(m.width)
}

// renderPreviewFullWidth renders the preview occupying the full terminal width.
func (m *NotesModel) renderPreviewFullWidth() string {
	pw := m.width - 4 // border + padding
	if pw < 10 {
		pw = 10
	}
	ph := m.previewHeight()

	var sb strings.Builder

	sel := m.selectedNote()
	titleText := "Preview"
	if sel != nil {
		titleText = sel.Title
	}
	sb.WriteString(TitleStyle.Render(truncate(titleText, pw)))
	sb.WriteString("\n")

	if sel == nil {
		sb.WriteString(EmptyStateStyle.Render("Select a note to preview."))
	} else {
		lines := strings.Split(m.previewRendered, "\n")
		total := len(lines)
		maxOff := total - ph
		if maxOff < 0 {
			maxOff = 0
		}
		off := m.previewOffset
		if off > maxOff {
			off = maxOff
		}
		end := off + ph
		if end > total {
			end = total
		}
		visible := lines[off:end]
		for _, line := range visible {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		for i := len(visible); i < ph; i++ {
			sb.WriteString("\n")
		}
		if off > 0 {
			sb.WriteString(SubtleStyle.Render("↑ more above") + "\n")
		}
		if off+ph < total {
			sb.WriteString(SubtleStyle.Render("↓ more below") + "\n")
		}
	}

	return PreviewStyle.Width(m.width - 2).Height(ph + 1).Render(sb.String())
}

// renderList renders the left pane: bordered box with note rows.
// width is the total pane width (including borders/padding).
func (m *NotesModel) renderList(width int) string {
	listH := m.listHeight()
	innerW := width - 4 // 2 border + 2 padding from PaneStyle

	var sb strings.Builder

	// Header row.
	countStr := fmt.Sprintf("%d personal notes", len(m.notes))
	header := TitleStyle.Render(truncate(countStr, innerW))
	sb.WriteString(header)
	sb.WriteString("\n")

	if len(m.notes) == 0 {
		sb.WriteString(EmptyStateStyle.Render("No personal notes."))
		for i := 1; i < listH; i++ {
			sb.WriteString("\n")
		}
	} else {
		end := m.offset + listH
		if end > len(m.notes) {
			end = len(m.notes)
		}
		rendered := 0
		for i := m.offset; i < end; i++ {
			row := m.renderNoteRow(m.notes[i], i == m.cursor, innerW)
			sb.WriteString(row)
			sb.WriteString("\n")
			rendered++
		}
		// Fill remaining lines.
		for rendered < listH {
			sb.WriteString("\n")
			rendered++
		}
	}

	return PaneStyle.Width(width).Height(m.listHeight() + 1).Render(sb.String())
}

// renderNoteRow renders a single row in the notes list showing topic key and title.
func (m *NotesModel) renderNoteRow(n memory.PersonalNote, selected bool, width int) string {
	// Build "KEY   TITLE" with consistent column widths.
	const keyColW = 14

	key := n.TopicKey
	if key == "" {
		key = SubtleStyle.Render("—")
	} else if len([]rune(key)) > keyColW-1 {
		key = string([]rune(key)[:keyColW-1])
	}
	// Pad key column.
	keyDisplay := padRight(key, keyColW)

	// Title column gets the rest.
	titleW := width - keyColW - 1
	if titleW < 5 {
		titleW = 5
	}
	title := truncate(n.Title, titleW)

	var rowStyle lipgloss.Style
	if selected {
		rowStyle = TableSelectedStyle
		// Prefix selected row with an arrow indicator.
		arrow := SubtitleStyle.Render("► ")
		keyDisplay = arrow + padRight(key, keyColW-2)
	} else {
		rowStyle = TableRowStyle
		keyDisplay = "  " + padRight(key, keyColW-2)
	}

	return rowStyle.Render(keyDisplay + title)
}

// renderPreview renders the right pane: bordered box with glamour-rendered content.
func (m *NotesModel) renderPreview() string {
	pw := m.previewWidth()
	ph := m.previewHeight()

	var sb strings.Builder

	// Header.
	sel := m.selectedNote()
	titleText := "Preview"
	if sel != nil {
		titleText = sel.Title
	}
	sb.WriteString(TitleStyle.Render(truncate(titleText, pw)))
	sb.WriteString("\n")

	if sel == nil {
		sb.WriteString(EmptyStateStyle.Render("Select a note to preview."))
	} else {
		lines := strings.Split(m.previewRendered, "\n")
		total := len(lines)

		maxOff := total - ph
		if maxOff < 0 {
			maxOff = 0
		}
		off := m.previewOffset
		if off > maxOff {
			off = maxOff
		}
		end := off + ph
		if end > total {
			end = total
		}

		visible := lines[off:end]
		for _, line := range visible {
			sb.WriteString(line)
			sb.WriteString("\n")
		}
		// Pad to fill height (reserving a line for scroll indicator if needed).
		padLines := ph - len(visible)
		if padLines < 0 {
			padLines = 0
		}
		scrollIndicator := ""
		if off > 0 && off+ph < total {
			scrollIndicator = SubtleStyle.Render("↑↓ more")
		} else if off > 0 {
			scrollIndicator = SubtleStyle.Render("↑ more above")
		} else if off+ph < total {
			scrollIndicator = SubtleStyle.Render("↓ more below")
		}
		if scrollIndicator != "" && padLines > 0 {
			padLines--
		}
		for i := 0; i < padLines; i++ {
			sb.WriteString("\n")
		}
		if scrollIndicator != "" {
			sb.WriteString(scrollIndicator + "\n")
		}
	}

	rightW := m.previewWidth()
	if rightW < 10 {
		rightW = 10
	}
	return PreviewStyle.Width(rightW).Height(ph + 2).Render(sb.String())
}

// renderHintBar renders the key hint line at the bottom.
func (m *NotesModel) renderHintBar() string {
	hints := []struct{ key, desc string }{
		{"↑↓/jk", "navigate"},
		{"J/K", "scroll preview"},
		{"g/G", "top/bottom"},
	}
	if m.isSinglePane() {
		label := "show preview"
		if m.showPreview {
			label = "show list"
		}
		hints = append(hints, struct{ key, desc string }{"p", label})
	}
	var parts []string
	for _, h := range hints {
		parts = append(parts,
			StatusKeyStyle.Render(h.key)+" "+StatusValueStyle.Render(h.desc),
		)
	}
	bar := strings.Join(parts, "  ")
	if m.width > 0 && lipgloss.Width(bar) > m.width-2 {
		bar = truncateANSI(bar, m.width-2)
	}
	return StatusBarStyle.Width(m.width).Render(bar)
}
