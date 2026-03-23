// Package tui implements the cx dashboard TUI using Bubble Tea.
// app.go is the top-level model that owns tab routing, global key handling,
// and the 5-second poll cycle.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AngelMaldonado/cx/internal/tui/data"
)

// Tab identifies which view is currently active.
type Tab int

const (
	TabHome Tab = iota
	TabMemories
	TabSessions
	TabRuns
	TabSync
	TabNotes
	TabGraph
	TabCrossProject
)

var tabNames = []string{
	"1 Home",
	"2 Memories",
	"3 Sessions",
	"4 Runs",
	"5 Sync",
	"6 Notes",
	"7 Graph",
	"8 Cross-Project",
}

// tabCount is the total number of tabs.
const tabCount = 8

// View is the interface all tab view models must implement.
// Each view receives size and data updates from AppModel and handles its own
// key events when it is the active tab.
type View interface {
	Init() tea.Cmd
	Update(msg tea.Msg) (View, tea.Cmd)
	View() string
	SetSize(width, height int)
	SetData(d *data.LoadedData)
	// IsInputFocused reports whether a text input widget is currently focused
	// inside this view. AppModel checks this before intercepting h/l/←/→ for
	// tab navigation — when true, those keys are passed through to the view.
	IsInputFocused() bool
}

// CursorPositioner is an optional interface that views can implement to
// expose their cursor position for the status bar position indicator.
type CursorPositioner interface {
	CursorPosition() (cursor, total int)
}

// DataLoadedMsg carries refreshed data from the loader after a poll cycle.
type DataLoadedMsg struct {
	Data *data.LoadedData
	Err  error
}

// SyncResultMsg carries the result of a push or pull operation.
type SyncResultMsg struct {
	Action  string // "push" or "pull"
	Result  string // combined stdout+stderr
	Success bool
}

// NavigateToMemoryMsg asks AppModel to switch to the Memories tab and
// pre-select the memory with the given ID.
type NavigateToMemoryMsg struct {
	ID string
}

// ConfirmDeprecateMsg asks AppModel to show a deprecation confirmation dialog.
type ConfirmDeprecateMsg struct {
	ID    string
	Title string
}

// statusBar holds the rendered state for the bottom status bar.
type statusBar struct {
	left    string
	right   string
	hints   string // pre-rendered key hints
	width   int
	isStale bool
}

func (s statusBar) render() string {
	left := StatusBarStyle.Render(s.left)
	right := s.right
	if s.isStale {
		right = StatusStaleStyle.Render("stale  ") + StatusBarStyle.Render(right)
	} else {
		right = StatusBarStyle.Render(right)
	}

	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)

	// Compute available space for key hints.
	available := s.width - lw - rw
	center := s.hints
	cw := lipgloss.Width(center)
	if available < 0 {
		available = 0
	}
	// If hints don't fit, truncate them gracefully.
	if cw > available {
		center = truncateANSI(center, available)
		cw = lipgloss.Width(center)
	}

	pad := s.width - lw - cw - rw
	if pad < 0 {
		pad = 0
	}
	return left + center + strings.Repeat(" ", pad) + right
}

// AppModel is the root Bubble Tea model.
// It owns tab switching, global key routing, polling, and view delegation.
type AppModel struct {
	loader *data.Loader
	data   *data.LoadedData

	activeTab Tab
	views     map[Tab]View

	// detail is an optional full-screen overlay. When non-nil and active,
	// all Update and View calls are routed to the detail overlay instead of
	// the active tab.
	detail *DetailModel

	// Status bar rendered inline to avoid import cycle with components package.
	bar statusBar

	// Loading spinner (bubbles/spinner directly, not components.SpinnerModel).
	spin        spinner.Model
	spinActive  bool

	width  int
	height int

	loading     bool
	err         error
	lastRefresh time.Time
	quitting    bool
	helpOverlay bool
}

// NewApp creates a new AppModel for the given loader.
// TabHome uses the real HomeModel; remaining tabs use PlaceholderView until
// their view files are created in later tasks.
func NewApp(loader *data.Loader) AppModel {
	views := make(map[Tab]View, tabCount)
	for i := Tab(0); i < tabCount; i++ {
		views[i] = &PlaceholderView{name: tabNames[i]}
	}
	views[TabHome] = NewHome()
	views[TabMemories] = NewMemoriesModel(loader)
	views[TabSessions] = NewSessionsModel()
	views[TabRuns] = NewRuns()
	views[TabSync] = NewSyncModel()
	views[TabNotes] = NewNotesModel()
	views[TabGraph] = NewGraphModel()
	views[TabCrossProject] = NewCrossProjectModel(loader)

	s := spinner.New(spinner.WithSpinner(spinner.MiniDot))
	s.Style = SubtitleStyle

	m := AppModel{
		loader:     loader,
		activeTab:  TabHome,
		views:      views,
		detail:     NewDetail(),
		spin:       s,
		spinActive: true,
		loading:    true,
		bar:        statusBar{width: 80},
	}
	m.refreshStatusBar()
	return m
}

// Init starts the initial data load and the 5-second poll timer.
func (m AppModel) Init() tea.Cmd {
	return tea.Batch(
		m.loadData(),
		data.PollCmd(data.DefaultPollInterval),
		m.spin.Tick,
	)
}

// Update is the central Bubble Tea update function.
// Global keys are handled here first; everything else is delegated to the
// active view or relevant sub-component.
func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Route to detail overlay first if it is visible.
		if m.detail != nil && m.detail.IsVisible() {
			updated, cmd := m.detail.Update(msg)
			m.detail = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Close help overlay first if it is open.
		if m.helpOverlay {
			switch msg.String() {
			case "?", "esc", "q":
				m.helpOverlay = false
			}
			return m, nil
		}

		// When a text input is focused, only ctrl+c and esc are handled
		// globally — all other keys are passed directly to the active view so
		// printable characters (q, r, ?, 1-8, h, l, …) reach the input widget.
		if m.views[m.activeTab].IsInputFocused() {
			switch msg.String() {
			case "ctrl+c":
				m.quitting = true
				return m, tea.Quit
			default:
				// esc and all printable keys go to the view.
				// The view is responsible for blurring the input on esc.
				updated, cmd := m.views[m.activeTab].Update(msg)
				m.views[m.activeTab] = updated
				if cmd != nil {
					cmds = append(cmds, cmd)
				}
			}
			return m, tea.Batch(cmds...)
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit

		case "?":
			m.helpOverlay = true
			return m, nil

		case "r":
			m.loading = true
			m.spinActive = true
			cmds = append(cmds, m.loadData())

		case "tab":
			m.activeTab = (m.activeTab + 1) % tabCount
			m.refreshStatusBar()

		case "shift+tab":
			m.activeTab = (m.activeTab + tabCount - 1) % tabCount
			m.refreshStatusBar()

		case "l", "right":
			m.activeTab = (m.activeTab + 1) % tabCount
			m.refreshStatusBar()

		case "h", "left":
			m.activeTab = (m.activeTab + tabCount - 1) % tabCount
			m.refreshStatusBar()

		case "1":
			m.activeTab = TabHome
			m.refreshStatusBar()
		case "2":
			m.activeTab = TabMemories
			m.refreshStatusBar()
		case "3":
			m.activeTab = TabSessions
			m.refreshStatusBar()
		case "4":
			m.activeTab = TabRuns
			m.refreshStatusBar()
		case "5":
			m.activeTab = TabSync
			m.refreshStatusBar()
		case "6":
			m.activeTab = TabNotes
			m.refreshStatusBar()
		case "7":
			m.activeTab = TabGraph
			m.refreshStatusBar()
		case "8":
			m.activeTab = TabCrossProject
			m.refreshStatusBar()

		default:
			// Delegate to the active view.
			updated, cmd := m.views[m.activeTab].Update(msg)
			m.views[m.activeTab] = updated
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}

	case CloseDetailMsg:
		// Detail overlay closed — hide it and return to normal tab routing.
		if m.detail != nil {
			m.detail.Hide()
		}

	case ShowDeprecateMsg:
		// Relay deprecation request from detail overlay to the Memories view.
		// Hide the detail overlay first so the confirmation dialog is visible.
		if m.detail != nil {
			m.detail.Hide()
		}
		m.activeTab = TabMemories
		m.refreshStatusBar()
		if memView, ok := m.views[TabMemories].(*MemoriesModel); ok {
			memView.deprecateID = msg.ID
			memView.deprecateTitle = msg.Title
			memView.showDeprecateConfirm = true
		}

	case NavigateToMemoryMsg:
		// Open the detail overlay for the specified memory.
		if m.detail != nil && m.data != nil {
			for _, mem := range m.data.Memories {
				if mem.ID == msg.ID {
					m.detail.SetSize(m.width, m.height)
					m.detail.SetMemory(mem)
					m.detail.Show()
					break
				}
			}
		}
		// Always switch to the Memories tab so the user knows where they came from.
		m.activeTab = TabMemories
		m.refreshStatusBar()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.bar.width = msg.Width
		// Distribute content area to all views.
		contentH := m.contentHeight()
		for tab, v := range m.views {
			v.SetSize(msg.Width, contentH)
			m.views[tab] = v
		}
		// Also resize the detail overlay if it exists.
		if m.detail != nil {
			m.detail.SetSize(msg.Width, msg.Height)
		}

	case refreshAfterSyncMsg:
		// Sync operation (push/pull) completed — reload data immediately.
		m.loading = true
		m.spinActive = true
		cmds = append(cmds, m.loadData())

	case data.PollMsg:
		// Refresh data and restart the poll timer.
		m.loading = true
		m.spinActive = true
		cmds = append(cmds, m.loadData(), data.PollCmd(data.DefaultPollInterval))

	case DataLoadedMsg:
		m.loading = false
		m.spinActive = false
		m.lastRefresh = time.Now()
		if msg.Err != nil {
			m.err = msg.Err
			m.bar.isStale = true
		} else {
			m.err = nil
			m.data = msg.Data
			m.bar.isStale = false
			// Distribute data to all views.
			for _, v := range m.views {
				v.SetData(msg.Data)
			}
		}
		m.refreshStatusBar()

	default:
		// Forward to spinner (keeps the animation ticking).
		if m.spinActive {
			var spinCmd tea.Cmd
			m.spin, spinCmd = m.spin.Update(msg)
			if spinCmd != nil {
				cmds = append(cmds, spinCmd)
			}
		}

		// Also forward to active view.
		updated, cmd := m.views[m.activeTab].Update(msg)
		m.views[m.activeTab] = updated
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View renders the full-screen layout: tab bar + content + status bar.
func (m AppModel) View() string {
	if m.quitting {
		return ""
	}

	// Terminal too small.
	if m.width > 0 && m.height > 0 && (m.width < 60 || m.height < 15) {
		msg := fmt.Sprintf("Terminal too small (%dx%d).\nPlease resize to at least 60×15.", m.width, m.height)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, msg)
	}

	// Detail overlay takes precedence over help and normal tab content.
	if m.detail != nil && m.detail.IsVisible() {
		return m.detail.View()
	}

	// Help overlay.
	if m.helpOverlay {
		return m.renderHelp()
	}

	tabBar := m.renderTabBar()
	contentH := m.contentHeight()
	content := m.views[m.activeTab].View()
	// Force content to fill exactly contentH rows so the status bar is always
	// pinned to the last terminal row with no gap.
	content = lipgloss.NewStyle().
		Width(m.width).
		Height(contentH).
		MaxHeight(contentH).
		Render(content)
	bar := m.bar.render()

	return tabBar + "\n" + content + "\n" + bar
}

// tabShortNames are abbreviated tab labels used when the terminal is narrow.
var tabShortNames = []string{
	"1 Home",
	"2 Mem",
	"3 Sess",
	"4 Runs",
	"5 Sync",
	"6 Notes",
	"7 Graph",
	"8 XProj",
}

// renderTabBar renders the horizontal tab bar at the top.
// On narrow terminals (width < 100), it uses abbreviated names.
// On very narrow terminals (width < 70), it shows only the active tab with
// arrow indicators.
func (m AppModel) renderTabBar() string {
	names := tabNames
	if m.width > 0 && m.width < 100 {
		names = tabShortNames
	}

	// Very narrow: show only active tab with prev/next arrows.
	if m.width > 0 && m.width < 70 {
		prevTab := (m.activeTab + tabCount - 1) % tabCount
		nextTab := (m.activeTab + 1) % tabCount
		bar := MutedStyle.Render("← "+tabShortNames[prevTab]) +
			" " + ActiveTabStyle.Render(names[m.activeTab]) +
			" " + MutedStyle.Render(tabShortNames[nextTab]+" →")
		return bar
	}

	var tabs []string
	for i, name := range names {
		if Tab(i) == m.activeTab {
			tabs = append(tabs, ActiveTabStyle.Render(name))
		} else {
			tabs = append(tabs, TabStyle.Render(name))
		}
	}
	bar := strings.Join(tabs, "")
	// Safety: if the bar is wider than the terminal, truncate.
	if m.width > 0 && lipgloss.Width(bar) > m.width {
		bar = truncateANSI(bar, m.width)
	}
	return bar
}

// renderHelp renders a static help overlay listing all global key bindings.
func (m AppModel) renderHelp() string {
	lines := []string{
		HelpKeyStyle.Render("Navigation"),
		"  " + HelpKeyStyle.Render("1-8") + "          switch to tab",
		"  " + HelpKeyStyle.Render("tab") + "          next tab",
		"  " + HelpKeyStyle.Render("shift+tab") + "    previous tab",
		"  " + HelpKeyStyle.Render("l / →") + "        next tab",
		"  " + HelpKeyStyle.Render("h / ←") + "        previous tab",
		"  " + HelpKeyStyle.Render("j / ↓") + "        navigate down",
		"  " + HelpKeyStyle.Render("k / ↑") + "        navigate up",
		"",
		HelpKeyStyle.Render("Global"),
		"  " + HelpKeyStyle.Render("q / ctrl+c") + "   quit",
		"  " + HelpKeyStyle.Render("?") + "            toggle this help",
		"  " + HelpKeyStyle.Render("r") + "            force refresh",
		"",
		HelpKeyStyle.Render("Memories (tab 2)"),
		"  " + HelpKeyStyle.Render("j/k") + "          navigate list",
		"  " + HelpKeyStyle.Render("/") + "            search",
		"  " + HelpKeyStyle.Render("f") + "            filter by type",
		"  " + HelpKeyStyle.Render("d") + "            deprecate selected",
		"  " + HelpKeyStyle.Render("enter") + "        open detail overlay",
		"  " + HelpKeyStyle.Render("esc") + "          clear search/filter",
		"",
		HelpKeyStyle.Render("Sessions (tab 3)"),
		"  " + HelpKeyStyle.Render("j/k") + "          navigate list",
		"",
		HelpKeyStyle.Render("Runs (tab 4)"),
		"  " + HelpKeyStyle.Render("j/k") + "          navigate",
		"  " + HelpKeyStyle.Render("space") + "        expand/collapse session",
		"",
		HelpKeyStyle.Render("Sync (tab 5)"),
		"  " + HelpKeyStyle.Render("p") + "            push pending memories",
		"  " + HelpKeyStyle.Render("P") + "            push --all",
		"  " + HelpKeyStyle.Render("u") + "            pull from remote",
		"",
		HelpKeyStyle.Render("Detail overlay"),
		"  " + HelpKeyStyle.Render("esc") + "          close",
		"  " + HelpKeyStyle.Render("j/k") + "          scroll",
		"  " + HelpKeyStyle.Render("d") + "            deprecate (memories only)",
		"",
		MutedStyle.Render("Press ? or esc to close"),
	}
	content := HelpTextStyle.Render(strings.Join(lines, "\n"))
	box := HelpStyle.Render(content)
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, box)
	}
	return box
}

// refreshStatusBar rebuilds the status bar content fields.
func (m *AppModel) refreshStatusBar() {
	m.bar.left = tabNames[m.activeTab]

	// Append cursor position indicator if the active view exposes it.
	if v, ok := m.views[m.activeTab].(CursorPositioner); ok {
		cursor, total := v.CursorPosition()
		if total > 0 {
			m.bar.left += fmt.Sprintf("  [%d/%d]", cursor+1, total)
		}
	}

	// Key hints
	hints := []string{
		StatusKeyStyle.Render("tab") + StatusValueStyle.Render(" next"),
		StatusKeyStyle.Render("1-8") + StatusValueStyle.Render(" jump"),
		StatusKeyStyle.Render("?") + StatusValueStyle.Render(" help"),
		StatusKeyStyle.Render("q") + StatusValueStyle.Render(" quit"),
	}
	m.bar.hints = StatusBarStyle.Render("  ") + strings.Join(hints, StatusBarStyle.Render("  "))

	if m.spinActive {
		m.bar.right = m.spin.View() + " loading"
	} else if m.err != nil {
		m.bar.right = StatusErrorStyle.Render("refresh failed")
	} else if !m.lastRefresh.IsZero() {
		elapsed := time.Since(m.lastRefresh).Round(time.Second)
		m.bar.right = fmt.Sprintf("refreshed %s ago", elapsed)
	} else {
		m.bar.right = ""
	}
}

// contentHeight returns the height available for the active view's content area.
// It accounts for the tab bar row (1), the "\n" separator after it (1),
// the "\n" separator before the status bar (1), and the status bar row (1).
func (m AppModel) contentHeight() int {
	h := m.height - 4 // tab bar (1) + "\n" (1) + "\n" (1) + status bar (1)
	if h < 0 {
		h = 0
	}
	return h
}

// loadData returns a Cmd that loads all data in a background goroutine.
func (m AppModel) loadData() tea.Cmd {
	loader := m.loader
	return func() tea.Msg {
		d, err := loader.LoadAll()
		return DataLoadedMsg{Data: d, Err: err}
	}
}

// PlaceholderView is a temporary implementation for tabs not yet implemented.
// Tasks 1.9, 1.10, and Phase 2 tasks will replace these with real views by
// registering their own types that satisfy the View interface.
type PlaceholderView struct {
	name   string
	width  int
	height int
}

func (v *PlaceholderView) Init() tea.Cmd { return nil }

func (v *PlaceholderView) Update(msg tea.Msg) (View, tea.Cmd) { return v, nil }

func (v *PlaceholderView) View() string {
	text := MutedStyle.Render(v.name + " (coming soon)")
	if v.width > 0 && v.height > 0 {
		return lipgloss.Place(v.width, v.height, lipgloss.Center, lipgloss.Center, text)
	}
	return text
}

func (v *PlaceholderView) SetSize(w, h int) {
	v.width = w
	v.height = h
}

func (v *PlaceholderView) SetData(_ *data.LoadedData) {}

func (v *PlaceholderView) IsInputFocused() bool { return false }

// Run is the entry point called from cmd/dashboard.go.
// It creates the data loader, builds the AppModel, and runs the Bubble Tea
// event loop in full-screen (alt-screen) mode.
func Run(projectPath string) error {
	loader, err := data.NewLoader(projectPath)
	if err != nil {
		return err
	}
	defer loader.Close()

	app := NewApp(loader)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
