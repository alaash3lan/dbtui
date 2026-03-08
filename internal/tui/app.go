package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/alaa/dbtui/internal/database"
	"github.com/alaa/dbtui/internal/tui/components/dataview"
	"github.com/alaa/dbtui/internal/tui/components/editor"
	"github.com/alaa/dbtui/internal/tui/components/sidebar"
	"github.com/alaa/dbtui/internal/tui/components/statusbar"
	"github.com/alaa/dbtui/internal/tui/components/titlebar"
)

// FocusedPane tracks which pane has keyboard focus.
type FocusedPane int

const (
	PaneSidebar FocusedPane = iota
	PaneDataView
	PaneQueryEditor
)

const totalPanes = 3

// editorHeightRatio is the fraction of the right pane given to the editor.
const editorHeightRatio = 0.30

// Model is the root Bubble Tea model.
type Model struct {
	db           *database.DB
	keyMap       KeyMap
	styles       Styles
	theme        Theme
	version      string
	queryTimeout time.Duration

	// Components
	sidebar   sidebar.Model
	dataView  dataview.Model
	editor    editor.Model
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
	showHelp     bool
	cancelQuery  context.CancelFunc
	queryRunning bool
}

// New creates the root model.
func New(db *database.DB, version string, queryTimeout time.Duration, pageSize int) Model {
	m := Model{
		db:           db,
		keyMap:       DefaultKeyMap(),
		styles:       DefaultStyles(),
		theme:        DarkTheme(),
		version:      version,
		queryTimeout: queryTimeout,
		sidebar:      sidebar.New(db.DatabaseName()),
		dataView:     dataview.New(pageSize),
		editor:       editor.New(),
		titleBar:     titlebar.New(version),
		statusBar:    statusbar.New(db.DatabaseName(), db.User(), db.Host()),
		sidebarRatio: 0.20,
		focused:      PaneSidebar,
	}
	m.applyTheme()
	return m
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
			if m.queryRunning && m.cancelQuery != nil {
				m.cancelQuery()
				m.cancelQuery = nil
				return m, nil
			}
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
		case key.Matches(msg, m.keyMap.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, m.keyMap.ToggleTheme):
			if m.theme.Name == "dark" {
				m.theme = LightTheme()
			} else {
				m.theme = DarkTheme()
			}
			m.applyTheme()
			return m, nil
		case key.Matches(msg, m.keyMap.Refresh):
			batch := []tea.Cmd{m.fetchTableListCmd()}
			if m.currentTable != "" {
				batch = append(batch, m.fetchTableDataCmd(m.currentTable), m.fetchCountCmd(m.currentTable))
			}
			return m, tea.Batch(batch...)
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
		m.err = nil
		m.editor.ClearStatus()
		return m, tea.Batch(
			m.fetchTableDataCmd(msg.TableName),
			m.fetchCountCmd(msg.TableName),
			m.fetchSchemaCmd(msg.TableName),
		)

	case sidebar.SchemaRequestMsg:
		return m, m.fetchSchemaCmd(msg.TableName)

	case editor.ExecuteQueryMsg:
		m.editor.SetRunning(true)
		m.queryRunning = true
		m.err = nil
		return m, m.executeQueryCmd(msg.SQL)

	case QueryResultMsg:
		m.editor.SetRunning(false)
		m.queryRunning = false
		m.cancelQuery = nil
		if msg.Err != nil {
			m.editor.SetError(msg.Err.Error())
			return m, nil
		}
		m.err = nil
		if msg.IsSelect {
			tableName := m.currentTable
			if tableName == "" {
				tableName = "query"
			}
			m.dataView.SetData(tableName, msg.Columns, msg.Rows)
			m.titleBar.SetRowCount(msg.RowCount)
			m.statusBar.SetQueryInfo(msg.Duration, msg.RowCount)
			m.editor.SetResult(fmt.Sprintf("%d rows in %s", msg.RowCount, msg.Duration))
		} else {
			// Handle database switch
			if msg.DatabaseChanged != "" {
				m.sidebar.SetDBName(msg.DatabaseChanged)
				m.statusBar.SetDBName(msg.DatabaseChanged)
				m.currentTable = ""
				m.dataView.SetData("", nil, nil)
				m.editor.SetResult(fmt.Sprintf("Database changed to %s", msg.DatabaseChanged))
				return m, m.fetchTableListCmd()
			}
			m.editor.SetResult(fmt.Sprintf("%d rows affected in %s", msg.AffectedRows, msg.Duration))
			m.statusBar.SetQueryInfo(msg.Duration, int(msg.AffectedRows))
			// Refresh table list (handles CREATE/DROP TABLE) and current table data
			batch := []tea.Cmd{m.fetchTableListCmd()}
			if m.currentTable != "" {
				batch = append(batch, m.fetchTableDataCmd(m.currentTable), m.fetchCountCmd(m.currentTable))
			}
			return m, tea.Batch(batch...)
		}

	case dataview.PageRequestMsg:
		return m, m.fetchPageCmd(msg.Table, msg.Page, msg.Offset, msg.Limit)

	case pageDataMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.dataView.SetData(msg.table, msg.columns, msg.rows)
		m.dataView.SetPageDirect(msg.page)
		m.titleBar.SetRowCount(len(msg.rows))
		return m, nil

	case tableCountMsg:
		m.dataView.SetTotalRows(msg.count)

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
	case PaneQueryEditor:
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
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

	// Minimum terminal size
	if m.width < 60 || m.height < 16 {
		return lipgloss.NewStyle().
			Foreground(m.theme.WarningColor).
			Bold(true).
			Render(fmt.Sprintf("Terminal too small (%dx%d). Minimum: 60x16", m.width, m.height))
	}

	// Help overlay
	if m.showHelp {
		return m.renderHelp()
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

	// Right pane: DataView (top) + Editor (bottom)
	editorHeight := int(float64(availHeight) * editorHeightRatio)
	if editorHeight < 6 {
		editorHeight = 6
	}
	dataViewHeight := availHeight - editorHeight

	dataViewView := m.dataView.View()
	editorView := m.editor.View()

	// Error overlay in data area
	if m.err != nil {
		errStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(m.theme.ErrorColor).
			Width(mainWidth - 2).
			Height(dataViewHeight - 2).
			Padding(1)
		dataViewView = errStyle.Render(
			m.styles.Error.Render(fmt.Sprintf("Error: %v", m.err)),
		)
	}

	rightPane := lipgloss.JoinVertical(lipgloss.Left, dataViewView, editorView)

	// Compose layout
	middleRow := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, rightPane)

	return lipgloss.JoinVertical(lipgloss.Left, titleView, middleRow, statusView)
}

func (m *Model) updateLayout() {
	sidebarWidth := int(float64(m.width) * m.sidebarRatio)
	mainWidth := m.width - sidebarWidth
	availHeight := m.height - 2

	editorHeight := int(float64(availHeight) * editorHeightRatio)
	if editorHeight < 6 {
		editorHeight = 6
	}
	dataViewHeight := availHeight - editorHeight

	m.sidebar.SetSize(sidebarWidth, availHeight)
	m.dataView.SetSize(mainWidth, dataViewHeight)
	m.editor.SetSize(mainWidth, editorHeight)
	m.updateFocus()
	m.titleBar.SetWidth(m.width)
	m.statusBar.SetWidth(m.width)
}

func (m *Model) updateFocus() {
	m.sidebar.SetFocused(m.focused == PaneSidebar)
	m.dataView.SetFocused(m.focused == PaneDataView)
	m.editor.SetFocused(m.focused == PaneQueryEditor)
}

func (m *Model) cycleFocus(dir int) {
	next := (int(m.focused) + dir + totalPanes) % totalPanes
	m.focused = FocusedPane(next)
}

// tea.Cmd factories

func (m Model) fetchTableListCmd() tea.Cmd {
	db := m.db
	return func() tea.Msg {
		tables, err := db.ListTables(context.Background())
		return TableListMsg{Tables: tables, Err: err}
	}
}

func (m Model) fetchTableDataCmd(table string) tea.Cmd {
	db := m.db
	pageSize := m.dataView.PageSize()
	return func() tea.Msg {
		result, err := db.FetchTableData(context.Background(), table, pageSize, 0)
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

func (m *Model) executeQueryCmd(sql string) tea.Cmd {
	db := m.db
	timeout := m.queryTimeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	m.cancelQuery = cancel
	return func() tea.Msg {
		defer cancel()
		result, err := db.Execute(ctx, sql)
		if err != nil {
			return QueryResultMsg{Err: err}
		}
		return QueryResultMsg{
			Columns:         result.Columns,
			Rows:            result.Rows,
			RowCount:        result.RowCount,
			AffectedRows:    result.AffectedRows,
			Duration:        result.Duration,
			IsSelect:        result.IsSelect,
			DatabaseChanged: result.DatabaseChanged,
		}
	}
}

type pageDataMsg struct {
	table   string
	page    int
	columns []string
	rows    [][]string
	err     error
}

type tableCountMsg struct {
	count int64
}

func (m Model) fetchPageCmd(table string, page, offset, limit int) tea.Cmd {
	db := m.db
	return func() tea.Msg {
		result, err := db.FetchTableData(context.Background(), table, limit, offset)
		if err != nil {
			return pageDataMsg{table: table, page: page, err: err}
		}
		return pageDataMsg{
			table:   table,
			page:    page,
			columns: result.Columns,
			rows:    result.Rows,
		}
	}
}

func (m Model) fetchCountCmd(table string) tea.Cmd {
	db := m.db
	return func() tea.Msg {
		count, err := db.CountRows(context.Background(), table)
		if err != nil {
			return tableCountMsg{count: 0}
		}
		return tableCountMsg{count: count}
	}
}

func (m Model) fetchSchemaCmd(table string) tea.Cmd {
	db := m.db
	return func() tea.Msg {
		info, err := db.DescribeTable(context.Background(), table)
		return SchemaInfoMsg{Info: info, Err: err}
	}
}

func (m Model) renderHelp() string {
	t := m.theme
	title := lipgloss.NewStyle().Foreground(t.Highlight).Bold(true)
	keyStyle := lipgloss.NewStyle().Foreground(t.WarningColor).Bold(true)
	desc := lipgloss.NewStyle().Foreground(t.Text)
	dim := lipgloss.NewStyle().Foreground(t.Subtle)

	help := title.Render("dbplus Keyboard Shortcuts") + "\n\n"

	help += title.Render("Global") + "\n"
	help += keyStyle.Render("  Ctrl+C      ") + desc.Render("Quit") + "\n"
	help += keyStyle.Render("  Tab         ") + desc.Render("Next pane") + "\n"
	help += keyStyle.Render("  Shift+Tab   ") + desc.Render("Previous pane") + "\n"
	help += keyStyle.Render("  Ctrl+Left   ") + desc.Render("Shrink sidebar") + "\n"
	help += keyStyle.Render("  Ctrl+Right  ") + desc.Render("Grow sidebar") + "\n"
	help += keyStyle.Render("  Ctrl+T      ") + desc.Render("Toggle dark/light theme") + "\n"
	help += keyStyle.Render("  Ctrl+R      ") + desc.Render("Refresh tables & data") + "\n"
	help += keyStyle.Render("  F1          ") + desc.Render("Toggle this help") + "\n\n"

	help += title.Render("Sidebar") + "\n"
	help += keyStyle.Render("  j/k arrows  ") + desc.Render("Navigate tables") + "\n"
	help += keyStyle.Render("  Enter       ") + desc.Render("Select table, load data") + "\n"
	help += keyStyle.Render("  i           ") + desc.Render("Toggle schema info") + "\n"
	help += keyStyle.Render("  g/G         ") + desc.Render("First/last table") + "\n\n"

	help += title.Render("Data View") + "\n"
	help += keyStyle.Render("  arrows/hjkl ") + desc.Render("Scroll grid") + "\n"
	help += keyStyle.Render("  / Ctrl+F    ") + desc.Render("Activate filter") + "\n"
	help += keyStyle.Render("  Escape      ") + desc.Render("Clear filter") + "\n"
	help += keyStyle.Render("  PgUp/PgDn   ") + desc.Render("Scroll viewport up/down") + "\n"
	help += keyStyle.Render("  n/p         ") + desc.Render("Next/prev server page") + "\n"
	help += keyStyle.Render("  Home/End    ") + desc.Render("First/last row") + "\n\n"

	help += title.Render("Query Editor") + "\n"
	help += keyStyle.Render("  Enter       ") + desc.Render("Execute (requires ;) or newline") + "\n"
	help += keyStyle.Render("  Ctrl+E      ") + desc.Render("Force execute") + "\n"
	help += keyStyle.Render("  Up/Down     ") + desc.Render("Navigate history") + "\n"
	help += keyStyle.Render("  Escape      ") + desc.Render("Clear input") + "\n\n"

	help += title.Render("Tips") + "\n"
	help += desc.Render("  Type USE dbname; to switch databases") + "\n"
	help += desc.Render("  Ctrl+C cancels running query, press again to quit") + "\n\n"

	help += dim.Render("Press F1 to close")

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Highlight).
		Padding(1, 2).
		Width(50)

	// Center in terminal
	rendered := box.Render(help)
	hPad := (m.width - lipgloss.Width(rendered)) / 2
	vPad := (m.height - lipgloss.Height(rendered)) / 2
	if hPad < 0 {
		hPad = 0
	}
	if vPad < 0 {
		vPad = 0
	}

	var padded string
	for i := 0; i < vPad; i++ {
		padded += "\n"
	}
	lines := strings.Split(rendered, "\n")
	for _, line := range lines {
		for i := 0; i < hPad; i++ {
			padded += " "
		}
		padded += line + "\n"
	}

	return padded
}

func (m *Model) applyTheme() {
	t := m.theme
	m.sidebar.SetColors(sidebar.Colors{
		Highlight:   t.Highlight,
		Subtle:      t.Subtle,
		Border:      t.Border,
		FocusBorder: t.FocusBorder,
		ActiveBg:    t.ActiveBg,
	})
	m.dataView.SetColors(dataview.Colors{
		Highlight:    t.Highlight,
		Subtle:       t.Subtle,
		Border:       t.Border,
		FocusBorder:  t.FocusBorder,
		SelectedBg:   t.SelectedBg,
		WarningColor: t.WarningColor,
	})
	m.editor.SetColors(editor.Colors{
		Highlight:    t.Highlight,
		Subtle:       t.Subtle,
		Border:       t.Border,
		FocusBorder:  t.FocusBorder,
		ErrorColor:   t.ErrorColor,
		SuccessColor: t.SuccessColor,
		WarningColor: t.WarningColor,
	})
	m.titleBar.SetColors(titlebar.Colors{
		Highlight:  t.Highlight,
		Text:       t.Text,
		Background: t.HeaderBg,
	})
	m.statusBar.SetColors(statusbar.Colors{
		Highlight:  t.Highlight,
		Text:       t.Text,
		Background: t.HeaderBg,
	})
	m.styles.Error = lipgloss.NewStyle().Foreground(t.ErrorColor).Bold(true)
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
