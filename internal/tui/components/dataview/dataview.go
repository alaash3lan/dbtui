package dataview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
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
	Filter   key.Binding
	ClearFilter key.Binding
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
		Filter: key.NewBinding(
			key.WithKeys("/", "ctrl+f"),
			key.WithHelp("/ ctrl+f", "filter"),
		),
		ClearFilter: key.NewBinding(
			key.WithKeys("escape"),
			key.WithHelp("esc", "clear filter"),
		),
	}
}

// Colors holds the theme colors used by the data view.
type Colors struct {
	Highlight    lipgloss.Color
	Subtle       lipgloss.Color
	Border       lipgloss.Color
	FocusBorder  lipgloss.Color
	SelectedBg   lipgloss.Color
	WarningColor lipgloss.Color
}

// DefaultColors returns dark theme colors.
func DefaultColors() Colors {
	return Colors{
		Highlight:    lipgloss.Color("#7DC4E4"),
		Subtle:       lipgloss.Color("#626262"),
		Border:       lipgloss.Color("#444444"),
		FocusBorder:  lipgloss.Color("#7DC4E4"),
		SelectedBg:   lipgloss.Color("#313244"),
		WarningColor: lipgloss.Color("#F9E2AF"),
	}
}

// Model represents the data viewer state.
type Model struct {
	columns      []string
	allRows      [][]string // unfiltered data
	rows         [][]string // currently displayed (filtered or all)
	colWidths    []int

	// Viewport cursor
	cursorRow int
	cursorCol int
	scrollRow int
	scrollCol int

	// Filter
	filterInput  textinput.Model
	filterActive bool // true when typing in filter
	filterText   string

	tableName  string
	page       int
	totalRows  int64
	pageSize   int
	focused    bool
	width      int
	height     int
	keyMap     KeyMap
	colors     Colors
}

// New creates a new data view model.
func New() Model {
	fi := textinput.New()
	fi.Placeholder = "column | value  or  value"
	fi.Prompt = "Filter: "
	fi.CharLimit = 256

	return Model{
		pageSize:    100,
		keyMap:      DefaultKeyMap(),
		filterInput: fi,
		colors:      DefaultColors(),
	}
}

// SetData loads new query results into the grid.
func (m *Model) SetData(tableName string, columns []string, rows [][]string) {
	m.tableName = tableName
	m.columns = columns
	m.allRows = rows
	m.cursorRow = 0
	m.cursorCol = 0
	m.scrollRow = 0
	m.scrollCol = 0
	m.applyFilter()
	m.colWidths = calculateColWidths(columns, m.rows)
}

// SetPage sets the current page info.
func (m *Model) SetPage(page int, totalRows int64) {
	m.page = page
	m.totalRows = totalRows
}

// SetFocused sets focus state.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
	if !focused {
		m.filterActive = false
		m.filterInput.Blur()
	}
}

// SetSize sets the component dimensions.
func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.filterInput.Width = width - 14 // account for prompt + border
}

// SetColors updates the theme colors.
func (m *Model) SetColors(c Colors) {
	m.colors = c
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
	if !m.focused {
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// When filter input is active, route keys there
		if m.filterActive {
			switch {
			case key.Matches(msg, m.keyMap.ClearFilter):
				m.filterActive = false
				m.filterInput.Blur()
				if m.filterText != "" {
					// Clear the filter
					m.filterInput.SetValue("")
					m.filterText = ""
					m.applyFilter()
				}
				return m, nil
			case msg.Type == tea.KeyEnter:
				// Confirm filter, exit filter mode
				m.filterText = m.filterInput.Value()
				m.filterActive = false
				m.filterInput.Blur()
				m.applyFilter()
				return m, nil
			default:
				var cmd tea.Cmd
				m.filterInput, cmd = m.filterInput.Update(msg)
				// Live filtering as user types
				m.filterText = m.filterInput.Value()
				m.applyFilter()
				return m, cmd
			}
		}

		// Grid navigation mode
		switch {
		case key.Matches(msg, m.keyMap.Filter):
			m.filterActive = true
			return m, m.filterInput.Focus()
		case key.Matches(msg, m.keyMap.ClearFilter):
			// Escape in grid mode clears filter if active
			if m.filterText != "" {
				m.filterInput.SetValue("")
				m.filterText = ""
				m.applyFilter()
				return m, nil
			}
		case key.Matches(msg, m.keyMap.Up):
			if m.cursorRow > 0 {
				m.cursorRow--
				m.adjustVerticalScroll()
			}
		case key.Matches(msg, m.keyMap.Down):
			if len(m.rows) > 0 && m.cursorRow < len(m.rows)-1 {
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
			if len(m.rows) > 0 {
				m.cursorRow = len(m.rows) - 1
				m.adjustVerticalScroll()
			}
		case key.Matches(msg, m.keyMap.PageDown):
			viewportRows := m.viewportRows()
			m.cursorRow += viewportRows
			if len(m.rows) > 0 && m.cursorRow >= len(m.rows) {
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

func (m *Model) applyFilter() {
	f := ParseFilter(m.filterText)
	if f.Value == "" {
		m.rows = m.allRows
	} else {
		m.rows = ApplyFilter(m.columns, m.allRows, f)
	}

	// Reset cursor if out of bounds
	if m.cursorRow >= len(m.rows) {
		if len(m.rows) > 0 {
			m.cursorRow = len(m.rows) - 1
		} else {
			m.cursorRow = 0
		}
	}
	m.scrollRow = 0
	m.colWidths = calculateColWidths(m.columns, m.rows)
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

	c := m.colors
	highlight := c.Highlight
	subtle := c.Subtle
	border := c.Border
	focusBorder := c.FocusBorder
	selectedBg := c.SelectedBg
	filterColor := c.WarningColor

	var b strings.Builder

	// Header
	headerStyle := lipgloss.NewStyle().Foreground(highlight).Bold(true)
	if m.tableName != "" {
		b.WriteString(headerStyle.Render(fmt.Sprintf(" Data View: '%s' table", m.tableName)))
	} else {
		b.WriteString(headerStyle.Render(" Data View"))
	}
	b.WriteString("\n")

	// Filter bar (always visible)
	if m.filterActive {
		b.WriteString(m.filterInput.View())
	} else if m.filterText != "" {
		f := ParseFilter(m.filterText)
		filterDisplay := ""
		if f.Column != "" {
			filterDisplay = fmt.Sprintf("[%s | %s]", f.Column, f.Value)
		} else {
			filterDisplay = fmt.Sprintf("[%s]", f.Value)
		}
		filterStyle := lipgloss.NewStyle().Foreground(filterColor)
		labelStyle := lipgloss.NewStyle().Foreground(subtle)
		b.WriteString(labelStyle.Render(" Filter: ") + filterStyle.Render(filterDisplay))
		matchInfo := fmt.Sprintf("  %d/%d", len(m.rows), len(m.allRows))
		b.WriteString(lipgloss.NewStyle().Foreground(subtle).Render(matchInfo))
	} else {
		b.WriteString(lipgloss.NewStyle().Foreground(subtle).Render(" / to filter"))
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

	if len(m.rows) == 0 && m.filterText != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(subtle).Render("  No matching rows"))
	} else {
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
	}

	// Pagination footer
	b.WriteString("\n")
	totalPages := 1
	if m.totalRows > 0 {
		totalPages = int((m.totalRows + int64(m.pageSize) - 1) / int64(m.pageSize))
	}
	pageInfo := fmt.Sprintf(" %d rows", len(m.rows))
	if m.totalRows > 0 && m.filterText == "" {
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
		} else if ci < len(cells) && cells[ci] == "<NULL>" {
			cell = lipgloss.NewStyle().Foreground(m.colors.Subtle).Italic(true).Render(cell)
		}

		parts = append(parts, cell)
	}
	sep := "│"
	if !isHeader {
		sep = lipgloss.NewStyle().Foreground(m.colors.Border).Render("│")
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
	// header(1) + filter(1) + col headers(1) + separator(1) + footer(1) + border(2)
	rows := m.height - 8
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
	if m.cursorCol > m.scrollCol+5 {
		m.scrollCol = m.cursorCol - 3
	}
}

func calculateColWidths(columns []string, rows [][]string) []int {
	if len(columns) == 0 {
		return nil
	}

	widths := make([]int, len(columns))

	for i, col := range columns {
		widths[i] = runeWidth(col)
	}

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
