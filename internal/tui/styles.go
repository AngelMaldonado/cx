package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// Color palette — 256-color ANSI codes for broad terminal compatibility.
// lipgloss degrades gracefully on 8-color terminals.
var (
	// Semantic colors
	ColorAccent     = lipgloss.Color("86")  // cyan-green
	ColorSelected   = lipgloss.Color("33")  // blue
	ColorSuccess    = lipgloss.Color("82")  // green
	ColorWarning    = lipgloss.Color("214") // orange
	ColorError      = lipgloss.Color("196") // red
	ColorDeprecated = lipgloss.Color("241") // dimmed gray

	// Entity type colors
	ColorObservation = lipgloss.Color("75")  // blue
	ColorDecision    = lipgloss.Color("141") // purple
	ColorSession     = lipgloss.Color("114") // green
	ColorAgentRun    = lipgloss.Color("180") // yellow

	// Structural colors
	ColorBorder   = lipgloss.Color("236") // dark gray border
	ColorMuted    = lipgloss.Color("245") // secondary text
	ColorSubtle   = lipgloss.Color("240") // very muted text
	ColorPrimary  = lipgloss.Color("63")  // purple/indigo — tab highlights
	ColorTabInact = lipgloss.Color("242") // inactive tab text
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

// Status bar (bottom row)
var (
	StatusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("235")).
			Foreground(ColorMuted).
			Padding(0, 1)

	// StatusKeyStyle renders keyboard shortcut keys in the status bar.
	StatusKeyStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	// StatusValueStyle renders the descriptive text next to a key hint.
	StatusValueStyle = lipgloss.NewStyle().
				Foreground(ColorMuted)

	// StatusErrorStyle renders error messages in the status bar.
	StatusErrorStyle = lipgloss.NewStyle().
				Foreground(ColorError).
				Bold(true)

	// StatusStaleStyle renders the stale-data indicator.
	StatusStaleStyle = lipgloss.NewStyle().
				Foreground(ColorWarning)
)

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
			Foreground(lipgloss.Color("252"))

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
			Foreground(lipgloss.Color("252"))

	// TableSelectedStyle styles the currently selected row.
	TableSelectedStyle = lipgloss.NewStyle().
				Foreground(ColorSelected).
				Bold(true).
				Background(lipgloss.Color("239"))

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
			Foreground(lipgloss.Color("252"))

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
