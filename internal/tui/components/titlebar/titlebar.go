package titlebar

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Colors holds the theme colors used by the title bar.
type Colors struct {
	Highlight  lipgloss.Color
	Text       lipgloss.Color
	Background lipgloss.Color
}

// Model represents the title bar state.
type Model struct {
	version  string
	rowCount int
	width    int
	colors   Colors
}

// New creates a new title bar.
func New(version string) Model {
	return Model{version: version}
}

// SetRowCount updates the displayed row count.
func (m *Model) SetRowCount(count int) {
	m.rowCount = count
}

// SetWidth sets the title bar width.
func (m *Model) SetWidth(width int) {
	m.width = width
}

// SetColors updates the theme colors.
func (m *Model) SetColors(c Colors) {
	m.colors = c
}

// View renders the title bar.
func (m Model) View() string {
	c := m.colors
	titleStyle := lipgloss.NewStyle().Foreground(c.Highlight).Bold(true)
	countStyle := lipgloss.NewStyle().Foreground(c.Text)
	barStyle := lipgloss.NewStyle().Background(c.Background).Width(m.width)

	left := titleStyle.Render(fmt.Sprintf(" dbplus v%s", m.version))
	right := ""
	if m.rowCount > 0 {
		right = countStyle.Render(fmt.Sprintf("%d ", m.rowCount))
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}

	padding := ""
	for i := 0; i < gap; i++ {
		padding += " "
	}

	return barStyle.Render(left + padding + right)
}
