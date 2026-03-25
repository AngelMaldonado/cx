---
name: light-theme
type: proposal
---

## Problem

The cx CLI and TUI dashboard are unreadable on light terminal backgrounds. All colors were chosen for Catppuccin Mocha (a dark theme), so nearly every visual element — text, borders, status bars, selected rows, badges — is either invisible or extremely low-contrast when the terminal has a white or near-white background. Users on macOS Terminal with the "Basic" or "Homebrew" profile, iTerm2 with a light background, or any default-light terminal see a broken experience.

The worst offenders: `#cdd6f4` (near-white text on white), `#f9e2af` (cream yellow on white), ANSI `"252"` (near-white value text), ANSI `"235"` (near-black status bar background), and ANSI `"239"` (near-black selected-row background).

## Approach

Introduce a dual-palette system using Catppuccin Mocha (existing dark) and Catppuccin Latte (new light). A single `Palette` struct in `internal/ui/theme.go` defines all color slots for the entire application — both the CLI output layer (`internal/ui`) and the TUI dashboard (`internal/tui`). At startup, `lipgloss.HasDarkBackground()` detects the terminal theme and selects the appropriate palette into a global `ActivePalette`. All existing color and style variables are then rebuilt as aliases of `ActivePalette` fields.

The TUI's `internal/tui/styles.go` is refactored to import `internal/ui` and source all its color vars from `ui.ActivePalette`, eliminating the hardcoded 256-color ANSI constants. Inline color literals scattered in `graph.go` and `crossproject.go` are consolidated into the shared palette system.

## Scope

In scope:
- `internal/ui/theme.go` — add `Palette` struct, `MochaPalette`, `LattePalette`, `ActivePalette`, and `init()` detection; rebuild existing color and style vars as palette aliases
- `internal/tui/styles.go` — replace all 15 ANSI color constant vars and 5 inline ANSI literals with references to `ui.ActivePalette` fields
- `internal/tui/graph.go` — replace 4 inline ANSI literals in `relationColors` with vars from `styles.go`
- `internal/tui/crossproject.go` — replace 8 inline ANSI literals in `projectColorPalette` with fixed Catppuccin hex values with sufficient dual-contrast
- `internal/ui/spinner.go` — move `styledFrames` initialization from a `var` block into an `init()` function (required to avoid init-ordering hazard with palette detection)

Out of scope:
- Custom glamour theme for markdown preview (glamour's `WithAutoStyle()` already handles this automatically)
- A `CX_THEME` environment variable override (can be added as a follow-up if needed)
- Any behavioral changes to CLI commands or TUI navigation

## Affected Specs

This change is entirely internal to the rendering layer — no behavioral specs are affected. It does not modify any user-facing commands, data model, or agent protocols. No delta specs are required.
