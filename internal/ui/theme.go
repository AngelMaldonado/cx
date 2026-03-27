package ui

import "github.com/charmbracelet/lipgloss"

// Palette holds all color slots used by both the CLI output layer and the TUI
// dashboard. It is populated once in init() based on terminal background
// detection and never mutated after that.
type Palette struct {
	// Base Catppuccin colors
	Green   lipgloss.Color
	Red     lipgloss.Color
	Yellow  lipgloss.Color
	Blue    lipgloss.Color
	Mauve   lipgloss.Color
	Text    lipgloss.Color
	Subtext lipgloss.Color
	Overlay lipgloss.Color
	Surface lipgloss.Color
	Base    lipgloss.Color
	// Semantic roles (mapped from TUI ANSI constants)
	Accent     lipgloss.Color // cyan-green tones
	Selected   lipgloss.Color // selection highlight color
	Success    lipgloss.Color // positive status
	Warning    lipgloss.Color // caution status
	Error      lipgloss.Color // error/failure status
	Deprecated lipgloss.Color // dimmed/struck-through items
	Border     lipgloss.Color // pane and table borders
	Muted      lipgloss.Color // secondary text
	Subtle     lipgloss.Color // very muted text (timestamps, metadata)
	Primary    lipgloss.Color // tab highlight and primary accent
	TabInact   lipgloss.Color // inactive tab text
	// Structural layout colors (previously inline ANSI literals in styles.go)
	StatusBg   lipgloss.Color // status bar background
	ValueText  lipgloss.Color // normal value / row text
	SelectedBg lipgloss.Color // selected row background
	// Entity type colors (for memory type badges)
	ObservationColor lipgloss.Color
	DecisionColor    lipgloss.Color
	SessionColor     lipgloss.Color
	AgentRunColor    lipgloss.Color
}

// MochaPalette is the Catppuccin Mocha (dark) palette.
// Semantic and structural slots use 256-color ANSI codes matching the
// original hardcoded values; base slots use Mocha hex.
var MochaPalette = Palette{
	// Catppuccin Mocha hex
	Green:   "#a6e3a1",
	Red:     "#f38ba8",
	Yellow:  "#f9e2af",
	Blue:    "#89b4fa",
	Mauve:   "#cba6f7",
	Text:    "#cdd6f4",
	Subtext: "#a6adc8",
	Overlay: "#6c7086",
	Surface: "#45475a",
	Base:    "#1e1e2e",
	// Semantic (ANSI 256 — existing values preserved)
	Accent:     "86",
	Selected:   "33",
	Success:    "82",
	Warning:    "214",
	Error:      "196",
	Deprecated: "241",
	Border:     "236",
	Muted:      "245",
	Subtle:     "240",
	Primary:    "63",
	TabInact:   "242",
	// Structural
	StatusBg:   "#45475a", // Surface0 — purple-tinted to match Mocha theme
	ValueText:  "252",
	SelectedBg: "239",
	// Entity types
	ObservationColor: "75",
	DecisionColor:    "141",
	SessionColor:     "114",
	AgentRunColor:    "180",
}

// LattePalette is the Catppuccin Latte (light) palette.
// All slots use hex values.
var LattePalette = Palette{
	// Catppuccin Latte hex
	Green:   "#40a02b",
	Red:     "#d20f39",
	Yellow:  "#df8e1d",
	Blue:    "#1e66f5",
	Mauve:   "#8839ef",
	Text:    "#4c4f69",
	Subtext: "#5c5f77",
	Overlay: "#7c7f93",
	Surface: "#acb0be",
	Base:    "#eff1f5",
	// Semantic
	Accent:     "#179299", // Teal
	Selected:   "#209fb5", // Sapphire
	Success:    "#40a02b", // Green
	Warning:    "#fe640b", // Peach
	Error:      "#d20f39", // Red
	Deprecated: "#9ca0b0", // Overlay0
	Border:     "#acb0be", // Surface2
	Muted:      "#7c7f93", // Overlay2
	Subtle:     "#8c8fa1", // Overlay1
	Primary:    "#8839ef", // Mauve
	TabInact:   "#9ca0b0", // Overlay0
	// Structural
	StatusBg:   "#dce0e8", // Crust
	ValueText:  "#4c4f69", // Text
	SelectedBg: "#ccd0da", // Surface0
	// Entity types
	ObservationColor: "#209fb5", // Sapphire
	DecisionColor:    "#8839ef", // Mauve
	SessionColor:     "#40a02b", // Green
	AgentRunColor:    "#df8e1d", // Yellow
}

// ActivePalette is the palette selected at init() time based on terminal
// background detection. It is safe to read after package initialization.
var ActivePalette Palette

// Color* vars are aliases of ActivePalette fields, set in init() after palette
// detection. They are package-level vars so other files in this package
// (output.go, forms.go, spinner.go) can reference them without importing ui.
var (
	ColorGreen   lipgloss.Color
	ColorRed     lipgloss.Color
	ColorYellow  lipgloss.Color
	ColorBlue    lipgloss.Color
	ColorMauve   lipgloss.Color
	ColorText    lipgloss.Color
	ColorSubtext lipgloss.Color
	ColorOverlay lipgloss.Color
	ColorSurface lipgloss.Color
	ColorBase    lipgloss.Color
)

// Style* vars are pre-built lipgloss styles, set in init() after Color* vars.
var (
	StyleSuccess lipgloss.Style
	StyleWarning lipgloss.Style
	StyleError   lipgloss.Style
	StyleInfo    lipgloss.Style
	StyleMuted   lipgloss.Style
	StyleBold    lipgloss.Style
	StyleAccent  lipgloss.Style
	StyleHeader  lipgloss.Style
	StyleItem    lipgloss.Style
)

// Symbol* vars are pre-rendered colored Unicode symbols, set in init() after
// Style* vars.
var (
	SymbolSuccess string
	SymbolError   string
	SymbolWarning string
)

func init() {
	// 1. Detect terminal background and select palette.
	if lipgloss.HasDarkBackground() {
		ActivePalette = MochaPalette
	} else {
		ActivePalette = LattePalette
	}

	// 2. Assign Color* aliases from the active palette.
	ColorGreen = ActivePalette.Green
	ColorRed = ActivePalette.Red
	ColorYellow = ActivePalette.Yellow
	ColorBlue = ActivePalette.Blue
	ColorMauve = ActivePalette.Mauve
	ColorText = ActivePalette.Text
	ColorSubtext = ActivePalette.Subtext
	ColorOverlay = ActivePalette.Overlay
	ColorSurface = ActivePalette.Surface
	ColorBase = ActivePalette.Base

	// 3. Build Style* vars using the now-set Color* vars.
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorGreen)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorYellow)
	StyleError = lipgloss.NewStyle().Foreground(ColorRed)
	StyleInfo = lipgloss.NewStyle().Foreground(ColorBlue)
	StyleMuted = lipgloss.NewStyle().Foreground(ColorOverlay)
	StyleBold = lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	StyleAccent = lipgloss.NewStyle().Foreground(ColorMauve)
	StyleHeader = lipgloss.NewStyle().Foreground(ColorMauve).Bold(true)
	StyleItem = lipgloss.NewStyle().Foreground(ColorText)

	// 4. Render Symbol* vars using the now-set Style* vars.
	SymbolSuccess = StyleSuccess.Render("\u2713")
	SymbolError = StyleError.Render("\u2717")
	SymbolWarning = StyleWarning.Render("\u26a0")

	// 5. Initialize styledFrames for spinner.go using StyleAccent.
	// This must happen here (in theme.go's init) rather than in spinner.go's
	// init because init() functions within a package run in source-file
	// alphabetical order: spinner.go sorts before theme.go, so spinner's init
	// would execute before StyleAccent is set. Consolidating here eliminates
	// the hazard.
	styledFrames = make([]string, len(spinnerFrames))
	for i, f := range spinnerFrames {
		styledFrames[i] = StyleAccent.Render(f)
	}
}
