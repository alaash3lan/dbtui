package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alaa/dbplus/internal/database"
	"github.com/alaa/dbplus/internal/tui/components/dataview"
	"github.com/alaa/dbplus/internal/tui/components/sidebar"
	"github.com/alaa/dbplus/internal/tui/components/statusbar"
	"github.com/alaa/dbplus/internal/tui/components/titlebar"
)

// FocusedPane tracks which pane has keyboard focus.
type FocusedPane int

const (
	PaneSidebar FocusedPane = iota
	PaneDataView
	PaneQueryEditor
)

const totalPanes = 2 // sidebar + dataview (editor added in Sprint 4)

// Model is the root Bubble Tea model.
type Model struct {
	db       *database.DB
	keyMap   KeyMap
	styles   Styles
	version  string

	// Components
	sidebar   sidebar.Model
	dataView  dataview.Model
	titleBar  titlebar.Model
	statusBar statusbar.Model

	// Layout
	width        int
	height       int
	sidebarRatio float64
	focused      FocusedPane

	// State
	tables       []database.TableInfo
	currentTable string
	ready        bool
	err          error
}

// New creates the root model.
func New(db *database.DB, version string) Model {
	return Model{
		db:           db,
		keyMap:       DefaultKeyMap(),
		styles:       DefaultStyles(),
		version:      version,
		sidebar:      sidebar.New(db.DatabaseName()),
		dataView:     dataview.New(),
		titleBar:     titlebar.New(version),
		statusBar:    statusbar.New(db.DatabaseName(), db.User(), db.Host()),
		sidebarRatio: 0.20,
		focused:      PaneSidebar,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return m.fetchTableListCmd()
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateLayout()
		m.ready = true

	case tea.KeyMsg:
		// Global keys handled first
		switch {
		case key.Matches(msg, m.keyMap.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keyMap.FocusNext):
			m.cycleFocus(1)
			m.updateFocus()
			return m, nil
		case key.Matches(msg, m.keyMap.FocusPrev):
			m.cycleFocus(-1)
			m.updateFocus()
			return m, nil
		case key.Matches(msg, m.keyMap.GrowSidebar):
			m.sidebarRatio = min64(m.sidebarRatio+0.02, 0.40)
			m.updateLayout()
			return m, nil
		case key.Matches(msg, m.keyMap.ShrinkSidebar):
			m.sidebarRatio = max64(m.sidebarRatio-0.02, 0.10)
			m.updateLayout()
			return m, nil
		}

	case TableListMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.tables = msg.Tables
		m.sidebar.SetTables(msg.Tables)

	case sidebar.TableSelectedMsg:
		m.currentTable = msg.TableName
		return m, tea.Batch(
			m.fetchTableDataCmd(msg.TableName),
			m.fetchCountCmd(msg.TableName),
			m.fetchSchemaCmd(msg.TableName),
		)

	case QueryResultMsg:
		if msg.Err != nil {
			m.err = msg.Err
			return m, nil
		}
		m.err = nil
		m.dataView.SetData(m.currentTable, msg.Columns, msg.Rows)
		m.titleBar.SetRowCount(msg.RowCount)
		m.statusBar.SetQueryInfo(msg.Duration, msg.RowCount)

	case tableCountMsg:
		m.dataView.SetPage(0, msg.count)

	case SchemaInfoMsg:
		if msg.Err == nil {
			m.sidebar.SetSchemaInfo(msg.Info)
		}
	}

	// Route to focused component
	switch m.focused {
	case PaneSidebar:
		var cmd tea.Cmd
		m.sidebar, cmd = m.sidebar.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	case PaneDataView:
		var cmd tea.Cmd
		m.dataView, cmd = m.dataView.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	// Title bar
	titleView := m.titleBar.View()

	// Status bar
	statusView := m.statusBar.View()

	// Available height for middle section
	availHeight := m.height - 2

	// Sidebar
	sidebarWidth := int(float64(m.width) * m.sidebarRatio)
	mainWidth := m.width - sidebarWidth

	sidebarView := m.sidebar.View()
	dataViewView := m.dataView.View()

	// Error overlay in data area
	if m.err != nil {
		errStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#F38BA8")).
			Width(mainWidth - 2).
			Height(availHeight - 2).
			Padding(1)
		dataViewView = errStyle.Render(
			m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)),
		)
	}

	// Compose layout
	middleRow := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, dataViewView)

	return lipgloss.JoinVertical(lipgloss.Left, titleView, middleRow, statusView)
}

func (m *Model) updateLayout() {
	sidebarWidth := int(float64(m.width) * m.sidebarRatio)
	mainWidth := m.width - sidebarWidth
	availHeight := m.height - 2

	m.sidebar.SetSize(sidebarWidth, availHeight)
	m.dataView.SetSize(mainWidth, availHeight)
	m.updateFocus()
	m.titleBar.SetWidth(m.width)
	m.statusBar.SetWidth(m.width)
}

func (m *Model) updateFocus() {
	m.sidebar.SetFocused(m.focused == PaneSidebar)
	m.dataView.SetFocused(m.focused == PaneDataView)
}

func (m *Model) cycleFocus(dir int) {
	next := (int(m.focused) + dir + totalPanes) % totalPanes
	m.focused = FocusedPane(next)
}

// tea.Cmd factories

func (m Model) fetchTableListCmd() tea.Cmd {
	return func() tea.Msg {
		tables, err := m.db.ListTables()
		return TableListMsg{Tables: tables, Err: err}
	}
}

func (m Model) fetchTableDataCmd(table string) tea.Cmd {
	db := m.db
	pageSize := m.dataView.PageSize()
	return func() tea.Msg {
		result, err := db.FetchTableData(table, pageSize, 0)
		if err != nil {
			return QueryResultMsg{Err: err}
		}
		return QueryResultMsg{
			Columns:  result.Columns,
			Rows:     result.Rows,
			RowCount: result.RowCount,
			Duration: result.Duration,
			IsSelect: true,
		}
	}
}

type tableCountMsg struct {
	count int64
}

func (m Model) fetchCountCmd(table string) tea.Cmd {
	db := m.db
	return func() tea.Msg {
		count, err := db.CountRows(table)
		if err != nil {
			return tableCountMsg{count: 0}
		}
		return tableCountMsg{count: count}
	}
}

func (m Model) fetchSchemaCmd(table string) tea.Cmd {
	db := m.db
	return func() tea.Msg {
		info, err := db.DescribeTable(table)
		return SchemaInfoMsg{Info: info, Err: err}
	}
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
