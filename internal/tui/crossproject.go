// Package tui implements the cx dashboard TUI using Bubble Tea.
// crossproject.go is the Cross-Project federated search view (tab 8).
// It searches across all registered projects via the global index DB.
package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AngelMaldonado/cx/internal/memory"
	"github.com/AngelMaldonado/cx/internal/tui/data"
)

// crossProjectSearchResultMsg is returned by the async federated search command.
type crossProjectSearchResultMsg struct {
	results []memory.ProjectMemoryResult
	err     error
}

// projectColorPalette is a fixed set of Catppuccin hex colors for assigning
// distinct colours to different project names in the results list.
// These mid-tone accent colors have sufficient contrast on both dark and light backgrounds.
var projectColorPalette = []lipgloss.Color{
	"#209fb5", // Sapphire
	"#8839ef", // Mauve
	"#40a02b", // Green
	"#fe640b", // Peach
	"#dd7878", // Flamingo
	"#179299", // Teal
	"#df8e1d", // Yellow
	"#7287fd", // Lavender
}

// CrossProjectModel is the Bubble Tea sub-model for the cross-project search view.
// It satisfies the View interface declared in app.go.
type CrossProjectModel struct {
	results     []memory.ProjectMemoryResult
	searchInput textinput.Model
	searchQuery string // query that produced the current results

	hasSearched bool
	searching   bool // async search in flight

	cursor int
	offset int

	// Preview pane.
	previewRendered string
	previewOffset   int
	showPreview     bool // user-toggled preview visibility in single-pane mode

	// Layout dimensions.
	width     int
	height    int
	listWidth int

	// Status / error.
	statusMsg string
	statusErr bool

	// projectColors maps project name → deterministic colour from palette.
	projectColors map[string]lipgloss.Color

	loader *data.Loader
}

// NewCrossProjectModel creates a CrossProjectModel ready to be registered in AppModel.
func NewCrossProjectModel(loader *data.Loader) *CrossProjectModel {
	ti := textinput.New()
	ti.Placeholder = "search all projects…"
	ti.CharLimit = 256

	return &CrossProjectModel{
		searchInput:   ti,
		projectColors: make(map[string]lipgloss.Color),
		loader:        loader,
	}
}

// ---- View interface --------------------------------------------------------

// Init satisfies View.
func (m *CrossProjectModel) Init() tea.Cmd { return nil }

// isSinglePane returns true when the terminal is narrow enough to collapse
// to a single results pane.
func (m *CrossProjectModel) isSinglePane() bool {
	return m.width < singlePaneThreshold
}

// SetSize satisfies View. Recalculates pane widths.
func (m *CrossProjectModel) SetSize(width, height int) {
	m.width = width
	m.height = height
	if width >= singlePaneThreshold {
		lw := width * 55 / 100
		if lw < 30 {
			lw = 30
		}
		m.listWidth = lw
	} else {
		m.listWidth = width
	}
	m.rebuildPreview()
}

// SetData satisfies View. Cross-project search is on-demand only; no poll data.
func (m *CrossProjectModel) SetData(_ *data.LoadedData) {}

// Update satisfies View.
func (m *CrossProjectModel) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {

	case crossProjectSearchResultMsg:
		m.searching = false
		if msg.err != nil {
			m.statusMsg = "Search error: " + msg.err.Error()
			m.statusErr = true
		} else {
			m.statusErr = false
			m.statusMsg = ""
			m.results = msg.results
			m.cursor = 0
			m.offset = 0
			m.assignProjectColors()
			if len(m.results) == 0 {
				m.statusMsg = "No matches found across projects"
			}
		}
		m.clampCursor()
		m.rebuildPreview()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward non-key messages to textinput while search is focused.
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	return m, cmd
}

// View satisfies View. Renders the full cross-project search view.
func (m *CrossProjectModel) View() string {
	if m.width == 0 || m.height == 0 {
		return MutedStyle.Render("Cross-Project Search (loading…)")
	}

	searchBar := m.renderSearchBar()
	body := m.renderBody()
	hintBar := m.renderHintBar()

	if m.isSinglePane() && m.showPreview {
		return searchBar + "\n" + body + "\n" + hintBar
	}
	header := m.renderHeader()
	return searchBar + "\n" + header + "\n" + body + "\n" + hintBar
}

// ---- Key handling ----------------------------------------------------------

func (m *CrossProjectModel) handleKey(msg tea.KeyMsg) (View, tea.Cmd) {
	// While search input is focused, most keys go to the textinput.
	if m.searchInput.Focused() {
		switch msg.String() {
		case "esc":
			m.searchInput.Blur()
			m.searchInput.SetValue("")
			m.searchQuery = ""
			m.results = nil
			m.hasSearched = false
			m.searching = false
			m.statusMsg = ""
			m.statusErr = false
			m.cursor = 0
			m.offset = 0
			m.previewRendered = ""
			m.previewOffset = 0
			return m, nil

		case "enter":
			q := strings.TrimSpace(m.searchInput.Value())
			if q == "" {
				return m, nil
			}
			m.searchQuery = q
			m.hasSearched = true
			m.searching = true
			m.statusMsg = "Searching…"
			m.statusErr = false
			m.searchInput.Blur()
			return m, m.execSearch(q)
		}

		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	switch msg.String() {
	case "esc":
		// Clear search and results.
		m.searchInput.SetValue("")
		m.searchQuery = ""
		m.results = nil
		m.hasSearched = false
		m.searching = false
		m.statusMsg = ""
		m.statusErr = false
		m.cursor = 0
		m.offset = 0
		m.previewRendered = ""
		m.previewOffset = 0
		m.searchInput.Focus()
		return m, nil

	case "enter":
		q := strings.TrimSpace(m.searchInput.Value())
		if q == "" {
			return m, nil
		}
		m.searchQuery = q
		m.hasSearched = true
		m.searching = true
		m.statusMsg = "Searching…"
		m.statusErr = false
		return m, m.execSearch(q)

	case "/":
		// Re-focus search input.
		m.searchInput.Focus()
		return m, nil

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
			m.clampCursor()
			m.rebuildPreview()
		}

	case "down", "j":
		if m.cursor < len(m.results)-1 {
			m.cursor++
			listH := m.listHeight()
			if m.cursor >= m.offset+listH {
				m.offset = m.cursor - listH + 1
			}
			m.clampCursor()
			m.rebuildPreview()
		}

	case "g", "home":
		m.cursor = 0
		m.offset = 0
		m.clampCursor()
		m.rebuildPreview()

	case "G", "end":
		if len(m.results) > 0 {
			m.cursor = len(m.results) - 1
			listH := m.listHeight()
			if m.cursor >= listH {
				m.offset = m.cursor - listH + 1
			}
			m.clampCursor()
			m.rebuildPreview()
		}

	case "ctrl+d":
		// Half-page down.
		half := m.listHeight() / 2
		if half < 1 {
			half = 1
		}
		m.cursor += half
		if m.cursor >= len(m.results) {
			m.cursor = len(m.results) - 1
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
		if m.cursor >= len(m.results) {
			m.cursor = len(m.results) - 1
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

	case "n":
		// Next search result.
		if m.cursor < len(m.results)-1 {
			m.cursor++
			m.clampCursor()
			m.rebuildPreview()
		}

	case "N":
		// Previous search result.
		if m.cursor > 0 {
			m.cursor--
			m.clampCursor()
			m.rebuildPreview()
		}

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
		m.clampCursor()
		m.rebuildPreview()

	case "M":
		// Jump to middle of visible area.
		visible := min(len(m.results)-m.offset, m.listHeight())
		m.cursor = m.offset + visible/2
		m.clampCursor()
		m.rebuildPreview()

	case "L":
		// Jump to bottom of visible area.
		visible := min(len(m.results)-m.offset, m.listHeight())
		if visible > 0 {
			m.cursor = m.offset + visible - 1
		}
		m.clampCursor()
		m.rebuildPreview()

	case "p":
		if m.isSinglePane() {
			m.showPreview = !m.showPreview
		}
	}

	return m, nil
}

// ---- Async search ----------------------------------------------------------

func (m *CrossProjectModel) execSearch(query string) tea.Cmd {
	loader := m.loader
	opts := memory.SearchOpts{Limit: 200}
	return func() tea.Msg {
		results, err := loader.SearchAllProjects(query, opts)
		return crossProjectSearchResultMsg{results: results, err: err}
	}
}

// ---- Project color assignment ----------------------------------------------

// assignProjectColors deterministically maps each project name to a colour from
// the palette. New project names encountered are assigned the next colour in order.
func (m *CrossProjectModel) assignProjectColors() {
	for _, r := range m.results {
		if _, ok := m.projectColors[r.ProjectName]; !ok {
			idx := len(m.projectColors) % len(projectColorPalette)
			m.projectColors[r.ProjectName] = projectColorPalette[idx]
		}
	}
}

// projectColor returns the colour for a project name, allocating one if needed.
func (m *CrossProjectModel) projectColor(name string) lipgloss.Color {
	if c, ok := m.projectColors[name]; ok {
		return c
	}
	idx := len(m.projectColors) % len(projectColorPalette)
	c := projectColorPalette[idx]
	m.projectColors[name] = c
	return c
}

// ---- Preview ---------------------------------------------------------------

// rebuildPreview re-renders the preview pane content for the selected result.
func (m *CrossProjectModel) rebuildPreview() {
	sel := m.selectedResult()
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

func (m *CrossProjectModel) buildPreviewMarkdown(r *memory.ProjectMemoryResult) string {
	var sb strings.Builder
	sb.WriteString("## " + r.Title + "\n\n")

	sb.WriteString("**Project:** " + r.ProjectName + "  \n")
	typeLine := r.EntityType
	if r.Subtype != "" {
		typeLine += " (" + r.Subtype + ")"
	}
	sb.WriteString("**Type:** " + typeLine + "  \n")
	if r.Author != "" {
		sb.WriteString("**Author:** " + r.Author + "  \n")
	}
	if r.ChangeID != "" {
		sb.WriteString("**Change:** " + r.ChangeID + "  \n")
	}
	if r.Tags != "" {
		sb.WriteString("**Tags:** " + r.Tags + "  \n")
	}
	if r.CreatedAt != "" {
		sb.WriteString("**Created:** " + formatTimestamp(r.CreatedAt) + "  \n")
	}
	sb.WriteString(fmt.Sprintf("**Rank:** %.4f  \n", r.Rank))
	if r.Content != "" {
		sb.WriteString("\n---\n\n")
		sb.WriteString(r.Content)
	}
	return sb.String()
}

// ---- Layout helpers --------------------------------------------------------

func (m *CrossProjectModel) listHeight() int {
	// Layout breakdown for two-pane mode (rows must sum to m.height):
	//   searchBar(1) + "\n"(1) + header(2) + "\n"(1) + body + "\n"(1) + hintBar(1)
	//   body = max(list_pane, preview_pane) = max(listH+3, listH+4) = listH+4
	//   header = TableHeaderStyle with BorderBottom = 2 rows (content + underline)
	//   Total: 1+1+2+1+(listH+4)+1+1 = listH+11 = height → listH = height-11
	h := m.height - 11
	if h < 1 {
		h = 1
	}
	return h
}

func (m *CrossProjectModel) previewHeight() int {
	h := m.listHeight()
	if h < 1 {
		h = 1
	}
	return h
}

func (m *CrossProjectModel) previewWidth() int {
	// Left pane outer = listWidth (content) + 2 (borders) + 2 (padding) = listWidth + 4.
	// Right pane outer = pw (content) + 2 (borders) + 2 (padding) = pw + 4.
	// Total: listWidth + 4 + pw + 4 = m.width → pw = m.width - listWidth - 8.
	pw := m.width - m.listWidth - 8
	if pw < 10 {
		pw = 10
	}
	return pw
}

func (m *CrossProjectModel) clampCursor() {
	if len(m.results) == 0 {
		m.cursor = 0
		m.offset = 0
		return
	}
	if m.cursor >= len(m.results) {
		m.cursor = len(m.results) - 1
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
// Returns the 0-based cursor position; app.go adds 1 when displaying.
func (m *CrossProjectModel) CursorPosition() (int, int) {
	return m.cursor, len(m.results)
}

// IsInputFocused satisfies View. Returns true when the search text input is
// focused, so AppModel passes h/l/←/→ through to this view instead of using
// them for global tab navigation.
func (m *CrossProjectModel) IsInputFocused() bool {
	return m.searchInput.Focused()
}

func (m *CrossProjectModel) selectedResult() *memory.ProjectMemoryResult {
	if len(m.results) == 0 || m.cursor >= len(m.results) {
		return nil
	}
	return &m.results[m.cursor]
}

// ---- Sub-renderers ---------------------------------------------------------

// renderSearchBar renders the search input line at the top.
func (m *CrossProjectModel) renderSearchBar() string {
	prefix := "Search: "
	var barStyle lipgloss.Style
	if m.searchInput.Focused() {
		barStyle = FilterActiveStyle
	} else {
		barStyle = SearchStyle
	}
	inputStr := barStyle.Render(prefix + m.searchInput.View())

	statusStr := ""
	if m.statusMsg != "" {
		if m.statusErr {
			statusStr = "  " + ErrorStyle.Render(m.statusMsg)
		} else {
			statusStr = "  " + MutedStyle.Render(m.statusMsg)
		}
	}
	line := inputStr + statusStr
	if m.width > 0 {
		line = truncate(line, m.width)
	}
	return line
}

// renderHeader renders the column header row of the results table.
// Rendered directly with TableHeaderStyle (no PaneStyle wrapper) so it
// occupies exactly 2 terminal rows: content + bottom border line.
func (m *CrossProjectModel) renderHeader() string {
	if m.width == 0 {
		return ""
	}
	// innerW matches the list pane inner width (listWidth minus PaneStyle borders+padding).
	innerW := m.listWidth - 4
	if innerW < 10 {
		innerW = 10
	}

	projectW := 14
	typeW := 14
	rankW := 6
	if innerW < 42 {
		projectW = max(innerW/4, 4)
		typeW = max(innerW/4, 4)
		rankW = max(innerW/8, 3)
	}
	titleW := innerW - projectW - typeW - rankW - 3 // 3 for separating spaces
	if titleW < 5 {
		titleW = 5
	}

	project := padRight("PROJECT", projectW)
	typ := padRight("TYPE", typeW)
	title := padRight("TITLE", titleW)
	rank := padRight("RANK", rankW)

	// Render directly without PaneStyle — TableHeaderStyle has BorderBottom(true)
	// which produces 2 rows (text + underline). PaneStyle would add 2 more rows.
	return TableHeaderStyle.Width(innerW).Render(project + " " + typ + " " + title + " " + rank)
}

// renderBody renders the two-pane list + preview area.
// In single-pane mode, the user can toggle between list and preview with `p`.
func (m *CrossProjectModel) renderBody() string {
	if m.isSinglePane() {
		if m.showPreview {
			return m.renderPreviewFullWidth()
		}
		return m.renderList()
	}
	listPane := m.renderList()
	previewPane := m.renderPreview()
	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, previewPane)
}

// renderList renders the left results list pane.
func (m *CrossProjectModel) renderList() string {
	listH := m.listHeight()
	innerW := m.listWidth - 4

	var sb strings.Builder

	if !m.hasSearched {
		// Empty state: not searched yet.
		empty := EmptyStateStyle.Render("Type a query to search across all registered projects")
		sb.WriteString(empty)
		for i := 1; i < listH; i++ {
			sb.WriteString("\n")
		}
		return PaneStyle.BorderTop(false).Width(innerW).Height(listH + 1).Render(sb.String())
	}

	if m.searching {
		sb.WriteString(MutedStyle.Render("Searching…"))
		for i := 1; i < listH; i++ {
			sb.WriteString("\n")
		}
		return PaneStyle.BorderTop(false).Width(innerW).Height(listH + 1).Render(sb.String())
	}

	if len(m.results) == 0 {
		empty := EmptyStateStyle.Render("No matches found across projects")
		sb.WriteString(empty)
		for i := 1; i < listH; i++ {
			sb.WriteString("\n")
		}
		return PaneStyle.BorderTop(false).Width(innerW).Height(listH + 1).Render(sb.String())
	}

	end := m.offset + listH
	if end > len(m.results) {
		end = len(m.results)
	}
	rendered := 0
	for i := m.offset; i < end; i++ {
		row := m.renderResultRow(m.results[i], i == m.cursor, innerW)
		sb.WriteString(row)
		sb.WriteString("\n")
		rendered++
	}
	// Fill remaining lines.
	for rendered < listH {
		sb.WriteString("\n")
		rendered++
	}

	return PaneStyle.BorderTop(false).Width(innerW).Height(listH + 1).Render(sb.String())
}

// renderResultRow renders a single result row in the list.
func (m *CrossProjectModel) renderResultRow(r memory.ProjectMemoryResult, selected bool, width int) string {
	projectW := 14
	typeW := 14
	rankW := 6
	if width < 42 {
		projectW = max(width/4, 4)
		typeW = max(width/4, 4)
		rankW = max(width/8, 3)
	}
	titleW := width - projectW - typeW - rankW - 3
	if titleW < 5 {
		titleW = 5
	}

	// Project name — colour-coded per project.
	projColor := m.projectColor(r.ProjectName)
	projStyle := lipgloss.NewStyle().Foreground(projColor)
	if selected {
		projStyle = projStyle.Bold(true)
	}
	projectStr := projStyle.Render(padRight(truncate(r.ProjectName, projectW), projectW))

	// Entity type badge.
	badgeStyle := EntityTypeBadge(r.EntityType)
	typeStr := badgeStyle.Render(padRight(truncate(r.EntityType, typeW), typeW))

	// Title.
	titleStr := truncate(r.Title, titleW)
	titleStr = padRight(titleStr, titleW)

	// Rank — format as 0.xx, negated because FTS5 ranks are negative.
	rankVal := -r.Rank
	if math.IsInf(rankVal, 0) || math.IsNaN(rankVal) || rankVal <= 0 {
		rankVal = 0
	}
	rankStr := fmt.Sprintf("%.2f", rankVal)
	if len(rankStr) > rankW {
		rankStr = rankStr[:rankW]
	}

	row := projectStr + " " + typeStr + " " + titleStr + " " + rankStr

	// Selected row background overrides per-column colours; render the row-level
	// style only for non-project columns to avoid flattening the project colour.
	if selected {
		return TableSelectedStyle.Render(row)
	}
	return TableRowStyle.Render(row)
}

// renderPreview renders the right-side preview pane.
func (m *CrossProjectModel) renderPreview() string {
	ph := m.previewHeight()

	var sb strings.Builder

	sel := m.selectedResult()
	titleText := "Preview"
	if sel != nil {
		titleText = sel.Title
	}
	pw := m.previewWidth()
	sb.WriteString(TitleStyle.Render(truncate(titleText, pw)))
	sb.WriteString("\n")

	if sel == nil {
		if !m.hasSearched {
			sb.WriteString(EmptyStateStyle.Render("Search results will appear here"))
		} else {
			sb.WriteString(EmptyStateStyle.Render("No result selected"))
		}
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
	return PreviewStyle.BorderTop(false).Width(rightW).Height(ph + 2).Render(sb.String())
}

// renderPreviewFullWidth renders the preview pane occupying the full view width
// (used in single-pane mode when the user has toggled preview with `p`).
// Layout: searchBar(1) + "\n"(1) + body + "\n"(1) + hintBar(1) = height
//   body = PreviewStyle total = ph+1+2 = ph+3 → body = height-4 → ph = height-7.
func (m *CrossProjectModel) renderPreviewFullWidth() string {
	pw := m.width - 4 // 2 border + 2 padding from PreviewStyle
	if pw < 10 {
		pw = 10
	}
	ph := m.height - 7 // searchBar(1) + 2×"\n"(2) + hintBar(1) + PreviewStyle borders(2) + content header(1)
	if ph < 1 {
		ph = 1
	}

	var sb strings.Builder

	sel := m.selectedResult()
	titleText := "Preview"
	if sel != nil {
		titleText = sel.Title
	}
	sb.WriteString(TitleStyle.Render(truncate(titleText, pw)))
	sb.WriteString("\n")

	if sel == nil {
		if !m.hasSearched {
			sb.WriteString(EmptyStateStyle.Render("Search results will appear here"))
		} else {
			sb.WriteString(EmptyStateStyle.Render("No result selected"))
		}
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
			sb.WriteString(SubtleStyle.Render("↑ more above"))
		}
		if off+ph < total {
			sb.WriteString(SubtleStyle.Render("↓ more below"))
		}
	}

	return PreviewStyle.BorderTop(false).Width(m.width - 4).Height(ph + 1).Render(sb.String())
}

// renderHintBar renders the key hint line at the bottom.
func (m *CrossProjectModel) renderHintBar() string {
	hints := []struct{ key, desc string }{
		{"↑↓/jk", "navigate"},
		{"enter", "search"},
		{"/", "focus search"},
		{"J/K", "scroll preview"},
		{"esc", "clear"},
	}
	if m.isSinglePane() {
		label := "preview"
		if m.showPreview {
			label = "list"
		}
		hints = append(hints, struct{ key, desc string }{"p", label})
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

