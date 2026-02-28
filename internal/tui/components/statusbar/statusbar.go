package statusbar

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Model represents the status bar state.
type Model struct {
	dbName    string
	user      string
	host      string
	queryTime time.Duration
	rowCount  int
	width     int
}

// New creates a new status bar.
func New(dbName, user, host string) Model {
	return Model{
		dbName: dbName,
		user:   user,
		host:   host,
	}
}

// SetQueryInfo updates the last query stats.
func (m *Model) SetQueryInfo(duration time.Duration, rowCount int) {
	m.queryTime = duration
	m.rowCount = rowCount
}

// SetWidth sets the status bar width.
func (m *Model) SetWidth(width int) {
	m.width = width
}

// View renders the status bar.
func (m Model) View() string {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7DC4E4")).
		Bold(true)

	valStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CDD6F4"))

	sep := valStyle.Render(" | ")

	left := keyStyle.Render("db: ") + valStyle.Render(m.dbName) +
		sep +
		keyStyle.Render("User: ") + valStyle.Render(m.user)

	if m.queryTime > 0 {
		left += sep + keyStyle.Render("Query Time: ") + valStyle.Render(formatDuration(m.queryTime))
		left += sep + keyStyle.Render("Rows: ") + valStyle.Render(fmt.Sprintf("%d", m.rowCount))
	}

	right := valStyle.Render("help: ") + keyStyle.Render("F1")

	barStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#1E1E2E")).
		Width(m.width)

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

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%dμs", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
