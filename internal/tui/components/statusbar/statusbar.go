package statusbar

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Colors holds the theme colors used by the status bar.
type Colors struct {
	Highlight  lipgloss.Color
	Text       lipgloss.Color
	Background lipgloss.Color
}

// Model represents the status bar state.
type Model struct {
	dbName    string
	user      string
	host      string
	queryTime time.Duration
	rowCount  int
	width     int
	colors    Colors
}

// New creates a new status bar.
func New(dbName, user, host string) Model {
	return Model{
		dbName: dbName,
		user:   user,
		host:   host,
	}
}

// SetDBName updates the displayed database name.
func (m *Model) SetDBName(name string) {
	m.dbName = name
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

// SetColors updates the theme colors.
func (m *Model) SetColors(c Colors) {
	m.colors = c
}

// View renders the status bar.
func (m Model) View() string {
	c := m.colors
	keyStyle := lipgloss.NewStyle().Foreground(c.Highlight).Bold(true)
	valStyle := lipgloss.NewStyle().Foreground(c.Text)
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
		Background(c.Background).
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
		return fmt.Sprintf("%dus", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
