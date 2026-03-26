// Package tui implements the cx dashboard TUI using Bubble Tea.
// helpers.go contains shared utility functions used across multiple view files.
// All functions are unexported and live in the tui package.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// cachedRenderer is a package-level glamour renderer shared across all views.
// It is invalidated when the terminal width changes.
var (
	cachedRenderer      *glamour.TermRenderer
	cachedRendererWidth int
)

// getRenderer returns a cached glamour.TermRenderer for the given width.
// A new renderer is allocated only when width changes or the cache is empty.
func getRenderer(width int) *glamour.TermRenderer {
	if cachedRenderer != nil && cachedRendererWidth == width {
		return cachedRenderer
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return nil
	}
	cachedRenderer = r
	cachedRendererWidth = width
	return r
}

// padRight pads s on the right to exactly width runes (using displayWidth).
// If s is longer than width it is truncated.
func padRight(s string, width int) string {
	// Use lipgloss.Width to handle ANSI escape codes.
	w := lipgloss.Width(s)
	if w >= width {
		return truncateANSI(s, width)
	}
	return s + strings.Repeat(" ", width-w)
}

// truncate shortens s to at most maxLen visible characters, adding "…" if needed.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

// truncateANSI truncates a string (which may contain ANSI codes) so that its
// visible width is at most maxWidth.  It strips all ANSI escape sequences,
// then re-renders a plain substring.  This is a best-effort fallback — callers
// should avoid producing strings wider than the terminal in the first place.
func truncateANSI(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	if w <= maxWidth {
		return s
	}
	// Strip ANSI and truncate the plain text.
	// lipgloss.Width handles ANSI stripping; we rebuild from rune count.
	runes := []rune(s)
	for end := len(runes); end > 0; end-- {
		candidate := string(runes[:end])
		if lipgloss.Width(candidate) <= maxWidth {
			return candidate
		}
	}
	return ""
}

// formatTimestamp shortens an RFC3339 / datetime string to "YYYY-MM-DD HH:MM".
func formatTimestamp(ts string) string {
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t.Local().Format("2006-01-02 15:04")
	}
	if len(ts) >= 16 {
		// Replace the T separator with a space for readability.
		s := ts[:16]
		if len(s) > 10 && s[10] == 'T' {
			s = s[:10] + " " + s[11:]
		}
		return s
	}
	return ts
}

// formatDurationMs converts milliseconds to a human-readable string like "2.3s"
// or "1m 5s" for longer durations.
func formatDurationMs(ms int) string {
	if ms <= 0 {
		return "—"
	}
	totalSec := ms / 1000
	remainMs := ms % 1000
	if totalSec < 60 {
		// e.g. "2.3s"
		return fmt.Sprintf("%d.%ds", totalSec, remainMs/100)
	}
	mins := totalSec / 60
	secs := totalSec % 60
	return fmt.Sprintf("%dm %ds", mins, secs)
}

// wrapText wraps s to at most maxWidth characters per line, breaking at spaces.
func wrapText(s string, maxWidth int) []string {
	if maxWidth <= 0 {
		return []string{s}
	}
	var lines []string
	words := strings.Fields(s)
	current := ""
	for _, w := range words {
		if current == "" {
			current = w
		} else if len(current)+1+len(w) <= maxWidth {
			current += " " + w
		} else {
			lines = append(lines, current)
			current = w
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	if len(lines) == 0 {
		return []string{""}
	}
	return lines
}

// renderGlamour renders markdown with glamour, falling back to plain text on error.
func renderGlamour(content string, width int) string {
	if content == "" {
		return ""
	}
	r := getRenderer(width)
	if r == nil {
		return content
	}
	rendered, err := r.Render(content)
	if err != nil {
		return content
	}
	return rendered
}
