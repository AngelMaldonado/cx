package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func PrintSuccess(msg string) {
	fmt.Printf("    %s %s\n", SymbolSuccess, StyleSuccess.Render(msg))
}

func PrintWarning(msg string) {
	fmt.Printf("    %s %s\n", SymbolWarning, StyleWarning.Render(msg))
}

func PrintError(msg string) {
	fmt.Printf("    %s %s\n", SymbolError, StyleError.Render(msg))
}

func PrintHeader(title string) {
	fmt.Printf("\n  %s\n", StyleHeader.Render(title))
}

func PrintItem(key, value string) {
	keyStyle := lipgloss.NewStyle().Foreground(ColorText).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(ColorOverlay)
	fmt.Printf("    %s %s\n", keyStyle.Render(key), valStyle.Render(value))
}

func PrintDivider() {
	line := StyleMuted.Render(strings.Repeat("─", 48))
	fmt.Printf("  %s\n", line)
}

func PrintSummary(errors, warnings int) {
	parts := []string{}
	if errors > 0 {
		parts = append(parts, StyleError.Render(fmt.Sprintf("%d error%s", errors, plural(errors))))
	}
	if warnings > 0 {
		parts = append(parts, StyleWarning.Render(fmt.Sprintf("%d warning%s", warnings, plural(warnings))))
	}
	if errors == 0 && warnings == 0 {
		parts = append(parts, StyleSuccess.Render("all checks passed"))
	}
	fmt.Printf("\n  %s\n", strings.Join(parts, StyleMuted.Render(", ")))
}

func PrintBanner(msg string) {
	style := lipgloss.NewStyle().Foreground(ColorGreen).Bold(true)
	fmt.Printf("\n  %s %s\n", StyleSuccess.Render("\u2713"), style.Render(msg))
}

func PrintMuted(msg string) {
	fmt.Printf("    %s\n", StyleMuted.Render(msg))
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}
