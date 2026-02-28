package dataview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// KeyMap defines dataview-specific keybindings.
type KeyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	PageDown key.Binding
	PageUp   key.Binding
	Home     key.Binding
	End      key.Binding
}

// DefaultKeyMap returns dataview key bindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "next page"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "prev page"),
		),
		Home: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home", "first row"),
		),
		End: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "last row"),
		),
	}
}

// Model represents the data viewer state.
type Model struct {
	columns   []string
	rows      [][]string
	colWidths []int

	// Viewport cursor
	cursorRow int
	cursorCol int
	scrollRow int
	scrollCol int

	tableName  string
	page       int
	totalRows  int64
	pageSize   int
	focused    bool
	width      int
	height     int
	keyMap     KeyMap
}

// New creates a new data view model.
func New() Model {
	return Model{
		pageSize: 100,
		keyMap:   DefaultKeyMap(),
	}
}

// SetData loads new query results into the grid.
func (m *Model) SetData(tableName string, columns []string, rows [][]string) {
	m.tableName = tableName
	m.columns = columns
	m.rows = rows
	m.cursorRow = 0
	m.cursorCol = 0
	m.scrollRow = 0
	m.scrollCol = 0
	m.colWidths = calculateColWidths(columns, rows)
}

// SetPage sets the current page info.
func (m *Model) SetPage(page int, totalRows int64) {
	m.page = page
	m.totalRows = totalRows
}

// SetFocused sets focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

// SetSize sets the component dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// PageSize returns the configured page size.
func (m Model) PageSize() int {
	return m.pageSize
}

// Page returns the current page number.
func (m Model) Page() int {
	return m.page
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles input events.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focused || len(m.rows) == 0 {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keyMap.Up):
			if m.cursorRow > 0 {
				m.cursorRow--
				m.adjustVerticalScroll()
			}
		case key.Matches(msg, m.keyMap.Down):
			if m.cursorRow < len(m.rows)-1 {
				m.cursorRow++
				m.adjustVerticalScroll()
			}
		case key.Matches(msg, m.keyMap.Left):
			if m.cursorCol > 0 {
				m.cursorCol--
				m.adjustHorizontalScroll()
			}
		case key.Matches(msg, m.keyMap.Right):
			if m.cursorCol < len(m.columns)-1 {
				m.cursorCol++
				m.adjustHorizontalScroll()
			}
		case key.Matches(msg, m.keyMap.Home):
			m.cursorRow = 0
			m.scrollRow = 0
		case key.Matches(msg, m.keyMap.End):
			m.cursorRow = len(m.rows) - 1
			m.adjustVerticalScroll()
		case key.Matches(msg, m.keyMap.PageDown):
			viewportRows := m.viewportRows()
			m.cursorRow += viewportRows
			if m.cursorRow >= len(m.rows) {
				m.cursorRow = len(m.rows) - 1
			}
			m.adjustVerticalScroll()
		case key.Matches(msg, m.keyMap.PageUp):
			viewportRows := m.viewportRows()
			m.cursorRow -= viewportRows
			if m.cursorRow < 0 {
				m.cursorRow = 0
			}
			m.adjustVerticalScroll()
		}
	}

	return m, nil
}

// View renders the data view.
func (m Model) View() string {
	if m.width <= 0 || m.height <= 0 {
		return ""
	}

	contentWidth := m.width - 2
	if contentWidth < 1 {
		contentWidth = 1
	}

	highlight := lipgloss.Color("#7DC4E4")
	subtle := lipgloss.Color("#626262")
	border := lipgloss.Color("#444444")
	focusBorder := lipgloss.Color("#7DC4E4")
	selectedBg := lipgloss.Color("#313244")

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().Foreground(highlight).Bold(true)
	if m.tableName != "" {
		b.WriteString(headerStyle.Render(fmt.Sprintf(" Data View: '%s' table", m.tableName)))
	} else {
		b.WriteString(headerStyle.Render(" Data View"))
	}
	b.WriteString("\n")

	if len(m.columns) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(subtle).Render("  Select a table to view data"))
		return m.applyBorder(b.String(), contentWidth, border, focusBorder)
	}

	// Calculate visible columns
	visibleCols := m.visibleColumns(contentWidth)

	// Column headers
	headerLine := m.renderRow(m.columns, visibleCols, -1, selectedBg, highlight, true)
	b.WriteString(headerLine)
	b.WriteString("\n")

	// Separator
	sepParts := make([]string, 0)
	for _, ci := range visibleCols {
		sepParts = append(sepParts, strings.Repeat("─", m.colWidths[ci]+2))
	}
	b.WriteString(lipgloss.NewStyle().Foreground(subtle).Render(strings.Join(sepParts, "┼")))
	b.WriteString("\n")

	// Data rows
	viewRows := m.viewportRows()
	endRow := m.scrollRow + viewRows
	if endRow > len(m.rows) {
		endRow = len(m.rows)
	}

	for i := m.scrollRow; i < endRow; i++ {
		line := m.renderRow(m.rows[i], visibleCols, i, selectedBg, highlight, false)
		b.WriteString(line)
		if i < endRow-1 {
			b.WriteString("\n")
		}
	}

	// Pagination footer
	b.WriteString("\n")
	totalPages := 1
	if m.totalRows > 0 {
		totalPages = int((m.totalRows + int64(m.pageSize) - 1) / int64(m.pageSize))
	}
	pageInfo := fmt.Sprintf(" %d rows", len(m.rows))
	if m.totalRows > 0 {
		pageInfo = fmt.Sprintf(" %d/%d", m.page+1, totalPages)
	}

	footerStyle := lipgloss.NewStyle().Foreground(subtle)
	padLen := contentWidth - lipgloss.Width(pageInfo)
	if padLen < 0 {
		padLen = 0
	}
	b.WriteString(footerStyle.Render(strings.Repeat(" ", padLen) + pageInfo))

	return m.applyBorder(b.String(), contentWidth, border, focusBorder)
}

func (m Model) renderRow(cells []string, visibleCols []int, rowIdx int, selectedBg, highlight lipgloss.Color, isHeader bool) string {
	var parts []string
	for _, ci := range visibleCols {
		val := ""
		if ci < len(cells) {
			val = truncate(cells[ci], m.colWidths[ci])
		}
		// Pad to column width
		padded := val + strings.Repeat(" ", m.colWidths[ci]-runeWidth(val))
		cell := " " + padded + " "

		if isHeader {
			cell = lipgloss.NewStyle().Bold(true).Foreground(highlight).Render(cell)
		} else if rowIdx == m.cursorRow && m.focused {
			cell = lipgloss.NewStyle().Background(selectedBg).Bold(true).Render(cell)
		} else if cells[ci] == "<NULL>" {
			cell = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Italic(true).Render(cell)
		}

		parts = append(parts, cell)
	}
	sep := "│"
	if !isHeader {
		sep = lipgloss.NewStyle().Foreground(lipgloss.Color("#444444")).Render("│")
	}
	return strings.Join(parts, sep)
}

func (m Model) applyBorder(content string, contentWidth int, border, focusBorder lipgloss.Color) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Width(contentWidth).
		Height(m.height - 2)

	if m.focused {
		style = style.BorderForeground(focusBorder)
	}

	return style.Render(content)
}

func (m Model) visibleColumns(maxWidth int) []int {
	if len(m.colWidths) == 0 {
		return nil
	}

	var cols []int
	usedWidth := 0
	for i := m.scrollCol; i < len(m.colWidths); i++ {
		needed := m.colWidths[i] + 3 // cell padding + separator
		if usedWidth+needed > maxWidth && len(cols) > 0 {
			break
		}
		cols = append(cols, i)
		usedWidth += needed
	}
	return cols
}

func (m Model) viewportRows() int {
	// header(1) + separator(1) + footer(1) + border(2) + header text(1)
	rows := m.height - 7
	if rows < 1 {
		return 1
	}
	return rows
}

func (m *Model) adjustVerticalScroll() {
	viewRows := m.viewportRows()
	if m.cursorRow < m.scrollRow {
		m.scrollRow = m.cursorRow
	}
	if m.cursorRow >= m.scrollRow+viewRows {
		m.scrollRow = m.cursorRow - viewRows + 1
	}
}

func (m *Model) adjustHorizontalScroll() {
	if m.cursorCol < m.scrollCol {
		m.scrollCol = m.cursorCol
	}
	// Simple: ensure the cursor column is visible
	if m.cursorCol > m.scrollCol+5 {
		m.scrollCol = m.cursorCol - 3
	}
}

func calculateColWidths(columns []string, rows [][]string) []int {
	if len(columns) == 0 {
		return nil
	}

	widths := make([]int, len(columns))

	// Start with header widths
	for i, col := range columns {
		widths[i] = runeWidth(col)
	}

	// Check first 50 rows for max widths
	checkRows := len(rows)
	if checkRows > 50 {
		checkRows = 50
	}
	for _, row := range rows[:checkRows] {
		for i, cell := range row {
			if i < len(widths) {
				w := runeWidth(cell)
				if w > widths[i] {
					widths[i] = w
				}
			}
		}
	}

	// Clamp between min and max
	for i := range widths {
		if widths[i] < 6 {
			widths[i] = 6
		}
		if widths[i] > 30 {
			widths[i] = 30
		}
	}

	return widths
}

func truncate(s string, maxLen int) string {
	if runeWidth(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	// Truncate rune-aware
	runes := []rune(s)
	w := 0
	for i, r := range runes {
		w += charWidth(r)
		if w > maxLen-3 {
			return string(runes[:i]) + "..."
		}
	}
	return s
}

func runeWidth(s string) int {
	w := 0
	for _, r := range s {
		w += charWidth(r)
	}
	return w
}

func charWidth(r rune) int {
	// Simplified: CJK and fullwidth chars take 2 cells
	if r >= 0x1100 &&
		(r <= 0x115f || r == 0x2329 || r == 0x232a ||
			(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
			(r >= 0xac00 && r <= 0xd7a3) ||
			(r >= 0xf900 && r <= 0xfaff) ||
			(r >= 0xfe10 && r <= 0xfe19) ||
			(r >= 0xfe30 && r <= 0xfe6f) ||
			(r >= 0xff00 && r <= 0xff60) ||
			(r >= 0xffe0 && r <= 0xffe6) ||
			(r >= 0x20000 && r <= 0x2fffd) ||
			(r >= 0x30000 && r <= 0x3fffd)) {
		return 2
	}
	return 1
}
