---
name: light-theme
type: masterfile
---

## Problem

The cx CLI and TUI dashboard are unreadable on light terminal backgrounds. Colors were chosen for Catppuccin Mocha (dark), so nearly every color — text, borders, status bars, selected rows — is either invisible or extremely low-contrast on white or near-white backgrounds. Users on macOS Terminal with "Basic" profile, iTerm with "Light Background", or any default-light terminal see a broken experience.

## Context

The codebase has two separate color systems that must be unified:

**System 1: `internal/ui/` (CLI output)**
- `theme.go` defines 10 Catppuccin Mocha hex color constants (`ColorGreen`, `ColorText`, etc.) and 9 pre-built `lipgloss.Style` variables.
- `output.go` has 2 inline `lipgloss.NewStyle()` calls in `PrintItem` that bypass the palette.
- `spinner.go` uses `StyleAccent` (references `ColorMauve`).
- `forms.go` constructs the `huh` form theme via `CXTheme()` which references 7 palette colors.

**System 2: `internal/tui/` (dashboard)**
- `styles.go` defines 15 color constants using 256-color ANSI codes (strings like `"86"`, `"235"`, `"239"`, `"252"`) and 20+ style variables. Three inline ANSI literals appear directly in styles (not via color constants): `lipgloss.Color("235")` in `StatusBarStyle`, `lipgloss.Color("252")` in `ValueStyle` and `TableRowStyle`, `lipgloss.Color("239")` in `TableSelectedStyle`, and `lipgloss.Color("252")` in `HelpTextStyle`.
- `graph.go` lines 53-58: `relationColors` map — 4 inline ANSI color literals.
- `crossproject.go` lines 27-36: `projectColorPalette` slice — 8 inline ANSI color literals.
- `sessions.go` lines 379-383: `renderModeTag` function — 3 inline `lipgloss.NewStyle()` calls referencing `ColorSuccess`, `ColorSelected`, `ColorWarning` from styles.go.
- `memories.go` lines 736-744: `renderFilterChips` — 2 inline `lipgloss.NewStyle()` calls referencing `ColorAccent` and `ColorMuted`.

**Worst offenders on light backgrounds:**
- `lipgloss.Color("252")` in `styles.go` — near-white text, invisible on white backgrounds.
- `lipgloss.Color("235")` in `styles.go` — near-black status bar background on white.
- `lipgloss.Color("239")` in `styles.go` — near-black selected-row background on white.
- `ColorText` = `#cdd6f4` in `theme.go` — near-white, invisible on light.
- `ColorYellow` = `#f9e2af` in `theme.go` — cream color, invisible on white.
- `ColorBorder` = `"236"` in `styles.go` — invisible borders on white.

**Detection:** `lipgloss.HasDarkBackground()` is already imported via glamour's `WithAutoStyle()` in `helpers.go`, proving the dependency is in the tree. The function is also directly accessible from any file importing `lipgloss`.

**Palette chosen:** Catppuccin Latte for light mode; Catppuccin Mocha (existing) for dark mode.

**Module path:** `github.com/AngelMaldonado/cx`

## Direction

### Architecture

`internal/ui/theme.go` becomes the single source of truth for all colors in both the CLI and TUI. The `internal/tui` package imports from `internal/ui` for all semantic color slots.

**Step 1 — Palette struct in `internal/ui/theme.go`:**

```go
type Palette struct {
    // Semantic roles
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
    // Extended roles needed by TUI
    Accent     lipgloss.Color // cyan-green tones (Teal in Latte, mapped from ANSI 86)
    Selected   lipgloss.Color // blue (same as Blue or Sapphire)
    Success    lipgloss.Color // green (same as Green)
    Warning    lipgloss.Color // orange/peach (Peach in Latte)
    Error      lipgloss.Color // red (same as Red)
    Deprecated lipgloss.Color // dimmed gray (Overlay0 in Latte)
    Border     lipgloss.Color // border gray (Surface2 in Latte, Surface0 in Mocha)
    Muted      lipgloss.Color // secondary text (Overlay2 in Latte)
    Subtle     lipgloss.Color // very muted (Overlay1 in Latte)
    Primary    lipgloss.Color // tab highlight (Mauve or Lavender)
    TabInact   lipgloss.Color // inactive tab (Overlay0)
    // Status bar
    StatusBg   lipgloss.Color // status bar background (Crust in Latte, ANSI 235 in Mocha)
    // Value / row text
    ValueText  lipgloss.Color // normal value text (Text in Latte, ANSI 252 in Mocha)
    SelectedBg lipgloss.Color // selected row background (Surface0 in Latte, ANSI 239 in Mocha)
}
```

**Step 2 — Predefined palettes:**

```go
var MochaPalette = Palette{
    Green: "#a6e3a1", Red: "#f38ba8", Yellow: "#f9e2af",
    Blue: "#89b4fa", Mauve: "#cba6f7", Text: "#cdd6f4",
    Subtext: "#a6adc8", Overlay: "#6c7086", Surface: "#45475a", Base: "#1e1e2e",
    Accent: "86", Selected: "33", Success: "82", Warning: "214", Error: "196",
    Deprecated: "241", Border: "236", Muted: "245", Subtle: "240",
    Primary: "63", TabInact: "242", StatusBg: "235", ValueText: "252", SelectedBg: "239",
}

var LattePalette = Palette{
    Green: "#40a02b", Red: "#d20f39", Yellow: "#df8e1d",
    Blue: "#1e66f5", Mauve: "#8839ef", Text: "#4c4f69",
    Subtext: "#5c5f77", Overlay: "#7c7f93", Surface: "#acb0be", Base: "#eff1f5",
    Accent: "#179299", Selected: "#209fb5", Success: "#40a02b", Warning: "#fe640b",
    Error: "#d20f39", Deprecated: "#9ca0b0", Border: "#acb0be", Muted: "#7c7f93",
    Subtle: "#8c8fa1", Primary: "#8839ef", TabInact: "#9ca0b0",
    StatusBg: "#dce0e8", ValueText: "#4c4f69", SelectedBg: "#ccd0da",
}
```

**Step 3 — Active palette detection:**

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

**Step 4 — Rebuild style variables from palette:**

The existing color `var` block (`ColorGreen`, `ColorText`, etc.) and style vars (`StyleSuccess`, etc.) are rebuilt from `ActivePalette` instead of hardcoded hex values. The symbol vars (`SymbolSuccess`, etc.) remain unchanged since they are derived from the style vars.

The `internal/ui` package keeps backward-compatible exported names: `ColorGreen = ActivePalette.Green`, etc.

**Step 5 — `internal/tui/styles.go` refactor:**

The 15 ANSI color constants are replaced with references to `ui.ActivePalette` fields. The 3 inline ANSI literals in style var declarations are replaced. The package gains an import of `github.com/AngelMaldonado/cx/internal/ui`. All `Color*` vars become:
```go
var (
    ColorAccent   = ui.ActivePalette.Accent
    ColorSelected = ui.ActivePalette.Selected
    // ...
)
```

**Step 6 — Consolidate inline colors:**

- `graph.go`: `relationColors` map uses Latte/Mocha palette values via `ui.ActivePalette` fields. The 4 ANSI literals (`"33"`, `"196"`, `"82"`, `"241"`) map to `Selected`, `Error`, `Success`, `Deprecated` respectively.
- `crossproject.go`: `projectColorPalette` slice replaces 8 ANSI literals with Catppuccin Latte/Mocha accent colors. The palette is defined in two variants and selected via `ui.ActivePalette`... Or simpler: use a fixed set of Catppuccin hex colors that work on both backgrounds. Given the use of a fixed palette for project color-coding (not semantic), use a set of Catppuccin hex accents that have sufficient contrast on both backgrounds (e.g., Sapphire `#209fb5`, Mauve `#8839ef`, Green `#40a02b`, Peach `#fe640b`, Flamingo `#dd7878`, Teal `#179299`, Yellow `#df8e1d`, Lavender `#7287fd`). These same values also work on dark backgrounds.
- `sessions.go`: `renderModeTag` already references `ColorSuccess`, `ColorSelected`, `ColorWarning` from `styles.go` — no change needed since styles.go will source from palette.
- `memories.go`: `renderFilterChips` already references `ColorAccent`, `ColorMuted`, `ColorSelected` from `styles.go` — no change needed.

**Step 7 — Glamour re-rendering on theme change:**

`helpers.go` uses `glamour.WithAutoStyle()` which detects dark/light automatically. Since `init()` runs before any rendering, and glamour's auto-style runs on first `NewTermRenderer` call, both systems detect at the same time. The cached renderer in `helpers.go` will correctly reflect the terminal theme. No additional changes needed.

**Step 8 — `forms.go` and `spinner.go`:**

Both already reference `Color*` and `Style*` vars from `theme.go`. After theme.go is rebuilt from `ActivePalette`, these automatically pick up the correct palette. No changes needed to these files.

### Key mapping decisions (Mocha ANSI → Latte hex)

| Old (Mocha ANSI) | Semantic role | Latte hex |
|---|---|---|
| `"86"` cyan-green Accent | Accent | `#179299` Teal |
| `"33"` blue Selected | Selected | `#209fb5` Sapphire |
| `"82"` green Success | Success | `#40a02b` Green |
| `"214"` orange Warning | Warning | `#fe640b` Peach |
| `"196"` red Error | Error | `#d20f39` Red |
| `"241"` dimmed gray Deprecated | Deprecated | `#9ca0b0` Overlay0 |
| `"236"` dark gray Border | Border | `#acb0be` Surface2 |
| `"245"` secondary text Muted | Muted | `#7c7f93` Overlay2 |
| `"240"` very muted Subtle | Subtle | `#8c8fa1` Overlay1 |
| `"63"` purple/indigo Primary | Primary | `#8839ef` Mauve |
| `"242"` inactive tab TabInact | TabInact | `#9ca0b0` Overlay0 |
| `"235"` StatusBg | StatusBg | `#dce0e8` Crust |
| `"252"` ValueText | ValueText | `#4c4f69` Text |
| `"239"` SelectedBg | SelectedBg | `#ccd0da` Surface0 |

## Open Questions

None. All decisions resolved in Direction.

## Observations

- `glamour.WithAutoStyle()` in `helpers.go` already handles dark/light markdown rendering; no extra work needed there.
- `sessions.go` and `memories.go` inline styles both ultimately reference colors from `styles.go`'s vars, so no direct file edits are needed beyond the styles.go refactor.
- The `projectColorPalette` in `crossproject.go` is used for project labeling (aesthetic, not semantic), so a fixed hex palette with good dual-contrast is the right call — not a theme-dependent selection.
- `internal/ui` package-level vars (`ColorGreen`, etc.) are also used in `output.go` PrintBanner inline style — that inline style only sets `Foreground(ColorGreen)` and `.Bold(true)`, so it does not need separate treatment; it can remain as-is since `ColorGreen` will be palette-driven.

## Files to Modify

### `internal/ui/theme.go` (primary refactor)
- Add `Palette` struct with all color slots (semantic + TUI structural).
- Define `MochaPalette` and `LattePalette` instances with full values.
- Add `ActivePalette Palette` var and `init()` that calls `lipgloss.HasDarkBackground()`.
- Rewrite the existing color `var` block to alias from `ActivePalette` fields.
- Rewrite the existing style `var` block to source from the aliased color vars (no logic change, just ensuring palette-driven values).

### `internal/tui/styles.go` (full refactor)
- Add import `"github.com/AngelMaldonado/cx/internal/ui"`.
- Replace all 15 ANSI color constant vars with `ui.ActivePalette.*` field references.
- Replace 4 inline ANSI literals in style var declarations with palette-based color vars:
  - `lipgloss.Color("235")` → `ColorStatusBg` (new var from `ui.ActivePalette.StatusBg`)
  - `lipgloss.Color("252")` in ValueStyle → `ColorValueText` (new var)
  - `lipgloss.Color("252")` in TableRowStyle → `ColorValueText`
  - `lipgloss.Color("239")` in TableSelectedStyle → `ColorSelectedBg` (new var)
  - `lipgloss.Color("252")` in HelpTextStyle → `ColorValueText`

### `internal/tui/graph.go` (inline consolidation)
- Replace the 4 ANSI literals in `relationColors` with vars from `styles.go`:
  - `"33"` → `ColorSelected`
  - `"196"` → `ColorError`
  - `"82"` → `ColorSuccess`
  - `"241"` → `ColorDeprecated`

### `internal/tui/crossproject.go` (inline consolidation)
- Replace the 8 ANSI literals in `projectColorPalette` with fixed Catppuccin hex values that have sufficient contrast on both dark and light backgrounds:
  `"#209fb5"` (Sapphire), `"#8839ef"` (Mauve), `"#40a02b"` (Green), `"#fe640b"` (Peach), `"#dd7878"` (Flamingo), `"#179299"` (Teal), `"#df8e1d"` (Yellow), `"#7287fd"` (Lavender)

### `internal/ui/output.go` (minor)
- The 2 inline `lipgloss.NewStyle()` calls in `PrintItem` (`Foreground(ColorText).Bold(true)` and `Foreground(ColorOverlay)`) are already palette-driven via `ColorText` and `ColorOverlay`. No change needed if those vars alias `ActivePalette`.
- `PrintBanner` inline style uses `Foreground(ColorGreen).Bold(true)` — same, palette-driven. No change needed.

### `internal/ui/forms.go` (no change)
References only `ColorMauve`, `ColorSubtext`, `ColorRed`, `ColorGreen`, `ColorOverlay`, `ColorBase`, `ColorText`, `ColorSurface` — all will be palette-driven after theme.go refactor.

### `internal/ui/spinner.go` (no change)
Uses `StyleAccent` which references `ColorMauve` — palette-driven after refactor.

### `internal/tui/sessions.go` (no change)
`renderModeTag` references `ColorSuccess`, `ColorSelected`, `ColorWarning` — all will come from styles.go after its refactor.

### `internal/tui/memories.go` (no change)
`renderFilterChips` references `ColorAccent`, `ColorMuted`, `ColorSelected` — all from styles.go after refactor.

### `internal/tui/helpers.go` (no change)
`glamour.WithAutoStyle()` handles markdown theme automatically. Cached renderer re-creates when width changes, not on theme — acceptable since theme is detected once at `init()` and does not change mid-session.

## Risks

1. **`lipgloss.HasDarkBackground()` detection accuracy**: This function queries the terminal and can return incorrect results in some environments (e.g., CI, pipes, tmux with unusual configs). The function already falls back to dark (Mocha) if detection fails, which is the safer default since most developer terminals used with cx are dark-themed.
   Mitigation: Document the fallback behavior. If a future need arises for an override, a `CX_THEME=light|dark` env var can be added as an addendum.

2. **`init()` ordering**: Go's `init()` functions in the same package run in source file order. `theme.go`'s `init()` must run before `spinner.go`'s `init()` which pre-renders frames using `StyleAccent`. Go guarantees alphabetical source file ordering within a package, so `spinner.go`'s `init()` runs after `theme.go`'s. But the `styledFrames` in `spinner.go` are initialized as a `var` (not `init()`), which runs before all `init()` functions. The var block uses `StyleAccent.Render(f)`, and `StyleAccent` is a package-level var in `theme.go`. Go initializes vars within a package based on dependency order, so `StyleAccent` will be set before `styledFrames` accesses it — but only if `ActivePalette` is initialized before `StyleAccent`. Since `ActivePalette` will be set in an `init()` function and `StyleAccent` is a package-level var, this creates a sequencing risk.
   Mitigation: Move `styledFrames` initialization into `spinner.go`'s own `init()` function instead of a `var` block, ensuring it runs after all `init()` calls in `theme.go`.

3. **Cross-package init ordering**: `internal/tui/styles.go` vars are initialized at package load time. They reference `ui.ActivePalette`, which is set in `internal/ui`'s `init()`. Go guarantees that imported packages are fully initialized before the importing package's `init()` runs, so `ui.ActivePalette` will be set before `tui/styles.go`'s var block executes.
   Mitigation: This is safe by Go's initialization guarantee. No extra action needed.

4. **Latte contrast validation**: The Latte color assignments are chosen from the Catppuccin Latte spec but have not been visually validated in a running terminal. Some edge cases (e.g., `SelectedBg` `#ccd0da` with `ColorSelected` `#209fb5` foreground) need contrast verification.
   Mitigation: After implementation, test on a real light-background terminal (macOS Terminal "Basic", or set iTerm background to white). Adjust any specific slots that fail contrast.

5. **glamour markdown renderer on light backgrounds**: `glamour.WithAutoStyle()` detects light/dark but uses its own internal theme, not Catppuccin colors. The preview panes may render markdown with slightly different aesthetics than the rest of the TUI.
   Mitigation: Acceptable. glamour's auto theme is well-tested for light terminals. No custom glamour theme is needed.

## Testing

1. **Dark terminal (existing behavior)**: Run `cx tui` and `cx doctor` / `cx brainstorm status` in a dark terminal. Verify colors match the existing Catppuccin Mocha appearance (no regression).

2. **Light terminal (new behavior)**: In a terminal with a light/white background (e.g., macOS Terminal with Basic/Homebrew profile, or iTerm with white background):
   - Run `cx doctor` — verify all status symbols (green check, red X, yellow warning) are readable.
   - Run `cx brainstorm status` — verify headers, items, and muted text are visible.
   - Run `cx tui` — navigate all 8 tabs:
     - Tab bar: active tab distinct from inactive.
     - Memories tab: rows readable, selected row has visible highlight (not dark-on-white).
     - Sessions tab: mode badges (BUILD/PLAN/CONTINUE) distinct colors, visible on light.
     - Graph tab: relation colors distinguishable.
     - Cross-project tab: project color palette readable.
     - Status bar at bottom: not a dark block on white terminal.
   - Run `cx init` in a test directory — verify huh form is readable (mauve titles, green selected options).
   - Run a command with spinner — verify spinner frames are visible (not Mocha mauve on white).

3. **Pipe/non-TTY (fallback)**: Run `cx doctor | cat`. `lipgloss.HasDarkBackground()` should return false (no TTY), defaulting to dark/Mocha palette. Verify no crash and output is still formatted.

4. **`go build ./...`**: Verify the refactored code compiles cleanly.

5. **`go vet ./...`**: Verify no vet issues introduced.

## References

- Catppuccin Latte palette: https://github.com/catppuccin/catppuccin#-palette
- Catppuccin Mocha palette: current values in `internal/ui/theme.go`
- `lipgloss.HasDarkBackground()`: https://pkg.go.dev/github.com/charmbracelet/lipgloss#HasDarkBackground
- glamour WithAutoStyle: https://pkg.go.dev/github.com/charmbracelet/glamour#WithAutoStyle
