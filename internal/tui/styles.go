package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	ui "github.com/AngelMaldonado/cx/internal/ui"
)

// Color palette — sourced from ui.ActivePalette, which is set in internal/ui's
// init() before any internal/tui code runs. Go's cross-package init ordering
// guarantees this is safe as package-level var initializers.
var (
	// Semantic colors
	ColorAccent     = ui.ActivePalette.Accent
	ColorSelected   = ui.ActivePalette.Selected
	ColorSuccess    = ui.ActivePalette.Success
	ColorWarning    = ui.ActivePalette.Warning
	ColorError      = ui.ActivePalette.Error
	ColorDeprecated = ui.ActivePalette.Deprecated

	// Entity type colors
	ColorObservation = ui.ActivePalette.ObservationColor
	ColorDecision    = ui.ActivePalette.DecisionColor
	ColorSession     = ui.ActivePalette.SessionColor
	ColorAgentRun    = ui.ActivePalette.AgentRunColor

	// Structural colors
	ColorBorder   = ui.ActivePalette.Border
	ColorMuted    = ui.ActivePalette.Muted
	ColorSubtle   = ui.ActivePalette.Subtle
	ColorPrimary  = ui.ActivePalette.Primary
	ColorTabInact = ui.ActivePalette.TabInact

	// Structural layout colors (previously inline ANSI literals)
	ColorStatusBg   = ui.ActivePalette.StatusBg
	ColorValueText  = ui.ActivePalette.ValueText
	ColorSelectedBg = ui.ActivePalette.SelectedBg
)

// Tab bar styles
var (
	TabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(ColorTabInact)

	ActiveTabStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(ColorPrimary).
			Bold(true).
			Underline(true)
)

// Pane / layout styles
var (
	// PaneStyle is a bordered content pane with rounded corners.
	PaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1)

	// PreviewStyle is the right-side preview pane — slightly distinct border.
	PreviewStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorAccent).
			Padding(0, 1)
)

// Status bar (bottom row).
// StatusBarStyle is the only lipgloss style — it wraps the full bar with
// a continuous background. Inner content uses StatusKey/StatusValue helper
// functions that emit raw ANSI foreground codes without resetting the
// background set by the outer wrapper.
var (
	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorStatusBg).
			Foreground(ColorMuted)
)

// StatusKey renders a keyboard shortcut key with accent color + bold,
// using raw ANSI so the outer StatusBarStyle background is preserved.
func StatusKey(s string) string {
	return "\033[1m" + colorToANSI(ColorAccent) + s + "\033[22m" + colorToANSI(ColorMuted)
}

// StatusError renders error text in the status bar.
func StatusError(s string) string {
	return "\033[1m" + colorToANSI(ColorError) + s + "\033[22m" + colorToANSI(ColorMuted)
}

// StatusStale renders the stale-data indicator.
func StatusStale(s string) string {
	return colorToANSI(ColorWarning) + s + colorToANSI(ColorMuted)
}

// colorToANSI converts a lipgloss.Color to a raw ANSI foreground escape.
func colorToANSI(c lipgloss.Color) string {
	r := lipgloss.NewStyle().Foreground(c).Render("")
	// Extract just the SGR prefix (everything before the reset).
	// Lipgloss renders "" as: <SGR seq>\033[0m — we want the SGR seq only.
	if idx := strings.Index(r, "\033[0m"); idx >= 0 {
		return r[:idx]
	}
	return r
}

// Entity type badge styles — used to label memory types in the list and preview.
var (
	ObservationBadge = lipgloss.NewStyle().
				Foreground(ColorObservation).
				Bold(true)

	DecisionBadge = lipgloss.NewStyle().
			Foreground(ColorDecision).
			Bold(true)

	SessionBadge = lipgloss.NewStyle().
			Foreground(ColorSession).
			Bold(true)

	AgentRunBadge = lipgloss.NewStyle().
			Foreground(ColorAgentRun).
			Bold(true)
)

// Result status badge styles — used in the agent runs view.
var (
	SuccessBadge = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	BlockedBadge = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	PendingBadge = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	NeedsInputBadge = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)
)

// Typography styles
var (
	// TitleStyle is the main heading for a view or dialog.
	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	// SubtitleStyle is a secondary heading.
	SubtitleStyle = lipgloss.NewStyle().
			Foreground(ColorAccent)

	// LabelStyle renders field labels ("Title:", "Tags:", etc.).
	LabelStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Bold(true)

	// ValueStyle renders field values.
	ValueStyle = lipgloss.NewStyle().
			Foreground(ColorValueText)

	// MutedStyle renders secondary / de-emphasised text.
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// SubtleStyle renders very muted text (timestamps, metadata).
	SubtleStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle)
)

// Table / list styles
var (
	// TableHeaderStyle styles the column header row.
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true).
				BorderBottom(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(ColorBorder)

	// TableRowStyle styles a normal (unselected) list row.
	TableRowStyle = lipgloss.NewStyle().
			Foreground(ColorValueText)

	// TableSelectedStyle styles the currently selected row.
	TableSelectedStyle = lipgloss.NewStyle().
				Foreground(ColorSelected).
				Bold(true).
				Background(ColorSelectedBg)

	// DeprecatedRowStyle styles a deprecated memory row — strikethrough + dimmed.
	// Both visual cues (color + strikethrough) are applied for accessibility.
	DeprecatedRowStyle = lipgloss.NewStyle().
				Foreground(ColorDeprecated).
				Strikethrough(true)
)

// Search / filter styles
var (
	// SearchStyle is the unfocused search bar container.
	// No border — renders as a single terminal row so layout math stays exact.
	SearchStyle = lipgloss.NewStyle().
			Foreground(ColorMuted).
			Padding(0, 1)

	// FilterActiveStyle highlights the search bar when active.
	// No border — renders as a single terminal row so layout math stays exact.
	FilterActiveStyle = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true).
				Padding(0, 1)
)

// Help / overlay styles
var (
	// HelpStyle renders the help overlay box.
	HelpStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	// HelpTextStyle renders text inside the help overlay.
	HelpTextStyle = lipgloss.NewStyle().
			Foreground(ColorValueText)

	// HelpKeyStyle renders key names inside the help overlay.
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)
)

// Dialog / confirmation styles
var (
	// DialogStyle is a modal confirmation dialog box.
	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorWarning).
			Padding(1, 2)

	// DialogTitleStyle renders the dialog prompt text.
	DialogTitleStyle = lipgloss.NewStyle().
				Foreground(ColorWarning).
				Bold(true)
)

// Empty state style — rendered when a view has no data to display.
var EmptyStateStyle = lipgloss.NewStyle().
	Foreground(ColorMuted).
	Italic(true)

// ErrorStyle renders an error message inline in a view body.
var ErrorStyle = lipgloss.NewStyle().
	Foreground(ColorError).
	Bold(true)

// EntityTypeBadge returns the appropriate badge style for a given entity_type string.
// This centralises badge colour selection so all views stay consistent.
func EntityTypeBadge(entityType string) lipgloss.Style {
	switch entityType {
	case "observation", "agent_interaction":
		return ObservationBadge
	case "decision":
		return DecisionBadge
	case "session":
		return SessionBadge
	case "agent_run":
		return AgentRunBadge
	default:
		return MutedStyle
	}
}

// ResultStatusBadge returns the appropriate badge style for a result_status string.
func ResultStatusBadge(status string) lipgloss.Style {
	switch status {
	case "success":
		return SuccessBadge
	case "blocked":
		return BlockedBadge
	case "needs-input":
		return NeedsInputBadge
	default:
		return PendingBadge
	}
}
