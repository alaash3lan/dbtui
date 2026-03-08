package dataview

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alaa/dbtui/internal/database"
	"github.com/alaa/dbtui/internal/stringutil"
)

// DeleteRowRequestMsg is emitted when the user requests to delete the current row.
type DeleteRowRequestMsg struct {
	Table      string
	PKColumns  []string
	PKValues   []string
	RowPreview string
}

// PageRequestMsg is emitted when the user navigates past the loaded page.
type PageRequestMsg struct {
	Table  string
	Page   int
	Offset int
	Limit  int
}

// CopyToClipboardMsg is emitted when the user copies a cell or row value.
type CopyToClipboardMsg struct {
	Text string
}

// KeyMap defines dataview-specific keybindings.
type KeyMap struct {
	Up          key.Binding
	Down        key.Binding
	Left        key.Binding
	Right       key.Binding
	PageDown    key.Binding
	PageUp      key.Binding
	Home        key.Binding
	End         key.Binding
	Filter      key.Binding
	ClearFilter key.Binding
	NextPage    key.Binding
	PrevPage    key.Binding
	CopyCell         key.Binding
	CopyRow          key.Binding
	Detail           key.Binding
	Sort             key.Binding
	ToggleRowNumbers key.Binding
	DeleteRow        key.Binding
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
		NextPage: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next page"),
		),
		PrevPage: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "prev page"),
		),
		CopyCell: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "copy cell"),
		),
		CopyRow: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy row"),
		),
		Detail: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "row detail"),
		),
		Sort: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort column"),
		),
		ToggleRowNumbers: key.NewBinding(
			key.WithKeys("ctrl+n"),
			key.WithHelp("ctrl+n", "toggle row numbers"),
		),
		DeleteRow: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "delete row"),
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

	// Sorting
	sortCol int // -1 = no sort
	sortDir int // 0=none, 1=asc, 2=desc

	// Row numbers
	showRowNumbers bool

	// Row detail mode
	detailMode   bool
	detailScroll int

	schemaInfo *database.SchemaInfo

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

// New creates a new data view model with the given page size.
func New(pageSize int) Model {
	fi := textinput.New()
	fi.Placeholder = "column | value  or  value"
	fi.Prompt = "Filter: "
	fi.CharLimit = 256

	if pageSize <= 0 {
		pageSize = 100
	}

	return Model{
		pageSize:       pageSize,
		keyMap:         DefaultKeyMap(),
		filterInput:    fi,
		sortCol:        -1,
		showRowNumbers: true,
	}
}

// SetData loads new query results into the grid and resets page to 0.
func (m *Model) SetData(tableName string, columns []string, rows [][]string) {
	m.tableName = tableName
	m.columns = columns
	m.allRows = rows
	m.page = 0
	m.cursorRow = 0
	m.cursorCol = 0
	m.scrollRow = 0
	m.scrollCol = 0
	m.sortCol = -1
	m.sortDir = 0
	m.applySortAndFilter()
	m.colWidths = calculateColWidths(columns, m.rows)
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

// SetTotalRows updates the total row count without changing the current page.
func (m *Model) SetTotalRows(totalRows int64) {
	m.totalRows = totalRows
}

// SetPageDirect sets the page without changing totalRows.
func (m *Model) SetPageDirect(page int) {
	m.page = page
}

// SetSchemaInfo stores the table schema for primary key resolution.
func (m *Model) SetSchemaInfo(info *database.SchemaInfo) {
	m.schemaInfo = info
}

// Columns returns the current column names.
func (m Model) Columns() []string {
	return m.columns
}

// Rows returns the currently displayed rows (after filtering).
func (m Model) Rows() [][]string {
	return m.rows
}

// CursorCellValue returns the value of the cell under the cursor.
func (m Model) CursorCellValue() string {
	if m.cursorRow < 0 || m.cursorRow >= len(m.rows) {
		return ""
	}
	row := m.rows[m.cursorRow]
	if m.cursorCol < 0 || m.cursorCol >= len(row) {
		return ""
	}
	return row[m.cursorCol]
}

// CursorRowValues returns all cell values of the current row as tab-separated text.
func (m Model) CursorRowValues() string {
	if m.cursorRow < 0 || m.cursorRow >= len(m.rows) {
		return ""
	}
	return strings.Join(m.rows[m.cursorRow], "\t")
}

// PageSize returns the configured page size.
func (m Model) PageSize() int {
	return m.pageSize
}

// Page returns the current page number.
func (m Model) Page() int {
	return m.page
}

// TotalPages returns the total number of pages.
func (m Model) TotalPages() int {
	if m.totalRows <= 0 {
		return 1
	}
	return int((m.totalRows + int64(m.pageSize) - 1) / int64(m.pageSize))
}

// TableName returns the current table name.
func (m Model) TableName() string {
	return m.tableName
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

		// Detail mode navigation
		if m.detailMode {
			switch {
			case key.Matches(msg, m.keyMap.Detail), key.Matches(msg, m.keyMap.ClearFilter):
				m.detailMode = false
				m.detailScroll = 0
				return m, nil
			case key.Matches(msg, m.keyMap.Up):
				if m.detailScroll > 0 {
					m.detailScroll--
				}
				return m, nil
			case key.Matches(msg, m.keyMap.Down):
				if m.detailScroll < len(m.columns)-1 {
					m.detailScroll++
				}
				return m, nil
			case key.Matches(msg, m.keyMap.Home):
				m.detailScroll = 0
				return m, nil
			case key.Matches(msg, m.keyMap.End):
				m.detailScroll = len(m.columns) - 1
				return m, nil
			}
			return m, nil
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
			// Request next server page if at end of data and more pages exist
			if m.filterText == "" && m.tableName != "" && len(m.rows) > 0 &&
				m.cursorRow == len(m.rows)-1 && m.page+1 < m.TotalPages() {
				nextPage := m.page + 1
				return m, func() tea.Msg {
					return PageRequestMsg{
						Table:  m.tableName,
						Page:   nextPage,
						Offset: nextPage * m.pageSize,
						Limit:  m.pageSize,
					}
				}
			}
		case key.Matches(msg, m.keyMap.PageUp):
			viewportRows := m.viewportRows()
			m.cursorRow -= viewportRows
			if m.cursorRow < 0 {
				m.cursorRow = 0
			}
			m.adjustVerticalScroll()
			// Request previous server page if at start and on a later page
			if m.filterText == "" && m.tableName != "" && m.cursorRow == 0 && m.page > 0 {
				prevPage := m.page - 1
				return m, func() tea.Msg {
					return PageRequestMsg{
						Table:  m.tableName,
						Page:   prevPage,
						Offset: prevPage * m.pageSize,
						Limit:  m.pageSize,
					}
				}
			}
		case key.Matches(msg, m.keyMap.NextPage):
			if m.filterText == "" && m.tableName != "" && m.page+1 < m.TotalPages() {
				nextPage := m.page + 1
				return m, func() tea.Msg {
					return PageRequestMsg{
						Table:  m.tableName,
						Page:   nextPage,
						Offset: nextPage * m.pageSize,
						Limit:  m.pageSize,
					}
				}
			}
		case key.Matches(msg, m.keyMap.PrevPage):
			if m.filterText == "" && m.tableName != "" && m.page > 0 {
				prevPage := m.page - 1
				return m, func() tea.Msg {
					return PageRequestMsg{
						Table:  m.tableName,
						Page:   prevPage,
						Offset: prevPage * m.pageSize,
						Limit:  m.pageSize,
					}
				}
			}
		case key.Matches(msg, m.keyMap.CopyCell):
			text := m.CursorCellValue()
			if text != "" {
				return m, func() tea.Msg {
					return CopyToClipboardMsg{Text: text}
				}
			}
		case key.Matches(msg, m.keyMap.CopyRow):
			text := m.CursorRowValues()
			if text != "" {
				return m, func() tea.Msg {
					return CopyToClipboardMsg{Text: text}
				}
			}
		case key.Matches(msg, m.keyMap.Detail):
			if len(m.rows) > 0 {
				m.detailMode = true
				m.detailScroll = 0
			}
		case key.Matches(msg, m.keyMap.Sort):
			if len(m.columns) > 0 {
				if m.sortCol == m.cursorCol {
					// Cycle: none -> asc -> desc -> none
					m.sortDir = (m.sortDir + 1) % 3
					if m.sortDir == 0 {
						m.sortCol = -1
					}
				} else {
					m.sortCol = m.cursorCol
					m.sortDir = 1
				}
				m.applySortAndFilter()
			}
		case key.Matches(msg, m.keyMap.ToggleRowNumbers):
			m.showRowNumbers = !m.showRowNumbers
		case key.Matches(msg, m.keyMap.DeleteRow):
			if len(m.rows) == 0 || m.cursorRow < 0 || m.cursorRow >= len(m.rows) {
				return m, nil
			}
			if m.schemaInfo == nil || m.tableName == "" {
				return m, nil
			}
			// Find primary key columns
			var pkCols []string
			for _, col := range m.schemaInfo.Columns {
				if col.Key == "PRI" {
					pkCols = append(pkCols, col.Name)
				}
			}
			if len(pkCols) == 0 {
				return m, nil
			}
			// Resolve PK values from the current row
			row := m.rows[m.cursorRow]
			var pkValues []string
			for _, pkCol := range pkCols {
				found := false
				for ci, colName := range m.columns {
					if colName == pkCol && ci < len(row) {
						pkValues = append(pkValues, row[ci])
						found = true
						break
					}
				}
				if !found {
					return m, nil
				}
			}
			// Build row preview
			var preview []string
			for ci, colName := range m.columns {
				if ci < len(row) {
					preview = append(preview, fmt.Sprintf("%s: %s", colName, row[ci]))
				}
			}
			tableName := m.tableName
			rowPreview := strings.Join(preview, "\n")
			return m, func() tea.Msg {
				return DeleteRowRequestMsg{
					Table:      tableName,
					PKColumns:  pkCols,
					PKValues:   pkValues,
					RowPreview: rowPreview,
				}
			}
		}
	}

	return m, nil
}

func (m *Model) applyFilter() {
	m.applySortAndFilter()
}

func (m *Model) applySortAndFilter() {
	// Step 1: Apply filter
	f := ParseFilter(m.filterText)
	if f.Value == "" {
		m.rows = make([][]string, len(m.allRows))
		copy(m.rows, m.allRows)
	} else {
		m.rows = ApplyFilter(m.columns, m.allRows, f)
	}

	// Step 2: Apply sort
	if m.sortCol >= 0 && m.sortCol < len(m.columns) && m.sortDir > 0 {
		asc := m.sortDir == 1
		col := m.sortCol
		sort.SliceStable(m.rows, func(i, j int) bool {
			a, b := "", ""
			if col < len(m.rows[i]) {
				a = m.rows[i][col]
			}
			if col < len(m.rows[j]) {
				b = m.rows[j][col]
			}

			// Try numeric comparison first
			af, aErr := strconv.ParseFloat(a, 64)
			bf, bErr := strconv.ParseFloat(b, 64)
			if aErr == nil && bErr == nil {
				if asc {
					return af < bf
				}
				return af > bf
			}

			// Fall back to case-insensitive string comparison
			cmp := strings.Compare(strings.ToLower(a), strings.ToLower(b))
			if asc {
				return cmp < 0
			}
			return cmp > 0
		})
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

	// Row detail mode
	if m.detailMode {
		return m.renderDetailView(contentWidth, border, focusBorder)
	}

	// Row number column width
	rowNumWidth := 0
	if m.showRowNumbers {
		rowNumWidth = m.rowNumberWidth()
	}

	// Calculate visible columns (account for row number column)
	visibleCols := m.visibleColumns(contentWidth - rowNumWidth)

	// Column headers
	var headerLine string
	if m.showRowNumbers {
		cell := lipgloss.NewStyle().Foreground(subtle).Render(
			" " + m.padRight("#", rowNumWidth-3) + " ",
		)
		headerLine = cell + lipgloss.NewStyle().Foreground(border).Render("│")
	}
	headerLine += m.renderRow(m.columnsWithSortIndicator(), visibleCols, -1, selectedBg, highlight, true)
	b.WriteString(headerLine)
	b.WriteString("\n")

	// Separator
	sepParts := make([]string, 0)
	if m.showRowNumbers {
		sepParts = append(sepParts, strings.Repeat("─", rowNumWidth-1))
	}
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
			var line string
			if m.showRowNumbers {
				num := strconv.Itoa(i + 1)
				padded := m.padLeft(num, rowNumWidth-3)
				cell := " " + padded + " "
				if i == m.cursorRow && m.focused {
					cell = lipgloss.NewStyle().Background(selectedBg).Foreground(subtle).Render(cell)
				} else {
					cell = lipgloss.NewStyle().Foreground(subtle).Render(cell)
				}
				line = cell + lipgloss.NewStyle().Foreground(border).Render("│")
			}
			line += m.renderRow(m.rows[i], visibleCols, i, selectedBg, highlight, false)
			b.WriteString(line)
			if i < endRow-1 {
				b.WriteString("\n")
			}
		}
	}

	// Pagination footer
	b.WriteString("\n")
	footerStyle := lipgloss.NewStyle().Foreground(subtle)
	var pageInfo string
	if m.filterText != "" {
		pageInfo = fmt.Sprintf(" %d/%d matched (filtered from page %d, n/p for pages)", len(m.rows), len(m.allRows), m.page+1)
	} else if m.totalRows > 0 {
		totalPages := m.TotalPages()
		startRow := m.page*m.pageSize + 1
		endRow := startRow + len(m.rows) - 1
		if endRow < startRow {
			endRow = startRow
		}
		pageInfo = fmt.Sprintf(" rows %d-%d of %d  page %d/%d  (n/p)", startRow, endRow, m.totalRows, m.page+1, totalPages)
	} else {
		pageInfo = fmt.Sprintf(" %d rows", len(m.rows))
	}

	padLen := contentWidth - lipgloss.Width(pageInfo)
	if padLen < 0 {
		padLen = 0
	}
	b.WriteString(footerStyle.Render(strings.Repeat(" ", padLen) + pageInfo))

	return m.applyBorder(b.String(), contentWidth, border, focusBorder)
}

func (m Model) renderDetailView(contentWidth int, border, focusBorder lipgloss.Color) string {
	c := m.colors
	highlight := c.Highlight
	subtle := c.Subtle

	var b strings.Builder

	// Header with row index
	headerStyle := lipgloss.NewStyle().Foreground(highlight).Bold(true)
	b.WriteString(headerStyle.Render(fmt.Sprintf(" Row Detail (%d/%d)", m.cursorRow+1, len(m.rows))))
	b.WriteString("\n")

	// Separator
	b.WriteString(lipgloss.NewStyle().Foreground(subtle).Render(strings.Repeat("─", contentWidth)))
	b.WriteString("\n")

	// Calculate max column name width for alignment
	maxColWidth := 0
	for _, col := range m.columns {
		w := stringutil.RuneWidth(col)
		if w > maxColWidth {
			maxColWidth = w
		}
	}
	// Cap label width to leave room for values
	if maxColWidth > contentWidth/3 {
		maxColWidth = contentWidth / 3
	}

	row := m.rows[m.cursorRow]
	colStyle := lipgloss.NewStyle().Foreground(highlight).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(subtle).Italic(true)

	// Available lines for detail rows: height - header(1) - separator(1) - footer(1) - border(2)
	visibleLines := m.height - 5
	if visibleLines < 1 {
		visibleLines = 1
	}

	// Clamp scroll
	if m.detailScroll > len(m.columns)-visibleLines {
		maxScroll := len(m.columns) - visibleLines
		if maxScroll < 0 {
			maxScroll = 0
		}
		m.detailScroll = maxScroll
	}

	endIdx := m.detailScroll + visibleLines
	if endIdx > len(m.columns) {
		endIdx = len(m.columns)
	}

	for i := m.detailScroll; i < endIdx; i++ {
		colName := stringutil.Truncate(m.columns[i], maxColWidth)
		padding := strings.Repeat(" ", maxColWidth-stringutil.RuneWidth(colName))

		val := ""
		if i < len(row) {
			val = row[i]
		}

		// Truncate value to fit remaining width
		valMaxWidth := contentWidth - maxColWidth - 4 // " : " separator + leading space
		if valMaxWidth < 1 {
			valMaxWidth = 1
		}
		val = stringutil.Truncate(val, valMaxWidth)

		label := colStyle.Render(" " + colName + padding)
		separator := lipgloss.NewStyle().Foreground(subtle).Render(" : ")

		if val == "<NULL>" {
			b.WriteString(label + separator + dimStyle.Render(val))
		} else {
			b.WriteString(label + separator + val)
		}
		if i < endIdx-1 {
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	footerStyle := lipgloss.NewStyle().Foreground(subtle)
	hint := fmt.Sprintf(" %d columns  (d/Esc to close)", len(m.columns))
	b.WriteString(footerStyle.Render(hint))

	return m.applyBorder(b.String(), contentWidth, border, focusBorder)
}

func (m Model) renderRow(cells []string, visibleCols []int, rowIdx int, selectedBg, highlight lipgloss.Color, isHeader bool) string {
	var parts []string
	for _, ci := range visibleCols {
		val := ""
		if ci < len(cells) {
			val = stringutil.Truncate(cells[ci], m.colWidths[ci])
		}
		// Pad to column width
		padded := val + strings.Repeat(" ", m.colWidths[ci]-stringutil.RuneWidth(val))
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

func (m Model) rowNumberWidth() int {
	n := len(m.rows)
	if n == 0 {
		n = 1
	}
	digits := len(strconv.Itoa(n))
	if digits < 1 {
		digits = 1
	}
	// " " + digits + " " + "│" = digits + 3
	return digits + 3
}

func (m Model) padLeft(s string, width int) string {
	gap := width - stringutil.RuneWidth(s)
	if gap <= 0 {
		return s
	}
	return strings.Repeat(" ", gap) + s
}

func (m Model) padRight(s string, width int) string {
	gap := width - stringutil.RuneWidth(s)
	if gap <= 0 {
		return s
	}
	return s + strings.Repeat(" ", gap)
}

func (m Model) columnsWithSortIndicator() []string {
	cols := make([]string, len(m.columns))
	copy(cols, m.columns)
	if m.sortCol >= 0 && m.sortCol < len(cols) && m.sortDir > 0 {
		indicator := " ▲"
		if m.sortDir == 2 {
			indicator = " ▼"
		}
		cols[m.sortCol] = cols[m.sortCol] + indicator
	}
	return cols
}

func calculateColWidths(columns []string, rows [][]string) []int {
	if len(columns) == 0 {
		return nil
	}

	widths := make([]int, len(columns))

	for i, col := range columns {
		widths[i] = stringutil.RuneWidth(col)
	}

	checkRows := len(rows)
	if checkRows > 50 {
		checkRows = 50
	}
	for _, row := range rows[:checkRows] {
		for i, cell := range row {
			if i < len(widths) {
				w := stringutil.RuneWidth(cell)
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

