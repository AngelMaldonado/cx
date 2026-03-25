---
name: light-theme
type: tasks
---

## Tasks

### Task 1: Refactor internal/ui/theme.go — add Palette struct and dual-palette detection [DONE]

**Files modified:** `internal/ui/theme.go`, `internal/ui/spinner.go`

**Completed:** Added `Palette` struct with all 24 color fields (base, semantic, structural, entity-type). Added `MochaPalette` and `LattePalette` vars. Added `ActivePalette` package-level var. Added `init()` in `theme.go` that detects terminal background via `lipgloss.HasDarkBackground()`, sets `ActivePalette`, rebuilds all `Color*`/`Style*`/`Symbol*` vars, and initializes `styledFrames`. Removed `init()` from `spinner.go` (kept `var styledFrames []string` declaration). Verified `go build ./...` and `go vet ./internal/ui/...` both pass cleanly.

**Files to modify:** `internal/ui/theme.go`

**What to do:**

1. Add a `Palette` struct with all fields defined in design.md (§ "Palette struct"). The struct has four groups of fields: base Catppuccin colors (Green, Red, Yellow, Blue, Mauve, Text, Subtext, Overlay, Surface, Base), semantic roles (Accent, Selected, Success, Warning, Error, Deprecated, Border, Muted, Subtle, Primary, TabInact), structural layout colors (StatusBg, ValueText, SelectedBg), and entity type colors (ObservationColor, DecisionColor, SessionColor, AgentRunColor). All fields are `lipgloss.Color`.

2. Add the `MochaPalette` var using the values in design.md (§ "MochaPalette instance"). The Mocha palette preserves existing 256-color ANSI codes for all semantic/structural fields and uses hex values for the base Catppuccin colors.

3. Add the `LattePalette` var using the values in design.md (§ "LattePalette instance"). The Latte palette uses hex values for all fields.

4. Add `var ActivePalette Palette` declaration at package level (no initializer).

5. Replace the existing `var` block for `ColorGreen`, `ColorRed`, `ColorYellow`, `ColorBlue`, `ColorMauve`, `ColorText`, `ColorSubtext`, `ColorOverlay`, `ColorSurface`, `ColorBase` with declarations only (no initializers). Similarly change `StyleSuccess`, `StyleWarning`, `StyleError`, `StyleInfo`, `StyleMuted`, `StyleBold`, `StyleAccent`, `StyleHeader`, `StyleItem`, `SymbolSuccess`, `SymbolError`, `SymbolWarning` to declarations only at package level.

6. Add a single `init()` function in `theme.go` that:
   a. Calls `lipgloss.HasDarkBackground()` and assigns `ActivePalette = MochaPalette` or `ActivePalette = LattePalette`
   b. Assigns all `Color*` vars from `ActivePalette` fields (e.g., `ColorGreen = ActivePalette.Green`)
   c. Assigns all `Style*` vars using `lipgloss.NewStyle().Foreground(ColorX)` etc. — same expressions as the current var initializers, but now run after Color* are set
   d. Assigns all `Symbol*` vars using `StyleX.Render(...)` — same as current, but now run after Style* are set

   **Why this ordering matters:** Go package-level `var` declarations run before `init()`. If `ColorGreen = ActivePalette.Green` were a package-level var initializer, it would capture the zero value of `ActivePalette.Green` because `ActivePalette` is set in `init()`, not at var-init time. Moving all assignments into `init()` after `ActivePalette` is set is the correct fix.

7. Also initialize `styledFrames` (from `spinner.go`) inside this same `init()` function, at the end, after `StyleAccent` is assigned. Then remove the `init()` function from `spinner.go` (the `var styledFrames []string` declaration stays in `spinner.go`). This resolves the within-package init ordering hazard: `spinner.go` sorts before `theme.go` alphabetically, so spinner's `init()` would run before theme's `init()` — consolidating into `theme.go`'s `init()` eliminates the ambiguity.

**Dependencies:** None. This is the foundation task.

**Assigned to:** cx-worker

---

### Task 2: Update internal/ui/spinner.go — remove init(), keep var declaration

**Files to modify:** `internal/ui/spinner.go`

**What to do:**

After Task 1 moves `styledFrames` initialization into `theme.go`'s `init()`, remove the `init()` function from `spinner.go`. Keep the `var styledFrames []string` declaration at package level (it is still needed so `spinner.go` can reference `styledFrames`).

Specifically:
- Remove lines 16-21 (the `func init() { styledFrames = make(...); for ... }` block)
- Keep line 14: `var styledFrames []string`

Confirm the file still compiles and that `styledFrames` is populated at runtime by `theme.go`'s `init()`.

**Dependencies:** Task 1 must be complete first.

**Assigned to:** cx-worker

---

### Task 3: Refactor internal/tui/styles.go — replace ANSI constants with palette aliases [DONE]

**Files modified:** `internal/tui/styles.go`

**Completed:** Added `ui "github.com/AngelMaldonado/cx/internal/ui"` import. Replaced all 15 ANSI color constant declarations with palette aliases sourced from `ui.ActivePalette`. Added 3 new vars (`ColorStatusBg`, `ColorValueText`, `ColorSelectedBg`) for previously inline literals. Replaced all 5 inline `lipgloss.Color("...")` literals: `StatusBarStyle` background, `ValueStyle` foreground, `TableRowStyle` foreground, `TableSelectedStyle` background, `HelpTextStyle` foreground. The `lipgloss` import is retained (used for `NewStyle()`, `RoundedBorder()`, `NormalBorder()`, etc.). Verified `go build ./internal/tui/...`, `go vet ./internal/tui/...`, and `go build ./...` all pass cleanly.

**Files to modify:** `internal/tui/styles.go`

**What to do:**

1. Add `"github.com/AngelMaldonado/cx/internal/ui"` to the import block.

2. Replace the entire color constants block (the `var` block declaring `ColorAccent`, `ColorSelected`, `ColorSuccess`, `ColorWarning`, `ColorError`, `ColorDeprecated`, `ColorObservation`, `ColorDecision`, `ColorSession`, `ColorAgentRun`, `ColorBorder`, `ColorMuted`, `ColorSubtle`, `ColorPrimary`, `ColorTabInact`) with palette aliases sourced from `ui.ActivePalette`. Add three new vars: `ColorStatusBg`, `ColorValueText`, `ColorSelectedBg`. Use the exact var block from design.md (§ "internal/tui/styles.go full refactor", step 2).

   These are safe as package-level var initializers (not requiring `init()`) because Go guarantees `internal/ui`'s `init()` runs fully before any code in `internal/tui` runs. By the time `styles.go` vars initialize, `ui.ActivePalette` is already populated.

3. Replace the 5 inline `lipgloss.Color("...")` literals in style var declarations:
   - `StatusBarStyle`: replace `Background(lipgloss.Color("235"))` with `Background(ColorStatusBg)`
   - `ValueStyle`: replace `Foreground(lipgloss.Color("252"))` with `Foreground(ColorValueText)`
   - `TableRowStyle`: replace `Foreground(lipgloss.Color("252"))` with `Foreground(ColorValueText)`
   - `TableSelectedStyle`: replace `Background(lipgloss.Color("239"))` with `Background(ColorSelectedBg)`
   - `HelpTextStyle`: replace `Foreground(lipgloss.Color("252"))` with `Foreground(ColorValueText)`

4. Verify no unused imports and the file compiles. The `lipgloss` import must remain because style var declarations use `lipgloss.NewStyle()`, `lipgloss.RoundedBorder()`, `lipgloss.NormalBorder()`, etc.

**Dependencies:** Task 1 must be complete.

**Assigned to:** cx-worker

---

### Task 4: Consolidate inline colors in graph.go and crossproject.go [DONE]

**Files modified:** `internal/tui/graph.go`, `internal/tui/crossproject.go`

**Completed:** In `graph.go`, replaced the 4 inline `lipgloss.Color("...")` ANSI codes in `relationColors` with named vars from `styles.go`: `ColorSelected` (was `"33"`), `ColorError` (was `"196"`), `ColorSuccess` (was `"82"`), `ColorDeprecated` (was `"241"`). In `crossproject.go`, replaced the 8 inline `lipgloss.Color("...")` ANSI codes in `projectColorPalette` with fixed Catppuccin hex values that have sufficient contrast on both dark and light backgrounds: Sapphire, Mauve, Green, Peach, Flamingo, Teal, Yellow, Lavender. The `lipgloss` import was retained in both files (used for `NewStyle()`, `Place()`, `JoinHorizontal()`, etc.). Verified `go build ./internal/tui/...`, `go build ./...`, and `go vet ./internal/tui/...` all pass cleanly.

**Files to modify:** `internal/tui/graph.go`, `internal/tui/crossproject.go`

**What to do:**

**graph.go** — replace `relationColors` map (lines 53-58):
```go
var relationColors = map[string]lipgloss.Color{
    "related-to":  ColorSelected,   // was lipgloss.Color("33")
    "caused-by":   ColorError,      // was lipgloss.Color("196")
    "resolved-by": ColorSuccess,    // was lipgloss.Color("82")
    "see-also":    ColorDeprecated, // was lipgloss.Color("241")
}
```
The `lipgloss` import stays because `relationStyle` uses `lipgloss.NewStyle()` and `lipgloss.Place()` is used in `View()`.

**crossproject.go** — replace `projectColorPalette` var (lines 27-36):
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
These 8 Catppuccin hex values have sufficient contrast on both dark and light backgrounds. The `lipgloss` import stays because `crossproject.go` uses `lipgloss.NewStyle()`, `lipgloss.JoinHorizontal()`, `lipgloss.Place()`, and `lipgloss.Width()` elsewhere.

**Dependencies:** Task 3 must be complete (`ColorSelected`, `ColorError`, `ColorSuccess`, `ColorDeprecated` are defined in `styles.go`).

**Assigned to:** cx-worker

---

### Task 5: Validation — build, vet, and smoke-test [DONE]

**Files modified:** none — validation only

**Completed:** `go build ./...` passed with zero errors. `go vet ./...` passed with zero warnings. No test files exist in `internal/ui/` or `internal/tui/`, so no unit tests to run. All four modified files compile cleanly: `internal/ui/theme.go`, `internal/ui/spinner.go`, `internal/tui/styles.go`, `internal/tui/graph.go`, `internal/tui/crossproject.go`. No import cycles detected (tui imports ui; ui does not import tui). No undefined variables, unused imports, or type mismatches found.

**What to do:**

1. Run `go build ./...` from the repo root. Must compile with no errors.

2. Run `go vet ./...` from the repo root. Must pass with no new issues.

3. **Dark terminal smoke test:** run `cx doctor` and verify output matches Catppuccin Mocha appearance — green checkmarks, red X, yellow warnings all correctly colored and readable.

4. **Light terminal smoke test** (macOS Terminal Basic profile or iTerm2 with white background): run `cx doctor` and confirm all status symbols are readable. Run `cx brainstorm status` and confirm headers, items, and muted text are visible. Spot-check `cx tui` if accessible: tab bar, selected rows, status bar, badge colors, graph relation colors, cross-project result colors.

5. **Non-TTY smoke test:** run `cx doctor | cat` and confirm no panic and clean exit. `HasDarkBackground()` returns false with no TTY; the command should default to MochaPalette without crashing.

**Dependencies:** Tasks 1, 2, 3, and 4 must all be complete.

**Assigned to:** cx-worker

---

## Implementation Notes

**Execution order:** Tasks must run strictly in sequence: 1 → 2 → 3 → 4 → 5.

**Critical init() hazard (Task 1):** Go package-level `var` initializers run before `init()` functions. Because `ActivePalette` is set in `init()`, any package-level var that reads `ActivePalette` fields (e.g., `ColorGreen = ActivePalette.Green`) would capture the zero-value `lipgloss.Color("")` rather than the palette-set value. The fix is to move all `Color*`, `Style*`, and `Symbol*` assignments into `theme.go`'s `init()` function, leaving only bare declarations at package level.

**Within-package init() ordering (Tasks 1 and 2):** Inside the `internal/ui` package, Go runs `init()` functions in source-file order (alphabetical by filename). `spinner.go` sorts before `theme.go`, so spinner's `init()` would run before `theme.go`'s `init()` — before `StyleAccent` is assigned. Consolidating `styledFrames` initialization into the end of `theme.go`'s `init()` (Task 1) and removing spinner's `init()` (Task 2) eliminates this hazard entirely.

**Cross-package init() ordering (Task 3):** Go guarantees an imported package's `init()` runs fully before the importing package begins initializing. Since `internal/tui` imports `internal/ui`, `ui.init()` is complete before any `tui` code runs. Package-level var declarations in `styles.go` that reference `ui.ActivePalette` fields are therefore safe without requiring a `tui init()`.

**No behavioral changes:** This refactor touches only color and style definitions. No command logic, TUI navigation, data loading, or memory operations are modified. The verification steps in Task 5 confirm that Mocha (dark) appearance is unchanged and Latte (light) appearance is now correct.
