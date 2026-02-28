package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all lipgloss styles used across the TUI.
type Styles struct {
	// Layout borders
	SidebarBorder  lipgloss.Style
	ContentBorder  lipgloss.Style
	FocusedBorder  lipgloss.Style

	// Title bar
	TitleBar       lipgloss.Style
	TitleText      lipgloss.Style

	// Status bar
	StatusBar      lipgloss.Style
	StatusKey      lipgloss.Style
	StatusValue    lipgloss.Style

	// Sidebar
	SidebarHeader  lipgloss.Style
	SidebarItem    lipgloss.Style
	SidebarActive  lipgloss.Style

	// Data view
	DataHeader     lipgloss.Style
	TableHeader    lipgloss.Style
	TableCell      lipgloss.Style
	TableSelected  lipgloss.Style

	// General
	Dimmed         lipgloss.Style
	Error          lipgloss.Style
}

// DefaultStyles returns the dark theme styles.
func DefaultStyles() Styles {
	subtle := lipgloss.Color("#626262")
	highlight := lipgloss.Color("#7DC4E4")
	border := lipgloss.Color("#444444")
	focusBorder := lipgloss.Color("#7DC4E4")
	headerBg := lipgloss.Color("#1E1E2E")
	statusBg := lipgloss.Color("#1E1E2E")
	activeBg := lipgloss.Color("#313244")

	return Styles{
		SidebarBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border),

		ContentBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(border),

		FocusedBorder: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(focusBorder),

		TitleBar: lipgloss.NewStyle().
			Background(headerBg).
			Padding(0, 1).
			Bold(true),

		TitleText: lipgloss.NewStyle().
			Foreground(highlight).
			Bold(true),

		StatusBar: lipgloss.NewStyle().
			Background(statusBg).
			Padding(0, 1),

		StatusKey: lipgloss.NewStyle().
			Foreground(highlight).
			Bold(true),

		StatusValue: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#CDD6F4")),

		SidebarHeader: lipgloss.NewStyle().
			Foreground(highlight).
			Bold(true).
			Padding(0, 1),

		SidebarItem: lipgloss.NewStyle().
			Padding(0, 2),

		SidebarActive: lipgloss.NewStyle().
			Background(activeBg).
			Foreground(highlight).
			Bold(true).
			Padding(0, 2),

		DataHeader: lipgloss.NewStyle().
			Foreground(highlight).
			Bold(true).
			Padding(0, 1),

		TableHeader: lipgloss.NewStyle().
			Bold(true).
			Foreground(highlight).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(subtle),

		TableCell: lipgloss.NewStyle().
			Padding(0, 1),

		TableSelected: lipgloss.NewStyle().
			Background(activeBg).
			Bold(true),

		Dimmed: lipgloss.NewStyle().
			Foreground(subtle),

		Error: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F38BA8")).
			Bold(true),
	}
}
