package ui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

func CXTheme() *huh.Theme {
	t := huh.ThemeBase()

	// Focused field styles
	t.Focused.Title = t.Focused.Title.Foreground(ColorMauve)
	t.Focused.Description = t.Focused.Description.Foreground(ColorSubtext)
	t.Focused.ErrorIndicator = t.Focused.ErrorIndicator.Foreground(ColorRed)
	t.Focused.ErrorMessage = t.Focused.ErrorMessage.Foreground(ColorRed)
	t.Focused.SelectSelector = t.Focused.SelectSelector.Foreground(ColorMauve)
	t.Focused.MultiSelectSelector = t.Focused.MultiSelectSelector.Foreground(ColorMauve)
	t.Focused.SelectedOption = t.Focused.SelectedOption.Foreground(ColorGreen)
	t.Focused.SelectedPrefix = lipgloss.NewStyle().Foreground(ColorGreen).SetString("[x] ")
	t.Focused.UnselectedOption = t.Focused.UnselectedOption.Foreground(ColorOverlay)
	t.Focused.UnselectedPrefix = lipgloss.NewStyle().Foreground(ColorOverlay).SetString("[ ] ")
	t.Focused.FocusedButton = t.Focused.FocusedButton.Foreground(ColorBase).Background(ColorMauve)
	t.Focused.BlurredButton = t.Focused.BlurredButton.Foreground(ColorText).Background(ColorSurface)
	t.Focused.TextInput.Cursor = t.Focused.TextInput.Cursor.Foreground(ColorMauve)
	t.Focused.TextInput.Placeholder = t.Focused.TextInput.Placeholder.Foreground(ColorOverlay)

	// Blurred field styles
	t.Blurred.Title = t.Blurred.Title.Foreground(ColorOverlay)
	t.Blurred.Description = t.Blurred.Description.Foreground(ColorOverlay)
	t.Blurred.SelectedOption = t.Blurred.SelectedOption.Foreground(ColorSubtext)
	t.Blurred.UnselectedOption = t.Blurred.UnselectedOption.Foreground(ColorOverlay)

	return t
}

func NewAgentSelect() ([]string, error) {
	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Which AI agents will you use?").
				Description("Skills and configs will be installed for each agent.").
				Options(
					huh.NewOption("Claude Code", "claude").Selected(true),
					huh.NewOption("Gemini CLI", "gemini"),
					huh.NewOption("Codex CLI", "codex"),
				).
				Value(&selected),
		),
	).WithTheme(CXTheme())

	err := form.Run()
	if err != nil {
		return nil, handleFormError(err)
	}
	return selected, nil
}

func NewProjectTypeSelect() (string, error) {
	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("What type of project is this?").
				Description("This shapes the DIRECTION.md guidance for your AI agents.").
				Options(
					huh.NewOption("Web API", "web-api"),
					huh.NewOption("Frontend", "frontend"),
					huh.NewOption("Firmware", "firmware"),
					huh.NewOption("CLI Tool", "cli"),
					huh.NewOption("Full-stack", "full-stack"),
					huh.NewOption("Other", "other"),
				).
				Value(&selected),
		),
	).WithTheme(CXTheme())

	err := form.Run()
	if err != nil {
		return "", handleFormError(err)
	}
	return selected, nil
}

func NewPrioritiesSelect() ([]string, error) {
	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("What are your top priorities?").
				Description("Pick up to 3. These add specific guidance to DIRECTION.md.").
				Options(
					huh.NewOption("Performance", "performance"),
					huh.NewOption("External systems", "external-systems"),
					huh.NewOption("Security", "security"),
					huh.NewOption("UX polish", "ux"),
					huh.NewOption("Data model", "data-model"),
					huh.NewOption("Infrastructure", "infrastructure"),
					huh.NewOption("Integration", "integration"),
				).
				Limit(3).
				Value(&selected),
		),
	).WithTheme(CXTheme())

	err := form.Run()
	if err != nil {
		return nil, handleFormError(err)
	}
	return selected, nil
}

func NewConfirmPrompt(title string) (bool, error) {
	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(title).
				Affirmative("Yes").
				Negative("No").
				Value(&confirmed),
		),
	).WithTheme(CXTheme())

	err := form.Run()
	if err != nil {
		return false, handleFormError(err)
	}
	return confirmed, nil
}

func handleFormError(err error) error {
	if err == huh.ErrUserAborted {
		fmt.Println()
		PrintMuted("Aborted.")
		os.Exit(0)
	}
	return err
}
