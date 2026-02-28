package titlebar

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Model represents the title bar state.
type Model struct {
	version  string
	rowCount int
	width    int
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

// View renders the title bar.
func (m Model) View() string {
	titleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7DC4E4")).
		Bold(true)

	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CDD6F4"))

	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1E1E2E")).
		Width(m.width)

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
