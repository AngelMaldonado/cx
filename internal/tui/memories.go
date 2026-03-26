// Package tui implements the cx dashboard TUI using Bubble Tea.
// memories.go is the Memories browser view (tab 2) with search, filter,
// preview, and deprecation support.
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/AngelMaldonado/cx/internal/memory"
	"github.com/AngelMaldonado/cx/internal/tui/data"
)

// searchResultMsg is returned by the async FTS5 search command.
type searchResultMsg struct {
	results []memory.MemoryResult
	err     error
}

// deprecateResultMsg is returned after a deprecate command completes.
type deprecateResultMsg struct {
	id  string
	err error
}

// memFilterOption is a filter chip for the memory type filter bar.
type memFilterOption struct {
	label  string
	value  string // "observation", "decision", "" = All
	active bool
}

// singlePaneThreshold is the terminal width below which two-pane views
// collapse to a single list pane.
const singlePaneThreshold = 100

// MemoriesModel is the Bubble Tea sub-model for the Memories browser view.
// It satisfies the View interface declared in app.go.
type MemoriesModel struct {
	// Full and filtered memory lists.
	allMemories []memory.Memory // populated from LoadedData on SetData
	memories    []memory.Memory // current display list (filtered/searched)

	// List navigation.
	cursor int
	offset int

	// Search bar.
	searchInput  textinput.Model
	searchActive bool
	searchQuery  string // query that produced the current searchResults
	// When non-nil, memories is derived from searchResults rather than allMemories.
	searchResults []memory.MemoryResult

	// Filter chips (Observations / Decisions / All).
	filters       []memFilterOption
	filterCursor  int
	filterFocused bool

	// Preview pane.
	previewContent  string
	previewRendered string // glamour-rendered markdown cache
	previewOffset   int    // scroll position in lines
	showPreview     bool   // user-toggled preview visibility in single-pane mode

	// Layout dimensions.
	width     int
	height    int
	listWidth int // left pane usable width (≈40% minus borders)

	// Deprecation confirmation dialog.
	showDeprecateConfirm bool
	deprecateID          string
	deprecateTitle       string

	// Status / error for this view.
	statusMsg string
	statusErr bool

	loader *data.Loader
}

// NewMemoriesModel creates a MemoriesModel ready to be registered in AppModel.
func NewMemoriesModel(loader *data.Loader) *MemoriesModel {
	ti := textinput.New()
	ti.Placeholder = "search memories…"
	ti.CharLimit = 256

	filters := []memFilterOption{
		{label: "All", value: "", active: true},
		{label: "Observations", value: "observation", active: false},
		{label: "Decisions", value: "decision", active: false},
		{label: "Interactions", value: "agent_interaction", active: false},
	}

	return &MemoriesModel{
		searchInput: ti,
		filters:     filters,
		loader:      loader,
	}
}

// ---- View interface --------------------------------------------------------

// Init satisfies View.
func (m *MemoriesModel) Init() tea.Cmd { return nil }

// isSinglePane returns true when the terminal is narrow enough that the
// two-pane layout collapses to a single list pane.
func (m *MemoriesModel) isSinglePane() bool {
	return m.width < singlePaneThreshold
}

// SetSize satisfies View. Recalculates pane widths.
func (m *MemoriesModel) SetSize(width, height int) {
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
func (m *MemoriesModel) SetData(d *data.LoadedData) {
	if d == nil {
		return
	}
	m.allMemories = d.Memories
	// If no active search query, rebuild display list from allMemories.
	if m.searchQuery == "" {
		m.applyFilter()
	}
	m.clampCursor()
	m.rebuildPreview()
}

// Update satisfies View.
func (m *MemoriesModel) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {

	// ---- Async results -----------------------------------------------------

	case searchResultMsg:
		if msg.err != nil {
			m.statusMsg = "Search error: " + msg.err.Error()
			m.statusErr = true
		} else {
			m.statusErr = false
			m.searchResults = msg.results
			m.buildMemoriesFromSearch()
		}
		m.clampCursor()
		m.rebuildPreview()
		return m, nil

	case deprecateResultMsg:
		if msg.err != nil {
			m.statusMsg = "Deprecate failed: " + msg.err.Error()
			m.statusErr = true
		} else {
			m.statusMsg = "Memory deprecated."
			m.statusErr = false
			// Mark the memory deprecated in local slice so UI updates immediately.
			for i := range m.allMemories {
				if m.allMemories[i].ID == msg.id {
					m.allMemories[i].Deprecated = 1
				}
			}
			m.applyFilter()
			m.clampCursor()
			m.rebuildPreview()
		}
		return m, nil

	// ---- Keyboard ----------------------------------------------------------

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Forward to textinput when search is active.
	if m.searchActive {
		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View satisfies View. Renders the full memories browser.
func (m *MemoriesModel) View() string {
	if m.width == 0 || m.height == 0 {
		return MutedStyle.Render("Memories (loading…)")
	}

	if m.loader != nil && !m.loader.ProjectReady() {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			EmptyStateStyle.Render("Project not initialized. Run cx init."))
	}

	// Build the three horizontal sections:
	//   1. top bar: search input + filter chips
	//   2. body: two-pane list + preview
	//   3. hint bar: key hints

	topBar := m.renderTopBar()
	body := m.renderBody()
	hintBar := m.renderHintBar()

	// Deprecation confirmation dialog overlaid on top.
	content := topBar + "\n" + body + "\n" + hintBar
	if m.showDeprecateConfirm {
		content = m.renderDeprecateDialog(content)
	}
	return content
}

// ---- Key handling ----------------------------------------------------------

func (m *MemoriesModel) handleKey(msg tea.KeyMsg) (View, tea.Cmd) {
	// While search input is active, most keys go to the textinput.
	if m.searchActive {
		switch msg.String() {
		case "esc":
			m.searchActive = false
			m.searchInput.Blur()
			m.searchInput.SetValue("")
			m.searchQuery = ""
			m.searchResults = nil
			m.applyFilter()
			m.clampCursor()
			m.rebuildPreview()
			return m, nil

		case "enter":
			q := strings.TrimSpace(m.searchInput.Value())
			if q == "" {
				return m, nil
			}
			m.searchQuery = q
			m.searchActive = false
			m.searchInput.Blur()
			return m, m.execSearch(q)
		}

		var cmd tea.Cmd
		m.searchInput, cmd = m.searchInput.Update(msg)
		return m, cmd
	}

	// Deprecate confirmation dialog is open.
	if m.showDeprecateConfirm {
		switch msg.String() {
		case "y", "Y":
			id := m.deprecateID
			m.showDeprecateConfirm = false
			m.deprecateID = ""
			m.deprecateTitle = ""
			return m, m.execDeprecate(id)
		case "n", "N", "esc":
			m.showDeprecateConfirm = false
			m.deprecateID = ""
			m.deprecateTitle = ""
		}
		return m, nil
	}

	// Filter bar is focused.
	if m.filterFocused {
		switch msg.String() {
		case "esc", "f":
			m.filterFocused = false
		case "h", "left":
			if m.filterCursor > 0 {
				m.filterCursor--
			}
		case "l", "right":
			if m.filterCursor < len(m.filters)-1 {
				m.filterCursor++
			}
		case " ", "enter":
			m.toggleFilter(m.filterCursor)
			if m.searchQuery == "" {
				m.applyFilter()
			} else {
				m.buildMemoriesFromSearch()
			}
			m.clampCursor()
			m.rebuildPreview()
		}
		return m, nil
	}

	// Normal navigation mode.
	switch msg.String() {
	case "p":
		// Toggle preview visibility in single-pane mode.
		if m.isSinglePane() {
			m.showPreview = !m.showPreview
		}

	case "/":
		m.searchActive = true
		m.searchInput.Focus()

	case "esc":
		if m.searchQuery != "" {
			m.searchQuery = ""
			m.searchResults = nil
			m.searchInput.SetValue("")
			m.applyFilter()
			m.clampCursor()
			m.rebuildPreview()
		}

	case "f":
		m.filterFocused = true

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
			m.rebuildPreview()
		}

	case "down", "j":
		if m.cursor < len(m.memories)-1 {
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
		if len(m.memories) > 0 {
			m.cursor = len(m.memories) - 1
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
		if m.cursor >= len(m.memories) {
			m.cursor = len(m.memories) - 1
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
		if m.cursor >= len(m.memories) {
			m.cursor = len(m.memories) - 1
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
		// Move to next item (next search result when in search mode).
		if m.cursor < len(m.memories)-1 {
			m.cursor++
			m.clampCursor()
			m.rebuildPreview()
		}

	case "N":
		// Move to previous item (previous search result when in search mode).
		if m.cursor > 0 {
			m.cursor--
			m.clampCursor()
			m.rebuildPreview()
		}

	case "H":
		// Jump to top of visible area.
		m.cursor = m.offset
		m.rebuildPreview()

	case "M":
		// Jump to middle of visible area.
		visible := min(len(m.memories)-m.offset, m.listHeight())
		m.cursor = m.offset + visible/2
		m.rebuildPreview()

	case "L":
		// Jump to bottom of visible area.
		visible := min(len(m.memories)-m.offset, m.listHeight())
		if visible > 0 {
			m.cursor = m.offset + visible - 1
		}
		m.rebuildPreview()

	// Preview scroll (when list is focused, J/K scroll preview).
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

	case "d":
		if sel := m.selectedMemory(); sel != nil {
			m.deprecateID = sel.ID
			m.deprecateTitle = sel.Title
			m.showDeprecateConfirm = true
		}

	case "enter":
		if sel := m.selectedMemory(); sel != nil {
			return m, func() tea.Msg {
				return NavigateToMemoryMsg{ID: sel.ID}
			}
		}
	}

	return m, nil
}

// ---- Filter / search logic -------------------------------------------------

// toggleFilter sets the filter at index i active, deactivating all others
// (radio-button semantics: only one filter active at a time).
func (m *MemoriesModel) toggleFilter(i int) {
	for j := range m.filters {
		m.filters[j].active = (j == i)
	}
}

// activeFilterValue returns the entity_type value of the active filter chip,
// or "" if "All" is selected.
func (m *MemoriesModel) activeFilterValue() string {
	for _, f := range m.filters {
		if f.active && f.value != "" {
			return f.value
		}
	}
	return ""
}

// applyFilter rebuilds m.memories from m.allMemories using the active filter.
// This is a local (in-memory) operation — no DB query.
func (m *MemoriesModel) applyFilter() {
	ft := m.activeFilterValue()
	if ft == "" {
		m.memories = m.allMemories
		return
	}
	var filtered []memory.Memory
	for _, mem := range m.allMemories {
		if mem.EntityType == ft {
			filtered = append(filtered, mem)
		}
	}
	m.memories = filtered
}

// buildMemoriesFromSearch converts m.searchResults → m.memories,
// applying the active type filter on top.
func (m *MemoriesModel) buildMemoriesFromSearch() {
	ft := m.activeFilterValue()
	m.memories = nil
	for _, r := range m.searchResults {
		if ft == "" || r.EntityType == ft {
			m.memories = append(m.memories, r.Memory)
		}
	}
}

// execSearch returns a Cmd that runs SearchMemories asynchronously.
func (m *MemoriesModel) execSearch(query string) tea.Cmd {
	loader := m.loader
	opts := memory.SearchOpts{Limit: 200}
	return func() tea.Msg {
		results, err := loader.SearchMemories(query, opts)
		return searchResultMsg{results: results, err: err}
	}
}

// execDeprecate returns a Cmd that calls DeprecateMemory asynchronously.
func (m *MemoriesModel) execDeprecate(id string) tea.Cmd {
	loader := m.loader
	return func() tea.Msg {
		err := loader.DeprecateMemory(id)
		return deprecateResultMsg{id: id, err: err}
	}
}

// ---- Layout helpers --------------------------------------------------------

// listHeight returns the number of visible rows in the list pane.
// Layout breakdown (rows must sum to m.height):
//   topBar(1) + "\n"(1) + body + "\n"(1) + hintBar(1) = height
//   body = PaneStyle bordered pane = listHeight+3 (list-only) or listHeight+4 (two-pane with preview)
//   Worst case (two-pane): 1+1+(listH+4)+1+1 = listH+8 = height → listH = height-8
func (m *MemoriesModel) listHeight() int {
	h := m.height - 8 // 4 separator/bar rows + 4 pane-overhead rows (2 borders + 1 list-header + 1 preview-extra)
	if h < 1 {
		h = 1
	}
	return h
}

// previewHeight returns the height of the preview pane content area.
func (m *MemoriesModel) previewHeight() int {
	h := m.listHeight()
	if h < 1 {
		h = 1
	}
	return h
}

// previewWidth returns the usable width of the right pane.
// Left pane outer = listWidth (content) + 2 (borders) + 2 (padding) = listWidth + 4.
// Right pane outer = rightW (content) + 2 (borders) + 2 (padding) = rightW + 4.
// Total: listWidth + 4 + rightW + 4 = m.width → rightW = m.width - listWidth - 8.
func (m *MemoriesModel) previewWidth() int {
	pw := m.width - m.listWidth - 8 // left-pane outer (4) + right-pane outer overhead (4)
	if pw < 10 {
		pw = 10
	}
	return pw
}

// clampCursor ensures cursor and offset are in bounds.
func (m *MemoriesModel) clampCursor() {
	if len(m.memories) == 0 {
		m.cursor = 0
		m.offset = 0
		return
	}
	if m.cursor >= len(m.memories) {
		m.cursor = len(m.memories) - 1
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

// selectedMemory returns the currently selected *memory.Memory or nil.
func (m *MemoriesModel) selectedMemory() *memory.Memory {
	if len(m.memories) == 0 || m.cursor >= len(m.memories) {
		return nil
	}
	return &m.memories[m.cursor]
}

// CursorPosition satisfies CursorPositioner so the status bar can show [cursor/total].
func (m *MemoriesModel) CursorPosition() (cursor, total int) {
	return m.cursor, len(m.memories)
}

// IsInputFocused satisfies View. Returns true when the search text input or
// the filter chip bar is focused, so AppModel passes h/l/←/→ through to this
// view instead of using them for global tab navigation.
func (m *MemoriesModel) IsInputFocused() bool {
	return m.searchActive || m.filterFocused
}

// ---- Preview rendering -----------------------------------------------------

// rebuildPreview updates previewContent and re-renders it with glamour.
func (m *MemoriesModel) rebuildPreview() {
	sel := m.selectedMemory()
	if sel == nil {
		m.previewContent = ""
		m.previewRendered = ""
		m.previewOffset = 0
		return
	}

	pw := m.previewWidth()
	m.previewContent = m.buildPreviewMarkdown(sel)
	m.previewRendered = renderGlamour(m.previewContent, pw)
	m.previewOffset = 0
}

// buildPreviewMarkdown formats a memory into the markdown shown in the preview pane.
func (m *MemoriesModel) buildPreviewMarkdown(mem *memory.Memory) string {
	var sb strings.Builder
	sb.WriteString("## " + mem.Title + "\n\n")

	typeLine := mem.EntityType
	if mem.Subtype != "" {
		typeLine += " (" + mem.Subtype + ")"
	}
	sb.WriteString("**Type:** " + typeLine + "  \n")
	if mem.Author != "" {
		sb.WriteString("**Author:** " + mem.Author + "  \n")
	}
	if mem.ChangeID != "" {
		sb.WriteString("**Change:** " + mem.ChangeID + "  \n")
	}
	if mem.Tags != "" {
		sb.WriteString("**Tags:** " + mem.Tags + "  \n")
	}
	if mem.CreatedAt != "" {
		sb.WriteString("**Created:** " + formatTimestamp(mem.CreatedAt) + "  \n")
	}
	if mem.Deprecated == 1 {
		sb.WriteString("\n> **[DEPRECATED]**\n")
	}
	if mem.Content != "" {
		sb.WriteString("\n---\n\n")
		sb.WriteString(mem.Content)
	}

	return sb.String()
}

// ---- Sub-renderers ---------------------------------------------------------

// renderTopBar renders the search input and filter chips row.
func (m *MemoriesModel) renderTopBar() string {
	// Search input segment.
	var searchStr string
	if m.searchActive {
		searchStr = FilterActiveStyle.Render("Search: " + m.searchInput.View())
	} else if m.searchQuery != "" {
		searchStr = FilterActiveStyle.Render("Search: " + m.searchQuery)
	} else {
		searchStr = SearchStyle.Render("/ to search")
	}

	// Filter chips.
	filterStr := m.renderFilterChips()

	// Status message (right-aligned).
	statusStr := ""
	if m.statusMsg != "" {
		if m.statusErr {
			statusStr = ErrorStyle.Render(m.statusMsg)
		} else {
			statusStr = SubtleStyle.Render(m.statusMsg)
		}
	}

	// Compose: search | filter chips | status
	sep := MutedStyle.Render("  ")
	bar := searchStr + sep + filterStr
	if statusStr != "" {
		bar += sep + statusStr
	}

	// Wrap in a top-bar style line, truncated to terminal width.
	if m.width > 0 {
		bar = truncate(bar, m.width)
	}
	return bar
}

// renderFilterChips renders the filter chip bar.
// Chips use no border so they stay on a single terminal row (matching the
// search bar height of 1 row). Active chip uses bold + accent colour;
// focused chip uses the selected colour.
func (m *MemoriesModel) renderFilterChips() string {
	var parts []string
	for i, f := range m.filters {
		var chipStyle lipgloss.Style
		if f.active {
			chipStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true).
				Padding(0, 1)
		} else {
			chipStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 1)
		}
		if m.filterFocused && i == m.filterCursor {
			chipStyle = chipStyle.Foreground(ColorSelected)
		}
		parts = append(parts, chipStyle.Render(f.label))
	}
	return strings.Join(parts, " ")
}

// renderBody renders the two-pane list + preview area.
// In single-pane mode (narrow terminal), the user can toggle between list and
// preview with the `p` key.
func (m *MemoriesModel) renderBody() string {
	if m.isSinglePane() {
		if m.showPreview {
			return m.renderPreviewFullWidth()
		}
		return m.renderListFullWidth()
	}
	listPane := m.renderList(m.listWidth)
	previewPane := m.renderPreview()

	// Join panes side by side.
	return lipgloss.JoinHorizontal(lipgloss.Top, listPane, previewPane)
}

// renderList renders the left pane: bordered box with memory rows.
// width is the total pane width (including borders/padding).
func (m *MemoriesModel) renderList(width int) string {
	listH := m.listHeight()
	innerW := width - 4 // 2 border + 2 padding from PaneStyle

	var sb strings.Builder

	// Header row.
	countStr := fmt.Sprintf("%d memories", len(m.memories))
	if m.searchQuery != "" {
		countStr = fmt.Sprintf("%d results for %q", len(m.memories), m.searchQuery)
	}
	header := TitleStyle.Render(truncate(countStr, innerW))
	sb.WriteString(header)
	sb.WriteString("\n")

	if len(m.memories) == 0 {
		emptyMsg := "No memories found."
		if m.searchQuery != "" {
			emptyMsg = "No results for \"" + m.searchQuery + "\"."
		}
		empty := EmptyStateStyle.Render(emptyMsg)
		// Pad to fill height.
		sb.WriteString(empty)
		for i := 1; i < listH; i++ {
			sb.WriteString("\n")
		}
	} else {
		end := m.offset + listH
		if end > len(m.memories) {
			end = len(m.memories)
		}
		rendered := 0
		for i := m.offset; i < end; i++ {
			row := m.renderMemoryRow(m.memories[i], i == m.cursor, innerW)
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

// renderMemoryRow renders a single row in the memory list.
// A gutter column is prepended: ► (accent) when selected, two spaces otherwise.
// This matches the pattern used in sessions.go and notes.go.
func (m *MemoriesModel) renderMemoryRow(mem memory.Memory, selected bool, width int) string {
	// Bullet: ● active, ◌ deprecated.
	bullet := "●"
	if mem.Deprecated == 1 {
		bullet = "◌"
	}

	// Badge color by entity type.
	badgeStyle := EntityTypeBadge(mem.EntityType)
	badge := badgeStyle.Render(bullet)

	// Gutter: ► indicator for the selected row, two spaces otherwise.
	const gutterW = 2 // "► " or "  "
	var gutter string
	if selected {
		gutter = SubtitleStyle.Render("►") + " "
	} else {
		gutter = "  "
	}

	// Title truncated to fit: subtract gutter(2) + bullet(1) + space(1) + margin(1).
	titleWidth := width - gutterW - 3 // gutterW=2, then bullet+space+margin=3
	if titleWidth < 5 {
		titleWidth = 5
	}
	title := truncate(mem.Title, titleWidth)

	var rowStyle lipgloss.Style
	if mem.Deprecated == 1 {
		rowStyle = DeprecatedRowStyle
	} else if selected {
		rowStyle = TableSelectedStyle
	} else {
		rowStyle = TableRowStyle
	}

	return gutter + rowStyle.Render(badge + " " + title)
}

// renderPreview renders the right pane: bordered box with glamour-rendered content.
func (m *MemoriesModel) renderPreview() string {
	pw := m.previewWidth()
	ph := m.previewHeight()

	var sb strings.Builder

	// Header.
	sel := m.selectedMemory()
	titleText := "Preview"
	if sel != nil {
		titleText = sel.Title
	}
	sb.WriteString(TitleStyle.Render(truncate(titleText, pw)))
	sb.WriteString("\n")

	if sel == nil {
		empty := EmptyStateStyle.Render("Select a memory to preview.")
		sb.WriteString(empty)
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
		// Pad to fill height (leave last line for scroll indicator if needed).
		padLines := ph - len(visible)
		if padLines < 0 {
			padLines = 0
		}
		// Reserve one line for scroll indicator if content overflows.
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

// renderListFullWidth renders the list occupying the entire view width
// (used in single-pane mode).
func (m *MemoriesModel) renderListFullWidth() string {
	return m.renderList(m.width)
}

// renderPreviewFullWidth renders the preview occupying the entire view width
// (used in single-pane mode when user has toggled preview).
func (m *MemoriesModel) renderPreviewFullWidth() string {
	pw := m.width - 4 // 2 border + 2 padding from PreviewStyle
	if pw < 10 {
		pw = 10
	}
	ph := m.previewHeight()

	var sb strings.Builder

	sel := m.selectedMemory()
	titleText := "Preview"
	if sel != nil {
		titleText = sel.Title
	}
	sb.WriteString(TitleStyle.Render(truncate(titleText, pw)))
	sb.WriteString("\n")

	if sel == nil {
		sb.WriteString(EmptyStateStyle.Render("Select a memory to preview."))
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

		// Scroll indicators.
		if off > 0 {
			sb.WriteString(SubtleStyle.Render("↑ more above"))
		}
		if off+ph < total {
			sb.WriteString(SubtleStyle.Render("↓ more below"))
		}
	}

	return PreviewStyle.Width(m.width - 2).Height(ph + 1).Render(sb.String())
}

// renderHintBar renders the key hint line at the bottom.
func (m *MemoriesModel) renderHintBar() string {
	hints := []struct{ key, desc string }{
		{"↑↓/jk", "navigate"},
		{"/", "search"},
		{"f", "filter"},
		{"d", "deprecate"},
		{"enter", "detail"},
		{"esc", "clear"},
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
	// Truncate if the hint bar is too wide.
	if m.width > 0 && lipgloss.Width(bar) > m.width-2 {
		bar = truncateANSI(bar, m.width-2)
	}
	return StatusBarStyle.Width(m.width).Render(bar)
}

// renderDeprecateDialog overlays a confirmation dialog on content.
func (m *MemoriesModel) renderDeprecateDialog(background string) string {
	dialogContent := strings.Join([]string{
		DialogTitleStyle.Render("Deprecate memory?"),
		"",
		MutedStyle.Render(truncate(m.deprecateTitle, 60)),
		"",
		HelpKeyStyle.Render("y") + MutedStyle.Render(" confirm  ") +
			HelpKeyStyle.Render("n/esc") + MutedStyle.Render(" cancel"),
	}, "\n")
	box := DialogStyle.Render(dialogContent)

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}
	return background + "\n" + box
}

