package ui

import "github.com/charmbracelet/lipgloss"

// Catppuccin Mocha palette
var (
	ColorGreen   = lipgloss.Color("#a6e3a1")
	ColorRed     = lipgloss.Color("#f38ba8")
	ColorYellow  = lipgloss.Color("#f9e2af")
	ColorBlue    = lipgloss.Color("#89b4fa")
	ColorMauve   = lipgloss.Color("#cba6f7")
	ColorText    = lipgloss.Color("#cdd6f4")
	ColorSubtext = lipgloss.Color("#a6adc8")
	ColorOverlay = lipgloss.Color("#6c7086")
	ColorSurface = lipgloss.Color("#45475a")
	ColorBase    = lipgloss.Color("#1e1e2e")
)

// Pre-built styles
var (
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorGreen)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorYellow)
	StyleError   = lipgloss.NewStyle().Foreground(ColorRed)
	StyleInfo    = lipgloss.NewStyle().Foreground(ColorBlue)
	StyleMuted   = lipgloss.NewStyle().Foreground(ColorOverlay)
	StyleBold    = lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	StyleAccent  = lipgloss.NewStyle().Foreground(ColorMauve)
	StyleHeader  = lipgloss.NewStyle().Foreground(ColorMauve).Bold(true)
	StyleItem    = lipgloss.NewStyle().Foreground(ColorText)
)

// Pre-rendered symbols
var (
	SymbolSuccess = StyleSuccess.Render("\u2713")
	SymbolError   = StyleError.Render("\u2717")
	SymbolWarning = StyleWarning.Render("\u26a0")
)
