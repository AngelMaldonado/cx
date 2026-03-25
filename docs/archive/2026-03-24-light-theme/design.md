---
name: light-theme
type: design
---

## Architecture

`internal/ui/theme.go` becomes the single source of truth for all colors in both the CLI output layer and the TUI dashboard. The flow is:

```
lipgloss.HasDarkBackground()
        │
        ├── true  → ActivePalette = MochaPalette
        └── false → ActivePalette = LattePalette

ActivePalette
    │
    ├── internal/ui: ColorGreen, ColorText, StyleSuccess, ... (alias ActivePalette fields)
    │
    └── internal/tui/styles.go: ColorAccent, ColorBorder, ... (alias ui.ActivePalette fields)
            │
            ├── graph.go: relationColors references ColorSelected, ColorError, ColorSuccess, ColorDeprecated
            └── crossproject.go: projectColorPalette uses fixed dual-contrast hex values
```

Go's initialization guarantees ensure correctness: imported packages initialize fully before the importing package's vars run, so `ui.ActivePalette` is set before `internal/tui/styles.go` var declarations execute.

## Technical Decisions

**Palette struct centralized in `internal/ui`**
The TUI imports `internal/ui` rather than duplicating palette values. This establishes a single authority for colors and prevents the two color systems from diverging again.

**`init()` detection, not constructor**
`lipgloss.HasDarkBackground()` is called once in `internal/ui`'s `init()` function and stored in `ActivePalette`. This is safe because `init()` runs before any application code; the palette is effectively immutable for the lifetime of the process.

**Mocha palette retains ANSI codes for structural slots**
The Mocha palette preserves the existing 256-color ANSI codes for structural slots (`StatusBg: "235"`, `ValueText: "252"`, `SelectedBg: "239"`, etc.) because those values already work on dark terminals. The Latte palette uses hex values for all slots. This asymmetry is intentional and correct.

**`projectColorPalette` uses fixed dual-contrast hex values**
Project label coloring in `crossproject.go` is aesthetic, not semantic. Rather than making it theme-dependent, a fixed set of 8 Catppuccin hex colors is chosen that have sufficient contrast on both dark and light backgrounds: Sapphire `#209fb5`, Mauve `#8839ef`, Green `#40a02b`, Peach `#fe640b`, Flamingo `#dd7878`, Teal `#179299`, Yellow `#df8e1d`, Lavender `#7287fd`.

**Entity type colors in TUI also converted**
`ColorObservation`, `ColorDecision`, `ColorSession`, `ColorAgentRun` in `styles.go` are currently ANSI codes (`"75"`, `"141"`, `"114"`, `"180"`). These are added as named slots in the `Palette` struct so they have correct Latte equivalents. Their Latte hex equivalents: Sapphire `#209fb5`, Mauve `#8839ef`, Green `#40a02b`, Yellow `#df8e1d`.

**spinner.go `styledFrames` moved to `init()`**
The `styledFrames` variable in `spinner.go` applies `StyleAccent.Render()` at package init time. `StyleAccent` is a package-level var in `theme.go` that references `ColorMauve`, which will reference `ActivePalette.Mauve`. Since `ActivePalette` is set in `theme.go`'s `init()`, and package-level vars initialize before `init()` functions, there is a sequencing risk: vars in `spinner.go` could run before `theme.go`'s `init()` sets `ActivePalette`. Moving `styledFrames` initialization to `spinner.go`'s own `init()` function eliminates this risk. (Go guarantees `init()` functions run after all package-level vars are initialized, and imported package `init()` functions run before the importing package's `init()`.)

## Implementation Notes

### Palette struct (add to `internal/ui/theme.go`)

```go
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
```

### MochaPalette instance

```go
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
    StatusBg:   "235",
    ValueText:  "252",
    SelectedBg: "239",
    // Entity types
    ObservationColor: "75",
    DecisionColor:    "141",
    SessionColor:     "114",
    AgentRunColor:    "180",
}
```

### LattePalette instance

```go
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
```

### ActivePalette detection (add to `internal/ui/theme.go`)

```go
var ActivePalette Palette

func init() {
    if lipgloss.HasDarkBackground() {
        ActivePalette = MochaPalette
    } else {
        ActivePalette = LattePalette
    }
}
```

### Existing color vars — rebuild as palette aliases (`internal/ui/theme.go`)

Replace the current hardcoded `var` block with:

```go
var (
    ColorGreen   = ActivePalette.Green
    ColorRed     = ActivePalette.Red
    ColorYellow  = ActivePalette.Yellow
    ColorBlue    = ActivePalette.Blue
    ColorMauve   = ActivePalette.Mauve
    ColorText     = ActivePalette.Text
    ColorSubtext = ActivePalette.Subtext
    ColorOverlay = ActivePalette.Overlay
    ColorSurface = ActivePalette.Surface
    ColorBase    = ActivePalette.Base
)
```

The style vars (`StyleSuccess`, `StyleWarning`, etc.) do not need changes — they already reference `ColorGreen`, `ColorYellow`, etc. Once those color vars source from `ActivePalette`, the styles automatically pick up the correct theme.

### spinner.go — `styledFrames` init ordering fix

In `internal/ui/spinner.go`, locate the var block that initializes `styledFrames` by applying `StyleAccent.Render()` to frame strings. Move this initialization into an `init()` function:

```go
// Before (risky):
var styledFrames = func() []string { ... StyleAccent.Render(f) ... }()

// After (safe):
var styledFrames []string

func init() {
    // Initialize after theme.go's init() has set ActivePalette
    for _, f := range frames {
        styledFrames = append(styledFrames, StyleAccent.Render(f))
    }
}
```

### `internal/tui/styles.go` — full refactor

1. Add import: `"github.com/AngelMaldonado/cx/internal/ui"`
2. Replace the entire color constants block with palette aliases:

```go
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
    ColorBorder     = ui.ActivePalette.Border
    ColorMuted      = ui.ActivePalette.Muted
    ColorSubtle     = ui.ActivePalette.Subtle
    ColorPrimary    = ui.ActivePalette.Primary
    ColorTabInact   = ui.ActivePalette.TabInact
    // New vars for previously-inline ANSI literals
    ColorStatusBg   = ui.ActivePalette.StatusBg
    ColorValueText  = ui.ActivePalette.ValueText
    ColorSelectedBg = ui.ActivePalette.SelectedBg
)
```

3. Replace inline ANSI literals in style var declarations:
   - `StatusBarStyle`: `Background(lipgloss.Color("235"))` → `Background(ColorStatusBg)`
   - `ValueStyle`: `Foreground(lipgloss.Color("252"))` → `Foreground(ColorValueText)`
   - `TableRowStyle`: `Foreground(lipgloss.Color("252"))` → `Foreground(ColorValueText)`
   - `TableSelectedStyle`: `Background(lipgloss.Color("239"))` → `Background(ColorSelectedBg)`
   - `HelpTextStyle`: `Foreground(lipgloss.Color("252"))` → `Foreground(ColorValueText)`

### `internal/tui/graph.go` — inline consolidation

Replace the `relationColors` map (lines 53-58) with:

```go
var relationColors = map[string]lipgloss.Color{
    "related-to":  ColorSelected,   // was "33"
    "caused-by":   ColorError,      // was "196"
    "resolved-by": ColorSuccess,    // was "82"
    "see-also":    ColorDeprecated, // was "241"
}
```

Remove the direct `lipgloss` import if it is no longer used after this change (check whether `lipgloss.Color` or `lipgloss.NewStyle` is still referenced in the file — `relationStyle` function uses `lipgloss.NewStyle`, so the import stays).

### `internal/tui/crossproject.go` — inline consolidation

Replace `projectColorPalette` (lines 27-36) with fixed dual-contrast hex values:

```go
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
```

These hex values are from the Catppuccin Latte palette and have sufficient contrast on both dark and light backgrounds.

### Files that require NO changes

- `internal/ui/output.go` — inline styles reference `ColorText` and `ColorOverlay`, which become palette-driven automatically
- `internal/ui/forms.go` — references `ColorMauve`, `ColorSubtext`, `ColorRed`, etc. — all palette-driven after theme.go refactor
- `internal/tui/sessions.go` — `renderModeTag` references `ColorSuccess`, `ColorSelected`, `ColorWarning` from styles.go
- `internal/tui/memories.go` — `renderFilterChips` references `ColorAccent`, `ColorMuted`, `ColorSelected` from styles.go
- `internal/tui/helpers.go` — uses `glamour.WithAutoStyle()` which handles light/dark markdown automatically

### ANSI-to-Latte mapping reference (for review)

| Old (Mocha ANSI) | Semantic role | Latte hex |
|---|---|---|
| `"86"` cyan-green | Accent | `#179299` Teal |
| `"33"` blue | Selected | `#209fb5` Sapphire |
| `"82"` green | Success | `#40a02b` Green |
| `"214"` orange | Warning | `#fe640b` Peach |
| `"196"` red | Error | `#d20f39` Red |
| `"241"` dimmed gray | Deprecated | `#9ca0b0` Overlay0 |
| `"236"` dark gray | Border | `#acb0be` Surface2 |
| `"245"` secondary text | Muted | `#7c7f93` Overlay2 |
| `"240"` very muted | Subtle | `#8c8fa1` Overlay1 |
| `"63"` purple/indigo | Primary | `#8839ef` Mauve |
| `"242"` inactive tab | TabInact | `#9ca0b0` Overlay0 |
| `"235"` status bar bg | StatusBg | `#dce0e8` Crust |
| `"252"` value text | ValueText | `#4c4f69` Text |
| `"239"` selected row bg | SelectedBg | `#ccd0da` Surface0 |
| `"75"` blue (observation) | ObservationColor | `#209fb5` Sapphire |
| `"141"` purple (decision) | DecisionColor | `#8839ef` Mauve |
| `"114"` green (session) | SessionColor | `#40a02b` Green |
| `"180"` yellow (agent run) | AgentRunColor | `#df8e1d` Yellow |

### Verification steps

1. `go build ./...` — must compile cleanly
2. `go vet ./...` — must pass with no new issues
3. Dark terminal: `cx tui` and `cx doctor` must match existing Catppuccin Mocha appearance (no regression)
4. Light terminal (macOS Terminal Basic or iTerm white background):
   - `cx doctor` — status symbols (green check, red X, yellow warning) readable
   - `cx brainstorm status` — headers, items, muted text visible
   - `cx tui` — all 8 tabs: tab bar, selected rows, status bar, badges, graph relations, project colors all readable
   - `cx init` in a test directory — huh form readable
   - Spinner visible (not Mocha mauve on white)
5. Pipe/non-TTY: `cx doctor | cat` — `HasDarkBackground()` returns false (no TTY), defaults to Mocha, no crash
