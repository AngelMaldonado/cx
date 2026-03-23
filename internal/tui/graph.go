// Package tui — graph.go implements the Memory Graph view (tab 7).
// It visualizes memory link relationships as a tree-style adjacency list.
package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/amald/cx/internal/memory"
	"github.com/amald/cx/internal/tui/data"
)

// graphItem is one row in the flat navigable list used internally by GraphModel.
// Each item is either a memory root node or a link child node.
type graphItem struct {
	isMemory bool             // true → this is a memory root node
	isLink   bool             // true → this is a link child node
	mem      *memory.Memory   // set when isMemory=true; also set on link rows for the "from" memory
	link     *memory.MemoryLink // set when isLink=true
	target   *memory.Memory   // the linked memory (the other end of the link); may be nil if not loaded
	depth    int              // 0 = root memory, 1 = link child
}

// GraphModel is the Bubble Tea sub-model for the Memory Graph view (tab 7).
// It satisfies the View interface declared in app.go.
type GraphModel struct {
	memories []memory.Memory
	links    []memory.MemoryLink

	// adjacency maps memory ID → links where that memory is the "from" side.
	adjacency map[string][]memory.MemoryLink

	// reverseAdj maps memory ID → links where that memory is the "to" side.
	reverseAdj map[string][]memory.MemoryLink

	// memoryIndex maps memory ID → Memory for title lookups.
	memoryIndex map[string]memory.Memory

	// items is the flat list used for rendering and navigation.
	items []graphItem

	cursor int
	offset int

	width  int
	height int
}

// relationColors maps relation types to lipgloss colors.
var relationColors = map[string]lipgloss.Color{
	"related-to":  lipgloss.Color("33"),  // blue
	"caused-by":   lipgloss.Color("196"), // red
	"resolved-by": lipgloss.Color("82"),  // green
	"see-also":    lipgloss.Color("241"), // gray
}

// relationStyle returns the appropriate lipgloss style for a relation type.
func relationStyle(relType string) lipgloss.Style {
	if c, ok := relationColors[relType]; ok {
		return lipgloss.NewStyle().Foreground(c).Bold(true)
	}
	return MutedStyle
}

// NewGraphModel creates a new GraphModel ready to be registered in AppModel.
func NewGraphModel() *GraphModel {
	return &GraphModel{}
}

// Init satisfies the View interface.
func (m *GraphModel) Init() tea.Cmd {
	return nil
}

// IsInputFocused satisfies View. Graph has no text input; always returns false.
func (m *GraphModel) IsInputFocused() bool { return false }

// Update satisfies the View interface. Handles j/k navigation and enter to navigate to a memory.
func (m *GraphModel) Update(msg tea.Msg) (View, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.cursor < len(m.items)-1 {
				m.cursor++
				m.clampOffset()
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.clampOffset()
			}
		case "g", "home":
			m.cursor = 0
			m.offset = 0
		case "G", "end":
			if len(m.items) > 0 {
				m.cursor = len(m.items) - 1
				m.clampOffset()
			}
		case "ctrl+d":
			// Half-page down.
			const overhead = 3
			visibleRows := m.height - overhead
			if visibleRows < 1 {
				visibleRows = 1
			}
			half := visibleRows / 2
			if half < 1 {
				half = 1
			}
			m.cursor += half
			if m.cursor >= len(m.items) {
				m.cursor = len(m.items) - 1
			}
			m.clampOffset()
		case "ctrl+u":
			// Half-page up.
			const overhead = 3
			visibleRows := m.height - overhead
			if visibleRows < 1 {
				visibleRows = 1
			}
			half := visibleRows / 2
			if half < 1 {
				half = 1
			}
			m.cursor -= half
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.clampOffset()
		case "ctrl+f":
			// Full-page down.
			const overhead = 3
			visibleRows := m.height - overhead
			if visibleRows < 1 {
				visibleRows = 1
			}
			m.cursor += visibleRows
			if m.cursor >= len(m.items) {
				m.cursor = len(m.items) - 1
			}
			m.clampOffset()
		case "ctrl+b":
			// Full-page up.
			const overhead = 3
			visibleRows := m.height - overhead
			if visibleRows < 1 {
				visibleRows = 1
			}
			m.cursor -= visibleRows
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.clampOffset()
		case "enter":
			// Navigate to the memory referenced by the selected item.
			id := m.selectedMemoryID()
			if id != "" {
				return m, func() tea.Msg {
					return NavigateToMemoryMsg{ID: id}
				}
			}
		case "H":
			// Jump to top of visible area.
			m.cursor = m.offset
			m.clampOffset()
		case "M":
			// Jump to middle of visible area.
			const overhead = 3
			visibleRows := m.height - overhead
			if visibleRows < 1 {
				visibleRows = 1
			}
			visible := min(len(m.items)-m.offset, visibleRows)
			m.cursor = m.offset + visible/2
			m.clampOffset()
		case "L":
			// Jump to bottom of visible area.
			const overhead = 3
			visibleRows := m.height - overhead
			if visibleRows < 1 {
				visibleRows = 1
			}
			visible := min(len(m.items)-m.offset, visibleRows)
			if visible > 0 {
				m.cursor = m.offset + visible - 1
			}
			m.clampOffset()
		}
	}
	return m, nil
}

// SetSize satisfies the View interface.
func (m *GraphModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetData satisfies the View interface. Called by AppModel after every data poll.
func (m *GraphModel) SetData(d *data.LoadedData) {
	if d == nil {
		m.memories = nil
		m.links = nil
		m.adjacency = nil
		m.reverseAdj = nil
		m.memoryIndex = nil
		m.items = nil
		m.cursor = 0
		m.offset = 0
		return
	}

	m.memories = d.Memories
	m.links = d.Links

	// Build index and adjacency lists.
	m.memoryIndex = make(map[string]memory.Memory, len(d.Memories))
	for _, mem := range d.Memories {
		m.memoryIndex[mem.ID] = mem
	}

	m.adjacency = make(map[string][]memory.MemoryLink)
	m.reverseAdj = make(map[string][]memory.MemoryLink)
	for _, l := range d.Links {
		m.adjacency[l.FromID] = append(m.adjacency[l.FromID], l)
		m.reverseAdj[l.ToID] = append(m.reverseAdj[l.ToID], l)
	}

	m.rebuildItems()

	// Clamp cursor after data update.
	if m.cursor >= len(m.items) {
		if len(m.items) > 0 {
			m.cursor = len(m.items) - 1
		} else {
			m.cursor = 0
		}
	}
	m.clampOffset()
}

// rebuildItems constructs the flat navigable list of graphItems.
// Only memories that have outbound links appear as root nodes.
// Memories that only appear as link targets (never as "from") are not shown
// as root nodes to avoid duplication — they are already visible as children
// under the "from" node. This keeps the view unidirectional (outbound-only).
func (m *GraphModel) rebuildItems() {
	m.items = nil

	if len(m.links) == 0 {
		return
	}

	// Identify which memory IDs have outbound links — these become root nodes.
	hasSources := make(map[string]bool)
	for _, l := range m.links {
		hasSources[l.FromID] = true
	}

	// Iterate memories in stable order (d.Memories is ordered by created_at DESC).
	for i := range m.memories {
		mem := m.memories[i]
		if !hasSources[mem.ID] {
			// This memory only appears as a link target; skip as a root node.
			continue
		}

		// Add root memory item.
		memCopy := mem
		rootItem := graphItem{
			isMemory: true,
			mem:      &memCopy,
			depth:    0,
		}
		m.items = append(m.items, rootItem)

		// Add outbound links (from this memory to another).
		for _, l := range m.adjacency[mem.ID] {
			lCopy := l
			var target *memory.Memory
			if t, ok := m.memoryIndex[l.ToID]; ok {
				tCopy := t
				target = &tCopy
			}
			m.items = append(m.items, graphItem{
				isLink: true,
				mem:    &memCopy,
				link:   &lCopy,
				target: target,
				depth:  1,
			})
		}
	}
}

// View satisfies the View interface. Renders the full graph content area.
func (m *GraphModel) View() string {
	if m.width == 0 || m.height == 0 {
		return MutedStyle.Render("loading…")
	}

	if len(m.links) == 0 {
		msg := EmptyStateStyle.Render("No memory links recorded. Use `cx memory link` to create relationships.")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	title := TitleStyle.Render("Memory Graph")
	hint := MutedStyle.Render("  ↑↓/jk navigate  enter detail  ? help")

	// Content area: title (1 line) + blank (1 line) + scrollable list + hint bar (1 line).
	// The outer height includes title + blank + hint = 3 lines overhead.
	const overhead = 3
	visibleRows := m.height - overhead
	if visibleRows < 1 {
		visibleRows = 1
	}

	var rows []string
	end := m.offset + visibleRows
	if end > len(m.items) {
		end = len(m.items)
	}
	for i := m.offset; i < end; i++ {
		rows = append(rows, m.renderItem(i))
	}

	// Pad to fill the visible area so the hint bar stays at the bottom.
	// Reserve the last row for scroll indicators if content overflows.
	padTo := visibleRows
	scrollIndicator := ""
	if m.offset > 0 && end < len(m.items) {
		scrollIndicator = SubtleStyle.Render("↑↓ more")
		padTo--
	} else if m.offset > 0 {
		scrollIndicator = SubtleStyle.Render("↑ more above")
		padTo--
	} else if end < len(m.items) {
		scrollIndicator = SubtleStyle.Render("↓ more below")
		padTo--
	}
	for len(rows) < padTo {
		rows = append(rows, "")
	}
	if scrollIndicator != "" {
		rows = append(rows, scrollIndicator)
	}

	body := strings.Join(rows, "\n")
	return strings.Join([]string{title, "", body, hint}, "\n")
}

// renderItem renders a single graphItem row.
func (m *GraphModel) renderItem(i int) string {
	item := m.items[i]
	selected := i == m.cursor

	if item.isMemory {
		return m.renderMemoryRow(item, selected)
	}
	if item.isLink {
		return m.renderLinkRow(item, selected)
	}
	return ""
}

// renderMemoryRow renders a root memory node row.
func (m *GraphModel) renderMemoryRow(item graphItem, selected bool) string {
	if item.mem == nil {
		return ""
	}
	prefix := "  "
	if selected {
		prefix = "► "
	}

	typeLabel := EntityTypeBadge(item.mem.EntityType).Render(item.mem.EntityType)
	title := item.mem.Title
	if title == "" {
		title = item.mem.ID
	}
	maxTitleW := m.width - 30
	if maxTitleW < 10 {
		maxTitleW = 10
	}
	title = truncate(title, maxTitleW)

	line := fmt.Sprintf("%s%s: %s", prefix, typeLabel, title)
	if selected {
		return TableSelectedStyle.Render(line)
	}
	return TableRowStyle.Render(line)
}

// renderLinkRow renders an indented link child row.
// It shows the relation type (color-coded) and the target memory title.
func (m *GraphModel) renderLinkRow(item graphItem, selected bool) string {
	if item.link == nil {
		return ""
	}

	var prefix string
	if selected {
		prefix = "  ► "
	} else {
		prefix = "      "
	}

	// Determine if this is an outbound link from the current root memory
	// or an inbound link (other memory links to this one).
	isOutbound := item.mem != nil && item.link.FromID == item.mem.ID

	var dirIndicator, targetTitle string
	if isOutbound {
		dirIndicator = "→"
		if item.target != nil {
			targetTitle = item.target.Title
		} else {
			targetTitle = item.link.ToID
		}
	} else {
		dirIndicator = "←"
		if item.mem != nil {
			targetTitle = item.mem.Title
		} else {
			targetTitle = item.link.FromID
		}
	}

	if targetTitle == "" {
		if isOutbound {
			targetTitle = item.link.ToID
		} else {
			targetTitle = item.link.FromID
		}
	}

	relLabel := relationStyle(item.link.RelationType).Render(
		fmt.Sprintf("[%s]", item.link.RelationType),
	)

	maxTargetW := m.width - 35
	if maxTargetW < 10 {
		maxTargetW = 10
	}
	targetTitle = truncate(targetTitle, maxTargetW)
	line := fmt.Sprintf("%s%s %s %s", prefix, dirIndicator, relLabel, targetTitle)

	if selected {
		return TableSelectedStyle.Render(line)
	}
	return MutedStyle.Render(line)
}

// selectedMemoryID returns the memory ID relevant to the currently selected item.
// For memory rows, it returns that memory's ID.
// For link rows, it returns the linked (target) memory's ID so enter navigates to it.
func (m *GraphModel) selectedMemoryID() string {
	if len(m.items) == 0 || m.cursor >= len(m.items) {
		return ""
	}
	item := m.items[m.cursor]
	if item.isMemory && item.mem != nil {
		return item.mem.ID
	}
	if item.isLink && item.link != nil {
		// Determine which end to navigate to.
		isOutbound := item.mem != nil && item.link.FromID == item.mem.ID
		if isOutbound {
			return item.link.ToID
		}
		return item.link.FromID
	}
	return ""
}

// CursorPosition satisfies CursorPositioner so the status bar can show [cursor/total].
// Returns the 0-based cursor position; app.go adds 1 when displaying.
func (m *GraphModel) CursorPosition() (int, int) {
	return m.cursor, len(m.items)
}

// clampOffset adjusts the scroll offset so the cursor is always visible
// within the visible rows area.
func (m *GraphModel) clampOffset() {
	const overhead = 3
	visibleRows := m.height - overhead
	if visibleRows < 1 {
		visibleRows = 1
	}

	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+visibleRows {
		m.offset = m.cursor - visibleRows + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}
