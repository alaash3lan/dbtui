package tui

import "github.com/charmbracelet/lipgloss"

// Theme holds all colors used across the TUI.
type Theme struct {
	Name string

	// Primary colors
	Highlight  lipgloss.Color
	Subtle     lipgloss.Color
	Text       lipgloss.Color
	Background lipgloss.Color

	// Borders
	Border      lipgloss.Color
	FocusBorder lipgloss.Color

	// Status bar / title bar
	HeaderBg lipgloss.Color

	// Interactive
	ActiveBg     lipgloss.Color
	SelectedBg   lipgloss.Color
	ErrorColor   lipgloss.Color
	SuccessColor lipgloss.Color
	WarningColor lipgloss.Color

	// SQL syntax highlighting
	KeywordColor lipgloss.Color
	StringColor  lipgloss.Color
	NumberColor  lipgloss.Color
}

// DarkTheme returns the dark color scheme.
func DarkTheme() Theme {
	return Theme{
		Name:         "dark",
		Highlight:    lipgloss.Color("#7DC4E4"),
		Subtle:       lipgloss.Color("#626262"),
		Text:         lipgloss.Color("#CDD6F4"),
		Background:   lipgloss.Color("#1E1E2E"),
		Border:       lipgloss.Color("#444444"),
		FocusBorder:  lipgloss.Color("#7DC4E4"),
		HeaderBg:     lipgloss.Color("#1E1E2E"),
		ActiveBg:     lipgloss.Color("#313244"),
		SelectedBg:   lipgloss.Color("#313244"),
		ErrorColor:   lipgloss.Color("#F38BA8"),
		SuccessColor: lipgloss.Color("#A6E3A1"),
		WarningColor: lipgloss.Color("#F9E2AF"),
		KeywordColor: lipgloss.Color("#CBA6F7"),
		StringColor:  lipgloss.Color("#A6E3A1"),
		NumberColor:  lipgloss.Color("#FAB387"),
	}
}

// LightTheme returns the light color scheme.
func LightTheme() Theme {
	return Theme{
		Name:         "light",
		Highlight:    lipgloss.Color("#1E66F5"),
		Subtle:       lipgloss.Color("#9CA0B0"),
		Text:         lipgloss.Color("#4C4F69"),
		Background:   lipgloss.Color("#EFF1F5"),
		Border:       lipgloss.Color("#BCC0CC"),
		FocusBorder:  lipgloss.Color("#1E66F5"),
		HeaderBg:     lipgloss.Color("#DCE0E8"),
		ActiveBg:     lipgloss.Color("#CCD0DA"),
		SelectedBg:   lipgloss.Color("#CCD0DA"),
		ErrorColor:   lipgloss.Color("#D20F39"),
		SuccessColor: lipgloss.Color("#40A02B"),
		WarningColor: lipgloss.Color("#DF8E1D"),
		KeywordColor: lipgloss.Color("#8839EF"),
		StringColor:  lipgloss.Color("#40A02B"),
		NumberColor:  lipgloss.Color("#FE640B"),
	}
}
